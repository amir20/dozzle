// A stand-in for the browser EventSource. Specs assign it to `global.EventSource`,
// find the instance the code opened by url in `sources`, and drive it by hand.
// Like the real thing before `onopen`, it starts CONNECTING (0); `emitOpen` moves it
// to OPEN (1) and `close` to CLOSED (2). It has no static readyState constants, so a
// spec that needs them assigns them itself.
type Listener = (event: any) => void;

export const sources: Record<string, EventSourceMock> = {};

export class EventSourceMock {
  url: string;
  withCredentials: boolean;
  readyState = 0;
  onopen: ((event?: any) => void) | null = null;
  onmessage: Listener | null = null;
  onerror: Listener | null = null;
  private listeners = new Map<string, Listener[]>();

  constructor(url: string, configuration: { withCredentials?: boolean } = { withCredentials: false }) {
    this.url = url;
    this.withCredentials = configuration.withCredentials ?? false;
    sources[url] = this;
  }

  addEventListener(eventName: string, listener: Listener) {
    this.listeners.set(eventName, [...(this.listeners.get(eventName) ?? []), listener]);
  }

  removeEventListener(eventName: string, listener: Listener) {
    const list = this.listeners.get(eventName) ?? [];
    const index = list.lastIndexOf(listener);
    if (index !== -1) this.listeners.set(eventName, list.toSpliced(index, 1));
  }

  close() {
    this.readyState = 2;
  }

  emit(eventName: string, messageEvent?: any) {
    for (const listener of this.listeners.get(eventName) ?? []) listener(messageEvent);
  }

  emitError(error?: any) {
    this.onerror?.(error);
  }

  emitOpen() {
    this.readyState = 1;
    this.onopen?.();
  }

  emitMessage(message: any) {
    this.onmessage?.(message);
  }
}

// Typed as the global so a spec can assign it to `global.EventSource` as is. The
// CONNECTING/OPEN/CLOSED statics the global's type promises are not there at runtime.
export default EventSourceMock as typeof EventSourceMock & typeof globalThis.EventSource;
