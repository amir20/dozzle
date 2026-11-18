import type { Container } from "@/models/Container";

/** One sample of what Dozzle already holds in the browser. CPU is a share of
 *  the cores it is measured against, memory is bytes. */
export type LivePoint = { cpu: number; memoryUsage: number };

/**
 * A live series for the metrics panel to fall back on while Cloud has too
 * little history to draw. The window is the 300 samples Dozzle keeps per
 * container, padded at the front like `Container.statsHistory`: everything
 * before `sampledFrom` is not a measurement.
 */
export type LiveSeries = {
  id: string;
  name: string;
  points: LivePoint[];
  sampledFrom: number;
  /** What memory is drawn against: the container's limit or the host's RAM. 0 for none. */
  memoryLimit: number;
};

/** Cloud needs this many points before its history is worth drawing. */
export const MIN_CLOUD_POINTS = 2;

/** One container's window, CPU divided by its limit or the host's cores like its table row. */
export function containerLiveSeries(container: Container, hostCores: number): LiveSeries {
  const cores = container.cpuLimit > 0 ? container.cpuLimit : hostCores || 1;
  const history = container.statsHistory;
  return {
    id: container.id,
    name: container.name,
    points: history.map((s) => ({ cpu: s.cpu / cores, memoryUsage: s.memoryUsage })),
    sampledFrom: Math.max(0, history.length - container.sampledStats),
    memoryLimit: container.memoryLimit > 0 ? container.memoryLimit : 0,
  };
}

/** A host's running containers summed per sample, the same way its home page card does. */
export function hostLiveSeries(
  host: { id: string; name: string; nCPU: number; memTotal: number },
  containers: Container[],
): LiveSeries {
  const running = containers.filter((c) => c.host === host.id && c.state === "running");
  const cores = host.nCPU || 1;
  const length = 300;
  const points: LivePoint[] = [];
  for (let i = length; i >= 1; i--) {
    let cpu = 0;
    let memoryUsage = 0;
    for (const c of running) {
      const stat = c.statsHistory.at(-i);
      if (!stat) continue;
      cpu += stat.cpu / cores;
      memoryUsage += stat.memoryUsage;
    }
    points.push({ cpu, memoryUsage });
  }
  // The longest real run: a container newer than the rest adds zero before it existed.
  const sampled = Math.min(length, Math.max(0, ...running.map((c) => c.sampledStats)));
  return { id: host.id, name: host.name, points, sampledFrom: length - sampled, memoryLimit: host.memTotal || 0 };
}

/** How many whole minutes the longest series covers. Samples arrive once a second. */
export function liveMinutes(series: LiveSeries[]) {
  const sampled = Math.max(0, ...series.map((s) => s.points.length - s.sampledFrom));
  return Math.max(1, Math.ceil(sampled / 60));
}
