// Logout ends at the login page, often by way of the provider's own logout, whose
// post_logout_redirect_uri is registered as the bare /login. So the page cannot
// be told by URL that the user just left; a sessionStorage mark carries it across.
const KEY = "dozzle:logged-out";

export function markLoggedOut() {
  try {
    sessionStorage.setItem(KEY, "1");
  } catch {
    // Storage blocked: the login page will redirect to the provider after logout.
  }
}

// Reads and clears the mark, so only the first visit after logout stays put.
export function consumeLoggedOut(): boolean {
  try {
    const marked = sessionStorage.getItem(KEY) !== null;
    sessionStorage.removeItem(KEY);
    return marked;
  } catch {
    return false;
  }
}
