/**
 * Listens for named SSE events with their payloads typed by a map from event name
 * to the JSON it carries. Each stream declares its map once, beside the code that
 * opens it, so a call site cannot pick its own payload type.
 *
 *   const on = typedEvents<EventsStream>(es);
 *   on("container-stat", (stat) => ...);
 */
export function typedEvents<M>(es: EventSource) {
  return <K extends keyof M & string>(name: K, listener: (data: M[K]) => void) =>
    es.addEventListener(name, (e) => listener(JSON.parse((e as MessageEvent).data) as M[K]));
}
