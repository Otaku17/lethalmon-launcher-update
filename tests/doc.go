// Package tests holds the launcher's test suite, kept out of the backend
// package so the two stay visually separate.
//
// The consequence is that these tests are black-box: they can only reach what
// package backend exports, which is why a number of internals — SafeExtractPath,
// VerifyUpdateSignature, CompareVersions, ReadGameOpts and friends — are
// exported despite not being part of the API Wails binds. Anything that needs
// to test genuinely private behaviour has to live in backend/ instead.
//
// A few files here are behind a windows build tag. Those are the ones most
// worth running — VC++ Redistributable detection, the self-update file swap,
// the process-filter quoting — and they are silently skipped on other
// platforms, so run the suite on Windows before trusting a green result.
package tests
