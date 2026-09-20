// A stream that lands in CLOSED cannot say why it did: EventSource hides the status
// code, so an expired session looks exactly like a restarted server or a proxy that
// is briefly down. useSseReconnect therefore retries all of them forever, which is
// right for every case but this one — no number of retries brings back a session that
// is gone, and the app sits behind an error toast reconnecting until someone reloads
// by hand. That is the black page an iOS home screen app comes back to: the webview
// survives for days, so it is always the resume path, never a fresh load, that finds
// the session expired.
//
// Asking a cheap authenticated endpoint is the only way to tell the two apart.

// Reloading hands the decision back to the server, which already knows where an
// unauthenticated request belongs — the login page under simple and oidc, the proxy's
// own 401 under forward-proxy — and what to put in redirectUrl. Rebuilding that here
// would be a second copy of executeTemplate's rules, drifting from the first.
const reload = () => window.location.reload();

let inFlight: Promise<void> | null = null;
let expired = false;

/**
 * Checks whether this browser still has a session, and reloads the page when it does
 * not. A no-op when the instance runs without auth, and after the first positive
 * answer: every open stream fails at once on resume, and they should cost one request
 * between them, not one each.
 */
export async function checkSession() {
  if (expired || config.authProvider === "none") return;
  inFlight ??= probe().finally(() => (inFlight = null));
  return inFlight;
}

async function probe() {
  let response: Response;
  try {
    // Cheap, authenticated, and already registered in every mode.
    response = await fetch(withBase("/api/version"), { cache: "no-store" });
  } catch {
    // Offline, or nothing is listening. Both are the retryable case this is here
    // to keep out of the reload path.
    return;
  }

  // Only 401 means the session. A 403 is a role the user does not have, which
  // signing in again cannot change.
  if (response.status !== 401) return;

  expired = true;
  reload();
}
