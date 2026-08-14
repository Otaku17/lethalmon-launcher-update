//go:build windows

package backend

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This file implements the Authenticode check guarding the Visual C++
// Redistributable auto-install (see ensureVCRedist in app_vcredist_windows.go)
// by calling Windows' own WinVerifyTrust, rather than shelling out to
// PowerShell's Get-AuthenticodeSignature.
//
// Going through powershell.exe was a liability on a security-critical path:
//
//   - It only works if PowerShell does. AppLocker, Constrained Language Mode,
//     an execution policy locked down by group policy, or PowerShell having
//     been removed all turn the check into a hard failure — and since the
//     check is (correctly) fail-closed, the redistributable then never
//     installs on those machines, which is exactly the situation this whole
//     feature exists to avoid.
//   - A GUI app spawning a hidden PowerShell that inspects and then silently
//     runs a downloaded executable is the behavioral pattern antivirus
//     heuristics flag. That's the same reasoning that already ruled out a
//     .bat/cmd.exe helper for the self-update (see installLauncherUpdate).
//   - The verdict came back as a process exit code assembled by string
//     concatenation into a script. Nothing was wrong with the escaping, but
//     "may this run with admin rights" deserves a narrower channel than
//     whatever powershell.exe happened to exit with.
//
// A signature can reach a file two different ways, and both have to be
// checked. Most third-party executables — including the VC++ Redistributable
// installer this file exists to verify — carry their signature embedded in
// the file itself. Many Windows system binaries instead rely on catalog
// signing: the file carries no signature at all, and its hash is looked up in
// a separately-signed .cat catalog file that ships with Windows. A checker
// that only looks for an embedded signature reports TRUST_E_NOSIGNATURE for a
// perfectly legitimate, Microsoft-signed catalog file — this is exactly what
// an earlier version of this file got caught on, against notepad.exe (see
// TestVerifyMicrosoftSignatureAcceptsWindowsBinary).
//
// VerifyAuthenticodeOrg below tries an embedded signature first and falls
// back to a catalog lookup only when the file carries no signature at all;
// an embedded signature that's actually invalid never gets a second chance
// via catalog, since that would blur "tampered" and "differently signed"
// into the same outcome. For the org check on a catalog-verified file, rather
// than reach into WinVerifyTrust's internal provider-chain state — every API
// for that returns a raw pointer we would have to reinterpret by hand, which
// even golang.org/x/sys/windows's own equivalent code trips go vet's
// unsafeptr check on — this re-parses the catalog FILE's own embedded PKCS#7
// signature (catalogs are themselves embedded-signed containers) using the
// exact same, typed, vet-clean path already used for the embedded case.
// WinVerifyTrust still does the actual trust decision (chain validity,
// tampering) for whichever subject was checked; this only ever reads WHO
// signed, from a signature that was independently established as trusted.

var (
	modWintrust = windows.NewLazySystemDLL("wintrust.dll")
	modCrypt32  = windows.NewLazySystemDLL("crypt32.dll")

	procCryptMsgGetParam = modCrypt32.NewProc("CryptMsgGetParam")
	procCryptMsgClose    = modCrypt32.NewProc("CryptMsgClose")

	// Catalog-signature lookup (see the package doc above). None of these
	// return a pointer we have to reinterpret: CryptCATAdminEnumCatalogFromHash
	// hands back an opaque HCATINFO handle (just an integer token, not a Go
	// pointer — safe to carry as a plain uintptr/Handle), and
	// CryptCATCatalogInfoFromContext writes into a CATALOG_INFO the caller
	// already owns.
	procCryptCATAdminAcquireContext2         = modWintrust.NewProc("CryptCATAdminAcquireContext2")
	procCryptCATAdminReleaseContext          = modWintrust.NewProc("CryptCATAdminReleaseContext")
	procCryptCATAdminCalcHashFromFileHandle2 = modWintrust.NewProc("CryptCATAdminCalcHashFromFileHandle2")
	procCryptCATAdminEnumCatalogFromHash     = modWintrust.NewProc("CryptCATAdminEnumCatalogFromHash")
	procCryptCATCatalogInfoFromContext       = modWintrust.NewProc("CryptCATCatalogInfoFromContext")
	procCryptCATAdminReleaseCatalogContext   = modWintrust.NewProc("CryptCATAdminReleaseCatalogContext")
)

const (
	// certNameAttrType (CERT_NAME_ATTR_TYPE) asks CertGetNameString for one
	// named attribute of the subject rather than a display string.
	certNameAttrType = windows.CERT_NAME_ATTR_TYPE

	// cmsgSignerCertInfoParam is CMSG_SIGNER_CERT_INFO_PARAM: asks a decoded
	// PKCS#7 message for the issuer and serial number identifying the
	// certificate that signed it.
	cmsgSignerCertInfoParam = 7

	maxPath = 260
)

// oidOrganizationName is szOID_ORGANIZATION_NAME, the "O=" component of a
// certificate subject — read as a single parsed attribute rather than by
// matching against the whole subject line.
//
// That precision is the point: substring-matching the full subject DN (what
// the PowerShell check did with -like '*O=Microsoft Corporation*') would also
// accept a certificate issued to "Microsoft Corporation Services Ltd", since
// that string contains the expected one. Comparing the parsed attribute
// exactly cannot be widened that way.
var oidOrganizationName = []byte("2.5.4.10\x00")

// wintrustCatalogInfo mirrors WINTRUST_CATALOG_INFO, used to verify a file
// whose signature lives in a separate .cat catalog rather than embedded in
// the file itself. golang.org/x/sys/windows only defines WinTrustFileInfo
// (the embedded-signature case), not this one.
type wintrustCatalogInfo struct {
	cbStruct             uint32
	dwCatalogVersion     uint32
	pcwszCatalogFilePath *uint16
	pcwszMemberTag       *uint16
	pcwszMemberFilePath  *uint16
	hMemberFile          windows.Handle
	pbCalculatedFileHash *byte
	cbCalculatedFileHash uint32
	pcCatalogContext     uintptr // PCCTL_CONTEXT, unused — left nil
	hCatAdmin            windows.Handle
}

// catalogInfo mirrors CATALOG_INFO: the catalog file a given hash was found
// in, as returned by CryptCATCatalogInfoFromContext.
type catalogInfo struct {
	cbStruct       uint32
	wszCatalogFile [maxPath]uint16
}

// Authenticode/catalog failure codes worth naming, so a rejected installer
// reports what was actually wrong instead of a bare hex status.
const (
	trustENoSignature       = 0x800B0100
	trustESubjectNotTrusted = 0x800B0004
	trustEBadDigest         = 0x80096010
	trustEExplicitDistrust  = 0x800B0111
	certEExpired            = 0x800B0101
	certEUntrustedRoot      = 0x800B0109
	certEChaining           = 0x800B010A
	cryptESecuritySettings  = 0x80092026
)

// trustStatusMessage turns a WinVerifyTrust failure into something a player
// could act on or report, rather than an opaque code.
func trustStatusMessage(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}

	switch uint32(errno) {
	case trustENoSignature:
		return "the file carries no Authenticode signature"
	case trustESubjectNotTrusted:
		return "the signature is present but not trusted"
	case trustEBadDigest:
		return "the file was modified after it was signed"
	case trustEExplicitDistrust:
		return "the signature is explicitly distrusted on this machine"
	case certEExpired:
		return "the signing certificate has expired"
	case certEUntrustedRoot:
		return "the signing certificate does not chain to a trusted root"
	case certEChaining:
		return "the signing certificate chain could not be built"
	case cryptESecuritySettings:
		return "local security policy blocked the signature check"
	default:
		return fmt.Sprintf("WinVerifyTrust returned 0x%08X", uint32(errno))
	}
}

// isNoSignatureError reports whether err is specifically "this file carries
// no signature at all", as opposed to any other verification failure (an
// invalid, tampered, or untrusted one). Only this specific case is worth
// retrying against a catalog: every other failure means a signature WAS
// found and rejected, which a catalog lookup must not be allowed to overturn.
func isNoSignatureError(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && uint32(errno) == trustENoSignature
}

// VerifyAuthenticodeOrg checks that the file at path carries a valid, trusted
// Authenticode signature — embedded in the file, or via a Windows catalog —
// AND that the signing certificate was issued to wantOrg. Both must hold: a
// valid signature alone only says "somebody with a code-signing certificate
// vouched for this file", which is not the question being asked before
// running an installer elevated.
func VerifyAuthenticodeOrg(path, wantOrg string) error {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("invalid installer path: %w", err)
	}

	org, err := verifyEmbeddedTrust(pathUTF16)
	if err == nil {
		return compareOrg(org, wantOrg)
	}
	if !isNoSignatureError(err) {
		return fmt.Errorf("Authenticode verification failed: %s", trustStatusMessage(err))
	}

	// No embedded signature: many Windows system files (and, in principle,
	// other Microsoft-distributed binaries) are catalog-signed instead — see
	// the package doc at the top of this file.
	org, catErr := verifyCatalogTrust(pathUTF16)
	if catErr != nil {
		return fmt.Errorf(
			"Authenticode verification failed: %s (and catalog verification failed: %v)",
			trustStatusMessage(err), catErr,
		)
	}
	return compareOrg(org, wantOrg)
}

func compareOrg(org, wantOrg string) error {
	if org != wantOrg {
		return fmt.Errorf("signed by %q, expected %q", org, wantOrg)
	}
	return nil
}

// verifyEmbeddedTrust checks path's own embedded Authenticode signature via
// WinVerifyTrust and, only once that succeeds, reads the signer's
// organization out of that same embedded signature.
//
// Revocation is left at WTD_REVOKE_NONE throughout this file, matching what
// Get-AuthenticodeSignature reported before. An online revocation check would
// make the verdict depend on reaching a CA's OCSP responder, so a captive
// portal, a corporate proxy or a CA outage would start rejecting installers
// that are perfectly fine — a fail-closed check has to avoid failure modes
// that aren't about the file. Chain validation against the machine's trusted
// roots is what actually establishes "this really is Microsoft's file".
func verifyEmbeddedTrust(pathUTF16 *uint16) (string, error) {
	fileInfo := windows.WinTrustFileInfo{FilePath: pathUTF16}
	fileInfo.Size = uint32(unsafe.Sizeof(fileInfo))

	data := windows.WinTrustData{
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_NONE,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(&fileInfo),
		StateAction:                     windows.WTD_STATEACTION_VERIFY,
		ProvFlags:                       windows.WTD_SAFER_FLAG,
	}
	data.Size = uint32(unsafe.Sizeof(data))

	if err := verifyAndClose(&data); err != nil {
		return "", err
	}

	return embeddedSignerOrg(pathUTF16)
}

// verifyCatalogTrust looks up path's hash in the system's catalog database,
// verifies against the catalog that claims it (if any), and — once that
// succeeds — reads the signer organization off the CATALOG FILE's own
// embedded signature (catalogs are themselves embedded-signed PKCS#7
// containers), reusing embeddedSignerOrg rather than reaching into
// WinVerifyTrust's internal state for it.
func verifyCatalogTrust(pathUTF16 *uint16) (string, error) {
	file, err := windows.CreateFile(pathUTF16, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return "", fmt.Errorf("open file for catalog lookup: %w", err)
	}
	defer windows.CloseHandle(file)

	catAdmin, err := catAdminAcquireContext()
	if err != nil {
		return "", err
	}
	defer catAdminReleaseContext(catAdmin)

	hash, err := catAdminCalcHash(catAdmin, file)
	if err != nil {
		return "", err
	}

	catInfoHandle, err := catAdminEnumCatalogFromHash(catAdmin, hash)
	if err != nil {
		return "", fmt.Errorf("no catalog vouches for this file's hash: %w", err)
	}
	defer catAdminReleaseCatalogContext(catAdmin, catInfoHandle)

	catalogPath, err := catCatalogInfoFromContext(catInfoHandle)
	if err != nil {
		return "", err
	}

	memberTag, err := windows.UTF16PtrFromString(hashHex(hash))
	if err != nil {
		return "", fmt.Errorf("encode catalog member tag: %w", err)
	}

	catInfo := wintrustCatalogInfo{
		pcwszCatalogFilePath: catalogPath,
		pcwszMemberTag:       memberTag,
		pcwszMemberFilePath:  pathUTF16,
		hMemberFile:          file,
		pbCalculatedFileHash: &hash[0],
		cbCalculatedFileHash: uint32(len(hash)),
		hCatAdmin:            catAdmin,
	}
	catInfo.cbStruct = uint32(unsafe.Sizeof(catInfo))

	data := windows.WinTrustData{
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_NONE,
		UnionChoice:                     windows.WTD_CHOICE_CATALOG,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(&catInfo),
		StateAction:                     windows.WTD_STATEACTION_VERIFY,
		ProvFlags:                       windows.WTD_SAFER_FLAG,
	}
	data.Size = uint32(unsafe.Sizeof(data))

	if err := verifyAndClose(&data); err != nil {
		return "", fmt.Errorf("catalog verification failed: %s", trustStatusMessage(err))
	}

	return embeddedSignerOrg(catalogPath)
}

// verifyAndClose runs WinVerifyTrust for data (whichever subject union it was
// already set up for) and always releases the provider state afterwards —
// required regardless of the verdict, per WinVerifyTrust's own contract.
func verifyAndClose(data *windows.WinTrustData) error {
	verifyErr := windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)

	data.StateAction = windows.WTD_STATEACTION_CLOSE
	_ = windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)

	return verifyErr
}

// hashHex renders hash as the uppercase hex string Windows expects as a
// catalog member tag.
func hashHex(hash []byte) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, len(hash)*2)
	for i, b := range hash {
		out[i*2] = digits[b>>4]
		out[i*2+1] = digits[b&0xF]
	}
	return string(out)
}

// catAdminAcquireContext opens a catalog administrator context hashing with
// SHA-256, the algorithm modern (Windows 8+) catalogs are keyed on.
func catAdminAcquireContext() (windows.Handle, error) {
	algo, err := windows.UTF16PtrFromString("SHA256")
	if err != nil {
		return 0, err
	}

	var admin windows.Handle
	ret, _, err := procCryptCATAdminAcquireContext2.Call(
		uintptr(unsafe.Pointer(&admin)),
		0, // pgSubsystem: NULL, no specific driver-verification subsystem
		uintptr(unsafe.Pointer(algo)),
		0, // pStrongHashPolicy: NULL
		0,
	)
	if ret == 0 {
		return 0, fmt.Errorf("CryptCATAdminAcquireContext2: %w", err)
	}
	return admin, nil
}

func catAdminReleaseContext(admin windows.Handle) {
	_, _, _ = procCryptCATAdminReleaseContext.Call(uintptr(admin), 0)
}

// catAdminCalcHash computes file's catalog hash, sized by first asking for
// the required buffer length.
func catAdminCalcHash(admin windows.Handle, file windows.Handle) ([]byte, error) {
	var size uint32
	_, _, _ = procCryptCATAdminCalcHashFromFileHandle2.Call(
		uintptr(admin), uintptr(file), uintptr(unsafe.Pointer(&size)), 0, 0,
	)
	if size == 0 {
		return nil, errors.New("CryptCATAdminCalcHashFromFileHandle2: empty hash")
	}

	hash := make([]byte, size)
	ret, _, err := procCryptCATAdminCalcHashFromFileHandle2.Call(
		uintptr(admin), uintptr(file), uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&hash[0])), 0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptCATAdminCalcHashFromFileHandle2: %w", err)
	}
	return hash[:size], nil
}

// catAdminEnumCatalogFromHash finds a catalog whose contents include hash.
// The returned value is an opaque HCATINFO token — a Windows handle, not a Go
// pointer — so carrying it as a plain windows.Handle (itself just a uintptr)
// needs no unsafe.Pointer conversion.
func catAdminEnumCatalogFromHash(admin windows.Handle, hash []byte) (windows.Handle, error) {
	var prev windows.Handle
	ret, _, err := procCryptCATAdminEnumCatalogFromHash.Call(
		uintptr(admin),
		uintptr(unsafe.Pointer(&hash[0])),
		uintptr(len(hash)),
		0,
		uintptr(unsafe.Pointer(&prev)),
	)
	if ret == 0 {
		return 0, err
	}
	return windows.Handle(ret), nil
}

func catAdminReleaseCatalogContext(admin, catInfo windows.Handle) {
	_, _, _ = procCryptCATAdminReleaseCatalogContext.Call(uintptr(admin), uintptr(catInfo), 0)
}

// catCatalogInfoFromContext returns the path of the catalog file catInfo
// refers to, written by Windows into a CATALOG_INFO the caller owns — no
// pointer-returning call involved.
func catCatalogInfoFromContext(catInfo windows.Handle) (*uint16, error) {
	var info catalogInfo
	info.cbStruct = uint32(unsafe.Sizeof(info))

	ret, _, err := procCryptCATCatalogInfoFromContext.Call(
		uintptr(catInfo), uintptr(unsafe.Pointer(&info)), 0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptCATCatalogInfoFromContext: %w", err)
	}
	return &info.wszCatalogFile[0], nil
}

// embeddedSignerOrg reads the organization the signing certificate was
// issued to, by independently re-parsing the embedded PKCS#7 signature at
// pathUTF16 (a plain file, or a .cat catalog file — both are embedded-signed
// containers). Meaningful only once WinVerifyTrust has separately established
// that this exact signature is valid and trusted: on its own this reports who
// *claims* to have signed the file, with nothing here checking the claim.
func embeddedSignerOrg(pathUTF16 *uint16) (string, error) {
	var (
		encoding    uint32
		contentType uint32
		formatType  uint32
		store       windows.Handle
		msg         windows.Handle
	)

	err := windows.CryptQueryObject(
		windows.CERT_QUERY_OBJECT_FILE,
		unsafe.Pointer(pathUTF16),
		windows.CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED,
		windows.CERT_QUERY_FORMAT_FLAG_BINARY,
		0,
		&encoding,
		&contentType,
		&formatType,
		&store,
		&msg,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("read embedded signature: %w", err)
	}
	defer func() {
		_ = windows.CertCloseStore(store, 0)
		cryptMsgClose(msg)
	}()

	certInfo, err := signerCertInfo(msg)
	if err != nil {
		return "", err
	}

	// CERT_FIND_SUBJECT_CERT matches on issuer + serial number, i.e. the exact
	// certificate the message names as its signer, rather than whichever
	// certificate in the bundle happens to come first.
	cert, err := windows.CertFindCertificateInStore(
		store,
		encoding,
		0,
		windows.CERT_FIND_SUBJECT_CERT,
		unsafe.Pointer(certInfo),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("locate signer certificate: %w", err)
	}
	defer func() { _ = windows.CertFreeCertificateContext(cert) }()

	return certOrganization(cert)
}

// signerCertInfo pulls the signer's issuer/serial out of a decoded PKCS#7
// message.
func signerCertInfo(msg windows.Handle) (*windows.CertInfo, error) {
	var size uint32
	if err := cryptMsgGetParam(msg, cmsgSignerCertInfoParam, nil, &size); err != nil {
		return nil, fmt.Errorf("read signer info size: %w", err)
	}
	if size == 0 {
		return nil, errors.New("signature carries no signer information")
	}

	// Backed by a []uint64 rather than a []byte so the buffer is guaranteed
	// 8-byte aligned: it gets reinterpreted as a CERT_INFO, whose fields
	// include pointers.
	words := make([]uint64, (size+7)/8)
	buf := (*byte)(unsafe.Pointer(&words[0]))

	if err := cryptMsgGetParam(msg, cmsgSignerCertInfoParam, buf, &size); err != nil {
		return nil, fmt.Errorf("read signer info: %w", err)
	}

	return (*windows.CertInfo)(unsafe.Pointer(&words[0])), nil
}

// certOrganization returns the O= attribute of a certificate's subject.
func certOrganization(cert *windows.CertContext) (string, error) {
	// Called with a nil buffer, CertGetNameString returns the size needed
	// counting the terminating NUL — so 1 means "attribute not present".
	size := windows.CertGetNameString(
		cert,
		certNameAttrType,
		0,
		unsafe.Pointer(&oidOrganizationName[0]),
		nil,
		0,
	)
	if size <= 1 {
		return "", errors.New("signer certificate carries no organization name")
	}

	buf := make([]uint16, size)
	written := windows.CertGetNameString(
		cert,
		certNameAttrType,
		0,
		unsafe.Pointer(&oidOrganizationName[0]),
		&buf[0],
		size,
	)
	if written <= 1 {
		return "", errors.New("could not read the signer certificate's organization name")
	}

	return windows.UTF16ToString(buf), nil
}

// cryptMsgGetParam wraps CryptMsgGetParam, which golang.org/x/sys/windows
// doesn't bind. A nil data with a non-nil size asks for the size to allocate.
func cryptMsgGetParam(msg windows.Handle, paramType uint32, data *byte, size *uint32) error {
	ret, _, err := procCryptMsgGetParam.Call(
		uintptr(msg),
		uintptr(paramType),
		0, // signer index: the first (and, for Authenticode, only) signer
		uintptr(unsafe.Pointer(data)),
		uintptr(unsafe.Pointer(size)),
	)
	if ret == 0 {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno != 0 {
			return errno
		}
		return errors.New("CryptMsgGetParam failed")
	}
	return nil
}

// cryptMsgClose releases the message handle CryptQueryObject handed back.
func cryptMsgClose(msg windows.Handle) {
	_, _, _ = procCryptMsgClose.Call(uintptr(msg))
}
