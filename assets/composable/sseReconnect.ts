const BASE_DELAY = 1_000;
const MAX_DELAY = 30_000;

type Options = {
  /** Opens a brand new EventSource. Must replace whatever `source()` returns. */
  connect: () => void;
  /** The EventSource currently held by the caller, or null before the first connect. */
  source: () => EventSource | null;
  /**
   * Milliseconds of silence after which an OPEN connection is assumed dead. Only useful
   * for a stream with an observable heartbeat; leave unset otherwise.
   */
  staleAfter?: number;
};

/**
 * Keeps an EventSource alive across the failures the browser does not handle itself.
 *
 * A browser retries a dropped stream only while the source is CONNECTING. The moment it
 * reaches CLOSED — any non-2xx response, a wrong content type, an auth redirect, which is
 * what a reverse proxy returns on a 502/504 or a restart — it gives up permanently and the
 * stream stays dead until the page is reloaded. That is the case this retries.
 */
export function useSseReconnect({ connect, source, staleAfter }: Options) {
  let attempt = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let lastActivity = Date.now();

  const cancel = () => {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  };

  const scheduleReconnect = () => {
    if (timer !== null) return;
    // jittered so a proxy coming back up isn't hit by every open tab at once
    const delay = Math.min(BASE_DELAY * 2 ** attempt, MAX_DELAY) + Math.random() * 500;
    attempt++;
    timer = setTimeout(() => {
      timer = null;
      connect();
    }, delay);
  };

  const reconnectNow = () => {
    cancel();
    attempt = 0;
    lastActivity = Date.now();
    connect();
  };

  /**
   * A machine waking from sleep leaves a socket that is still OPEN but dead: no error
   * fires and nothing arrives. Coming back to the tab or regaining network is the moment
   * to notice, so both are checked against the heartbeat.
   */
  const wake = () => {
    if (document.visibilityState !== "visible" || !navigator.onLine) return;
    const es = source();
    const stale = staleAfter !== undefined && Date.now() - lastActivity > staleAfter;
    if (!es || es.readyState === EventSource.CLOSED || stale) {
      reconnectNow();
    }
  };

  document.addEventListener("visibilitychange", wake);
  window.addEventListener("online", wake);

  return {
    /** Call from the source's `error` handler. */
    onError() {
      if (source()?.readyState === EventSource.CLOSED) {
        scheduleReconnect();
      }
    },
    /** Call from the source's `open` handler. */
    onOpen() {
      cancel();
      attempt = 0;
      lastActivity = Date.now();
    },
    /** Call whenever the stream delivers anything, heartbeats included. */
    onActivity() {
      lastActivity = Date.now();
    },
    dispose() {
      cancel();
      document.removeEventListener("visibilitychange", wake);
      window.removeEventListener("online", wake);
    },
  };
}
