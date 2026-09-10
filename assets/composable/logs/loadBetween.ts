import { type Ref } from "vue";
import { type LogEvent, type LogMessage, LogEntry, asLogEntry } from "@/models/LogEntry";
import { Container } from "@/models/Container";

export function parseMessage(data: string): LogEntry<LogMessage> {
  const e = JSON.parse(data) as LogEvent;
  return asLogEntry(e);
}

export async function loadBetween(
  container: Container | Ref<Container>,
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
  const c = toValue(container);
  const url = `/api/hosts/${c.host}/containers/${c.id}/logs`;
  const abortController = new AbortController();
  const signal = abortController.signal;

  function buildUrl() {
    const loadMoreParams = new URLSearchParams(params.value);
    loadMoreParams.append("from", from.toISOString());
    loadMoreParams.append("to", to.toISOString());
    if (min) {
      loadMoreParams.append("min", String(min));
    }
    if (maxStart) {
      loadMoreParams.append("maxStart", String(maxStart));
    }
    if (lastSeenId) {
      loadMoreParams.append("lastSeenId", String(lastSeenId));
    }
    if (startId) {
      loadMoreParams.append("startId", String(startId));
    }
    return withBase(`${url}?${loadMoreParams.toString()}`);
  }

  const fullUrl = buildUrl();
  const stopWatcher = watchOnce(params, () => abortController.abort("stream changed"));
  const logs = await (await fetch(fullUrl, { signal })).text();
  stopWatcher();

  if (!logs) return { logs: [] as LogEntry<LogMessage>[], signal };

  return {
    logs: logs
      .trim()
      .split("\n")
      .map((line) => parseMessage(line)),
    signal,
  };
}
