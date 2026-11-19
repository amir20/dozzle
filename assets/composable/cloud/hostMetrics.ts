/** One bucket of a host's history, as /api/cloud/metrics/hosts sends it. The
 *  optional fields are absent from an older cloud. */
export type HostPoint = {
  ts: number;
  cpu: number;
  memoryUsage: number;
  cpuMax?: number;
  memoryUsageMax?: number;
  networkRx?: number;
  networkTx?: number;
  diskRead?: number;
  diskWrite?: number;
};

export type Metric = "cpu" | "memory" | "network" | "disk";

/** One container's share of a host over the window: CPU percent, memory bytes,
 *  network and disk bytes per second. */
export type TopContainer = { name: string; avg: number; peak: number };

export type HostSeries = {
  hostId: string;
  points: HostPoint[];
  containers?: number;
  /** Only the list the request named. Absent for a reader with a container
   *  filter: the totals, not the names. */
  top?: Partial<Record<Metric, TopContainer[]>>;
};

const sumDefined = (xs: (number | undefined)[]) =>
  xs.some((x) => x !== undefined) ? xs.reduce<number>((a, x) => a + (x ?? 0), 0) : undefined;

/**
 * Every host's series as one, bucket by bucket. CPU is the average host's load:
 * each host's percent is of its own cores, so adding them says nothing. Memory
 * and I/O add up. A bucket only some hosts reported is combined from those.
 */
export function aggregateHosts(all: HostPoint[][]): HostPoint[] {
  const byTs = new Map<number, HostPoint[]>();
  for (const points of all) {
    for (const p of points) {
      const bucket = byTs.get(p.ts);
      if (bucket) bucket.push(p);
      else byTs.set(p.ts, [p]);
    }
  }
  return [...byTs.entries()]
    .sort(([a], [b]) => a - b)
    .map(([ts, ps]) => {
      return {
        ts,
        cpu: ps.reduce((a, p) => a + p.cpu, 0) / ps.length,
        cpuMax: ps.some((p) => p.cpuMax !== undefined)
          ? ps.reduce((a, p) => a + (p.cpuMax ?? p.cpu), 0) / ps.length
          : undefined,
        memoryUsage: ps.reduce((a, p) => a + p.memoryUsage, 0),
        // A host without a max counts with its average, so the sum never drops below the averages.
        memoryUsageMax: ps.some((p) => p.memoryUsageMax !== undefined)
          ? ps.reduce((a, p) => a + (p.memoryUsageMax ?? p.memoryUsage), 0)
          : undefined,
        networkRx: sumDefined(ps.map((p) => p.networkRx)),
        networkTx: sumDefined(ps.map((p) => p.networkTx)),
        diskRead: sumDefined(ps.map((p) => p.diskRead)),
        diskWrite: sumDefined(ps.map((p) => p.diskWrite)),
      };
    });
}

export type Stat = { avg: number; peak: number };

/** A host's average and peak per metric over the whole window. I/O is both
 *  directions together, as its chart draws it. */
export function hostStats(points: HostPoint[]): Record<Metric, Stat> {
  const stat = (avgOf: (p: HostPoint) => number, peakOf: (p: HostPoint) => number): Stat => ({
    avg: points.length ? points.reduce((a, p) => a + avgOf(p), 0) / points.length : 0,
    peak: points.length ? Math.max(...points.map(peakOf)) : 0,
  });
  const net = (p: HostPoint) => (p.networkRx ?? 0) + (p.networkTx ?? 0);
  const disk = (p: HostPoint) => (p.diskRead ?? 0) + (p.diskWrite ?? 0);
  return {
    cpu: stat(
      (p) => p.cpu,
      (p) => p.cpuMax ?? p.cpu,
    ),
    memory: stat(
      (p) => p.memoryUsage,
      (p) => p.memoryUsageMax ?? p.memoryUsage,
    ),
    network: stat(net, net),
    disk: stat(disk, disk),
  };
}

/**
 * The heaviest containers across hosts, heaviest first. Each host sends its own
 * top five, and the instance's top five are always among them, so merging the
 * lists is exact.
 */
export function mergeTop(lists: { hostId: string; list: TopContainer[] }[], limit: number) {
  return lists
    .flatMap(({ hostId, list }) => list.map((c) => ({ ...c, hostId })))
    .sort((a, b) => b.avg - a.avg)
    .slice(0, limit);
}
