/**
 * Parses the JSON payload of an SSE `MessageEvent`. EventSource listeners
 * receive a `MessageEvent` whose `data` is an untyped string, so the parsed
 * shape is provided via the type parameter at each call site.
 */
export function parseEventData<T>(event: Event): T {
  return JSON.parse((event as MessageEvent).data) as T;
}
