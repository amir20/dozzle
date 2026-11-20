// The login page under oidc sends the browser straight to the provider. Two
// things must stop it, and neither can be told by URL, so sessionStorage
// carries them. With storage blocked, the page never redirects on its own.

const LOGGED_OUT = "dozzle:logged-out";
const LAST_AUTO_LOGIN = "dozzle:auto-login-at";

// A sign in that works at the provider but leaves no cookie Dozzle accepts
// (Secure over plain http behind a proxy, replicas without a shared /data)
// lands back here within a second or two, and the provider signs the user in
// again silently. A second automatic redirect inside this window is that loop.
const LOOP_WINDOW_MS = 15_000;

// Logout ends at the login page, often by way of the provider's own logout,
// whose post_logout_redirect_uri is registered as the bare /login.
export function markLoggedOut() {
  try {
    sessionStorage.setItem(LOGGED_OUT, "1");
  } catch {
    // Storage blocked: the page will not auto-redirect anyway.
  }
}

// Reads and clears the mark, so only the first visit after logout stays put.
export function consumeLoggedOut(): boolean {
  try {
    const marked = sessionStorage.getItem(LOGGED_OUT) !== null;
    sessionStorage.removeItem(LOGGED_OUT);
    return marked;
  } catch {
    return true;
  }
}

// True when the page may redirect now, and records that it did.
export function claimAutoLogin(now = Date.now()): boolean {
  try {
    const last = Number(sessionStorage.getItem(LAST_AUTO_LOGIN));
    if (last && now - last < LOOP_WINDOW_MS) return false;
    sessionStorage.setItem(LAST_AUTO_LOGIN, String(now));
    return true;
  } catch {
    return false;
  }
}
