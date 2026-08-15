/**
 * Extracts the actual message from a rejected Wails backend call.
 *
 * Wails always rejects with `new Error(goErr.Error())` (see
 * `internal/frontend/runtime/desktop/calls.js`'s `Callback`), never with the
 * raw string. `String(err)` on that Error therefore returns
 * "Error: <message>" — good enough for an exact string match to always fail
 * silently (wine_not_found never matched in HomePage's handleLaunch), and
 * noisy when shown to the player ("Échec du déplacement : Error: ...").
 */
export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
