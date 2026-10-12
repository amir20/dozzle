import { type Ref } from "vue";
import { type LogEvent, type LogMessage, LogEntry, asLogEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";

export function parseMessage(data: string): LogEntry<LogMessage> {
  const e = JSON.parse(data) as LogEvent;
  return asLogEntry(e);
}

/**
 * One page of lines from any number of containers in one request, merged by time.
 * `min` applies to each container; `lastSeenId` matches only the container that
 * line belongs to. `startId` applies to every container, and one that never sees
 * that id sends nothing, so pass it with a single container only.
 */
export async function loadBetween(
  containers: Container[],
  params: Ref<URLSearchParams>,
  from: Date,
  to: Date,
  {
    lastSeenId,
    startId,
    min,
    maxStart,
  }: { lastSeenId?: number; startId?: number; min?: number; maxStart?: number } = {},
) {
  const abortController = new AbortController();
  const signal = abortController.signal;

  const query = new URLSearchParams(params.value);
  query.set("format", "jsonl");
  for (const c of containers) query.append("container", `${c.host}/${c.id}`);
  query.set("from", from.toISOString());
  query.set("to", to.toISOString());
  if (min) query.set("min", String(min));
  if (maxStart) query.set("maxStart", String(maxStart));
  if (lastSeenId) query.set("lastSeenId", String(lastSeenId));
  if (startId) query.set("startId", String(startId));

  const stopWatcher = watchOnce(params, () => abortController.abort("stream changed"));
  const response = await fetch(withBase(`/api/containers/logs?${query}`), { signal });
  stopWatcher();
  if (!response.ok) throw new Error(`loading logs failed: ${response.status} ${(await response.text()).trim()}`);
  const logs = await response.text();

  if (!logs) return { logs: [] as LogEntry<LogMessage>[], signal };

  return {
    logs: logs
      .trim()
      .split("\n")
      .map((line) => parseMessage(line)),
    signal,
  };
}
