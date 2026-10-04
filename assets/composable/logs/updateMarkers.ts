import { type LogMessage, DeployLogEntry, LogEntry, LoadMoreLogEntry, RangeEdgeLogEntry } from "@/models/LogEntry";
import { isStreamLog } from "@/composable/cloud/alertMerger";

// How long after an update its container's first line can come and still count as
// the start of that container. Most print something the moment they start; a slow
// one that does not is found by loading older lines instead.
export const MARKER_START_SLACK_MS = 2 * 60_000;

/** The date of each container's oldest line in the window. */
export function firstLineByContainer(list: LogEntry<LogMessage>[]): Map<string, number> {
  const first = new Map<string, number>();
  for (const entry of list) {
    if (!entry.containerID || !isStreamLog(entry)) continue;
    const t = entry.date.getTime();
    const seen = first.get(entry.containerID);
    if (seen === undefined || t < seen) first.set(entry.containerID, t);
  }
  return first;
}

/**
 * Whether the window reaches back to the start of the container an update created,
 * so its marker has a place: the container's oldest line on screen came right after
 * the update. A container with no line on screen yet has nothing to place it against.
 */
export function windowReachesUpdate(marker: DeployLogEntry, first: Map<string, number>): boolean {
  const line = first.get(marker.containerID);
  return line !== undefined && line - marker.date.getTime() <= MARKER_START_SLACK_MS;
}

/**
 * Puts the marker where its update happened: above its container's first line, and
 * above any other line that came after it. The load-more row stays on top.
 */
export function insertMarker(list: LogEntry<LogMessage>[], marker: DeployLogEntry): LogEntry<LogMessage>[] {
  let i = 0;
  while (i < list.length && (list[i] instanceof LoadMoreLogEntry || list[i] instanceof RangeEdgeLogEntry)) i++;
  const at = marker.date.getTime();
  while (i < list.length) {
    const entry = list[i];
    if (entry.date.getTime() > at) break;
    if (entry.containerID === marker.containerID && isStreamLog(entry)) break;
    i++;
  }
  return [...list.slice(0, i), marker, ...list.slice(i)];
}

/**
 * Moves each marker that has a line of its own container above it back over that
 * container's first line. Loading older lines prepends them, which leaves a marker
 * placed on a container with no lines yet below the lines that then arrive.
 */
export function repositionMarkers(list: LogEntry<LogMessage>[]): LogEntry<LogMessage>[] {
  const seen = new Set<string>();
  const misplaced: DeployLogEntry[] = [];
  for (const entry of list) {
    if (entry instanceof DeployLogEntry) {
      if (seen.has(entry.containerID)) misplaced.push(entry);
    } else if (entry.containerID && isStreamLog(entry)) {
      seen.add(entry.containerID);
    }
  }
  if (misplaced.length === 0) return list;
  const rest = list.filter((e) => !misplaced.includes(e as DeployLogEntry));
  return misplaced.reduce(insertMarker, rest);
}
