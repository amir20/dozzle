/**
 * Resolves a caller-supplied post-login redirect, keeping it on this origin.
 *
 * Prefix checks alone are not enough. A value like "/\t/evil.com" starts with a
 * single slash and so survives a `startsWith("//")` test, but the URL parser
 * strips ASCII tabs and newlines, leaving a protocol-relative URL that
 * navigates to evil.com. Resolving with the browser's own parser and comparing
 * origins is the only check that agrees with what navigation will actually do.
 *
 * The structural check still runs first so the result does not depend on
 * whether a base path happens to be configured: with a base, "//evil.com"
 * concatenates into a harmless same-origin path, without one it does not.
 */
export function safeRedirect(path: string | null | undefined, base: string, origin: string): string {
  const fallback = `${base}/`.replace("//", "/");

  if (!path || !path.startsWith("/") || path.startsWith("//") || path.startsWith("/\\")) {
    return fallback;
  }

  try {
    const url = new URL(`${base}${path}`, origin);
    if (url.origin !== origin) return fallback;
    return url.pathname + url.search + url.hash;
  } catch {
    return fallback;
  }
}
