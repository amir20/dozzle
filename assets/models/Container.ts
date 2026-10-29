import type {
  ContainerHealth,
  ContainerJson,
  ContainerMount,
  ContainerStat,
  ContainerState,
  MountStat,
  StatColumns,
  VolumeUsage,
} from "@/types/Container";
import { Ref, ShallowRef } from "vue";

export type Stat = Omit<ContainerStat, "id">;

export const emptyStat = (): Stat => ({
  cpu: 0,
  memory: 0,
  memoryUsage: 0,
  networkRxTotal: 0,
  networkTxTotal: 0,
  diskReadTotal: 0,
  diskWriteTotal: 0,
});

const DELTA_ENCODED = ["memoryUsage", "networkRxTotal", "networkTxTotal", "diskReadTotal", "diskWriteTotal"] as const;

// Turns the columnar history the server sends back into one Stat per point.
export function decodeStats(columns: StatColumns | undefined): Stat[] {
  if (!columns) return [];
  const sums = Object.fromEntries(DELTA_ENCODED.map((k) => [k, 0])) as Record<(typeof DELTA_ENCODED)[number], number>;
  return columns.cpu.map((cpu, i) => {
    for (const k of DELTA_ENCODED) sums[k] += columns[k][i];
    return { cpu, memory: columns.memory[i], ...sums };
  });
}

type Rates = { networkRx: number; networkTx: number; diskRead: number; diskWrite: number };

const emptyRates = (): Rates => ({ networkRx: 0, networkTx: 0, diskRead: 0, diskWrite: 0 });

// A counter that went backwards was reset (the container restarted), not negative.
const ratesBetween = (prev: Stat, last: Stat): Rates => ({
  networkRx: Math.max(0, last.networkRxTotal - prev.networkRxTotal),
  networkTx: Math.max(0, last.networkTxTotal - prev.networkTxTotal),
  diskRead: Math.max(0, last.diskReadTotal - prev.diskReadTotal),
  diskWrite: Math.max(0, last.diskWriteTotal - prev.diskWriteTotal),
});

// The live hosts map, not config.hosts: that is the list the page loaded with, and an
// agent added since then would have no name to show.
const { hosts } = useHosts();

/**
 * Not running and not about to: it writes no more logs and reports no more stats
 * until someone starts it. That includes one that was created and never started.
 * Paused and restarting containers are on their way back, and a restarting one
 * still carries the previous run's finishedAt, so neither counts.
 */
export const isStopped = (container: { state: ContainerState }) =>
  container.state === "exited" ||
  container.state === "dead" ||
  container.state === "deleted" ||
  container.state === "created";

/**
 * Bytes per sample between the last two real samples, which at the collectors' one
 * second cadence is bytes per second. Undefined until there are two to compare, or
 * once the container stops reporting.
 */
export const networkRate = (container: Container, average = false): { rx: number; tx: number } | undefined => {
  if (isStopped(container) || container.sampledStats < 2) return undefined;
  const rates = average ? container.movingAverageRate : latestRates(container);
  return { rx: rates.networkRx, tx: rates.networkTx };
};

/** Same as `networkRate`, for the block device reads and writes. */
export const diskRate = (container: Container, average = false): { read: number; write: number } | undefined => {
  if (isStopped(container) || container.sampledStats < 2) return undefined;
  const rates = average ? container.movingAverageRate : latestRates(container);
  return { read: rates.diskRead, write: rates.diskWrite };
};

const latestRates = (container: Container) => {
  const [prev, last] = container.statsHistory.slice(-2);
  return ratesBetween(prev, last);
};

// Docker leaves startedAt/finishedAt at the zero time when a container never ran or is
// still running, which renders as "2027 years ago".
const isSet = (date: Date) => date.getFullYear() > 1;

// Only absolute http(s) URLs are honored so a label can't smuggle in `javascript:`.
const httpUrl = (value: string | undefined) => {
  const raw = value?.trim();
  if (!raw) return undefined;
  try {
    const { protocol } = new URL(raw);
    return protocol === "http:" || protocol === "https:" ? raw : undefined;
  } catch {
    return undefined;
  }
};

/**
 * What `docker ps` puts in its STATUS column, as a state and the moment it is measured
 * from: up since it started, or exited or restarting since it last stopped. Undefined
 * for a container that never ran, where there is no such moment.
 */
export const uptime = (
  container: Pick<Container, "state" | "startedAt" | "finishedAt">,
): { state: "up" | "paused" | "restarting" | "exited"; since: Date } | undefined => {
  switch (container.state) {
    case "running":
    case "paused":
      return isSet(container.startedAt)
        ? { state: container.state === "running" ? "up" : "paused", since: container.startedAt }
        : undefined;
    case "restarting":
      return isSet(container.finishedAt) ? { state: "restarting", since: container.finishedAt } : undefined;
    case "exited":
    case "dead":
      return isSet(container.finishedAt) ? { state: "exited", since: container.finishedAt } : undefined;
    default:
      return undefined;
  }
};

/**
 * The timestamp a container's last state change happened at, falling back to its
 * creation when it never ran.
 */
export const lastTransition = (container: Pick<Container, "state" | "startedAt" | "finishedAt" | "created">) => {
  const date =
    container.state === "running" || container.state === "paused" ? container.startedAt : container.finishedAt;
  return isSet(date) ? date : container.created;
};

/**
 * Which of start and stop the container takes. Docker refuses to start a paused
 * container but stops one fine, and a deleted one takes neither.
 */
export const powerAction = (container: { state: ContainerState }): "start" | "stop" | undefined => {
  if (container.state === "deleted") return undefined;
  return isStopped(container) ? "start" : "stop";
};

export class GroupedContainers {
  constructor(
    public readonly name: string,
    public readonly containers: Container[],
  ) {}
}

export class HistoricalContainer {
  constructor(
    public readonly container: Container,
    public readonly date: Date,
    // Set for a time range, where `date` is its start; absent for a single moment.
    public readonly until?: Date,
  ) {}
}

export class Container {
  private _stat: Ref<Stat>;
  private _name: string;
  private _health?: ContainerHealth;
  // Shallow, and `markRaw` on the array inside it: a Container lives in the store's
  // deeply reactive `containers` array, so a plain `ref` here would proxy the window
  // and all 300 `Stat`s in it. Every tick then pays for a proxy plus a deep array
  // trigger, per container, while the table's chart cells read the whole window back
  // through those proxies once a second. Measured at 50 containers: 61ms of main
  // thread per tick that way, 1.5ms this way.
  //
  // `markRaw` is doing real work and is not redundant with `shallowRef`: the proxy
  // re-wraps whatever a getter *returns*, so without it `statsHistory` would hand
  // out a reactive array however the field is declared.
  //
  // The series is append-only and nothing edits a sample after it lands, so none of
  // that tracking bought anything -- readers only have to be woken, which the
  // `triggerRef` in `updateStat` does.
  private readonly _statsHistory: ShallowRef<Stat[]>;
  // How many of the entries in `_statsHistory` are real samples rather than the
  // padding in front of them. Always counts from the end.
  private _sampledStats: Ref<number>;
  private readonly movingAverageStat: Ref<Stat>;
  // The stats carry running totals for network and disk, so their average has to be
  // taken over the per-sample difference rather than the totals themselves.
  private readonly movingAverageRateRef: Ref<Rates>;

  public mounts: ContainerMount[];
  public mountStats: Record<string, MountStat>;

  constructor(
    public readonly id: string,
    public readonly created: Date,
    public startedAt: Date,
    public finishedAt: Date,
    public readonly image: string,
    name: string,
    public readonly command: string,
    public readonly host: string,
    public readonly labels = {} as Record<string, string>,
    public state: ContainerState,
    public readonly cpuLimit: number,
    public readonly memoryLimit: number,
    stats: Stat[],
    public group?: string,
    health?: ContainerHealth,
    public isNew: boolean = false,
    mounts: ContainerMount[] = [],
    mountStats: Record<string, MountStat> = {},
    public readonly ports: string[] = [],
    // bytes in the writable layer; undefined until the server has measured it
    public sizeRw?: number,
    public volumes: VolumeUsage[] = [],
  ) {
    this._health = health;
    this.mounts = mounts;
    this.mountStats = mountStats;
    const defaultStat = emptyStat();
    this._stat = shallowRef(stats.at(-1) || defaultStat);
    const recentStats = stats.slice(-300);
    // Padded to a full window on purpose: the chart keeps its width and the bars
    // stay put as samples arrive, instead of growing in from the left. The padding
    // is not a measurement, so anything that reads a value back out (averages, the
    // hover readout) has to tell the two apart -- see `sampledStats`.
    const padding = Array(300 - recentStats.length).fill(defaultStat);
    this._statsHistory = shallowRef(markRaw([...padding, ...recentStats]));
    this._sampledStats = shallowRef(recentStats.length);
    this.movingAverageStat = shallowRef(stats.at(-1) || defaultStat);
    this.movingAverageRateRef = shallowRef(
      recentStats.length >= 2 ? ratesBetween(recentStats.at(-2)!, recentStats.at(-1)!) : emptyRates(),
    );

    this._name = name;
  }

  // Raw, and mutating it in place would tell nobody. Treat it as read-only and go
  // through `updateStat`, which does the waking.
  get statsHistory() {
    return unref(this._statsHistory);
  }

  // The count of real samples at the end of `statsHistory`. Everything before them
  // is padding that keeps the chart full width while the series scrolls in, and is
  // not a measurement: it must never be averaged or read out as a value.
  get sampledStats() {
    return unref(this._sampledStats);
  }

  get movingAverage() {
    return unref(this.movingAverageStat);
  }

  get movingAverageRate() {
    return unref(this.movingAverageRateRef);
  }

  get stat() {
    return unref(this._stat);
  }

  get hostLabel() {
    return hosts.value[this.host]?.name;
  }

  // Writable layer plus every volume the container mounts. A shared volume counts in
  // each container that uses it. Undefined until something has been measured.
  get diskTotal(): number | undefined {
    if (this.sizeRw === undefined && this.volumes.length === 0) return undefined;
    return this.volumes.reduce((sum, v) => sum + v.size, this.sizeRw ?? 0);
  }

  get storageKey() {
    return `${stripVersion(this.image)}:${this.command}`;
  }

  get namespace() {
    return (
      this.labels["dev.dozzle.group"] ||
      this.labels["coolify.projectName"] ||
      this.labels["com.docker.stack.namespace"] ||
      this.labels["com.docker.compose.project"]
    );
  }

  get customGroup() {
    return this.group;
  }

  /**
   * Bundled app icon slug for this container, derived from its image. `dev.dozzle.icon`
   * overrides the guess, and `none` opts a container out when the guess is wrong. The
   * label may also carry the icon itself as a data URI, which is returned untouched
   * because base64 does not survive lowercasing.
   */
  get icon() {
    const label = this.labels["dev.dozzle.icon"]?.trim();
    if (label && isDataIcon(label)) return label;
    const override = label?.toLowerCase();
    if (override) return override === "none" || !hasIcon(override) ? undefined : override;
    return iconSlugForImage(this.image);
  }

  /**
   * Opt-in link to whatever web UI this container serves, from `dev.dozzle.url`.
   */
  get url() {
    return httpUrl(this.labels["dev.dozzle.url"]);
  }

  /**
   * Where to read what an update changes. `dev.dozzle.releases` wins; otherwise a GitHub
   * repository the image declares as its source gets its releases page. Docker copies image
   * labels onto the container, so most published images need no label at all.
   */
  get releaseNotesUrl() {
    const override = httpUrl(this.labels["dev.dozzle.releases"]);
    if (override) return override;
    const source = httpUrl(this.labels["org.opencontainers.image.source"] || this.labels["org.label-schema.vcs-url"]);
    if (!source) return undefined;
    const { hostname, pathname } = new URL(source);
    if (hostname !== "github.com" && hostname !== "www.github.com") return undefined;
    const [owner, repo] = pathname.split("/").filter(Boolean);
    if (!owner || !repo) return undefined;
    return `https://github.com/${owner}/${repo.replace(/\.git$/, "")}/releases`;
  }

  /**
   * The release behind a floating `latest` (or untagged) image, read from the version
   * label the image carries. Docker copies image labels onto the container, so this
   * needs no registry call; images without the label have nothing to say.
   */
  get imageVersion() {
    const ref = stripImageDigest(this.image);
    const lastSegment = ref.slice(ref.lastIndexOf("/") + 1);
    const tag = lastSegment.includes(":") ? lastSegment.slice(lastSegment.indexOf(":") + 1) : "latest";
    if (tag !== "latest") return undefined;
    const version = (
      this.labels["org.opencontainers.image.version"] || this.labels["org.label-schema.version"]
    )?.trim();
    return version && version !== tag ? version : undefined;
  }

  /**
   * Published tcp bindings, parsed out of Docker's `ip:host->container/proto` strings.
   * Ports that are only exposed, never published, have no host side and are dropped.
   */
  get portMappings() {
    const byHostPort = new Map<number, number>();

    for (const binding of this.ports) {
      if (!binding.endsWith("/tcp") || !binding.includes("->")) continue;
      const [from, to] = binding.split("->");
      const host = Number(from.split(":").pop());
      const target = Number(to.replace("/tcp", ""));
      if (!Number.isInteger(host) || host <= 0 || host > 65535) continue;
      if (!byHostPort.has(host)) byHostPort.set(host, Number.isInteger(target) ? target : host);
    }

    return [...byHostPort.entries()].sort(([a], [b]) => a - b).map(([host, container]) => ({ host, container }));
  }

  /** Host ports this container publishes. Suggests a value when `dev.dozzle.url` is missing. */
  get publishedPorts() {
    return this.portMappings.map(({ host }) => host);
  }

  /**
   * Public URLs read out of Traefik v2/v3 router labels. A container behind a reverse proxy
   * publishes no host port, so the only place its browser-reachable address exists is here.
   * Suggestions for the `dev.dozzle.url` snippet only, never rendered as a link.
   */
  get traefikUrls() {
    if (this.labels["traefik.enable"]?.trim().toLowerCase() === "false") return [];

    const routers = new Map<string, { rule?: string; entrypoints?: string; tls: boolean }>();
    for (const [key, value] of Object.entries(this.labels)) {
      const match = key.match(/^traefik\.http\.routers\.([^.]+)\.(rule|entrypoints|tls)(\..+)?$/i);
      if (!match) continue;
      const [, name, prop, suffix] = match;
      const router = routers.get(name) ?? { tls: false };
      switch (prop.toLowerCase()) {
        case "rule":
          if (!suffix) router.rule = value;
          break;
        case "entrypoints":
          if (!suffix) router.entrypoints = value;
          break;
        // Bare `tls` can be turned off; any sub-key such as `tls.certresolver` implies https.
        case "tls":
          router.tls ||= suffix ? true : value.trim().toLowerCase() !== "false";
          break;
      }
      routers.set(name, router);
    }

    const urls = new Set<string>();
    for (const name of [...routers.keys()].sort()) {
      const { rule, entrypoints, tls } = routers.get(name)!;
      if (!rule) continue;
      const secure = tls || /(^|,)\s*(websecure|https)\s*(,|$)/i.test(entrypoints ?? "");
      const path = rule.match(/\bPath(?:Prefix)?\(`([^`]+)`\)/)?.[1] ?? "";
      for (const [, hosts] of rule.matchAll(/\bHost\(([^)]*)\)/g)) {
        for (const [, host] of hosts.matchAll(/`([^`]+)`/g)) {
          urls.add(`${secure ? "https" : "http"}://${host}${path === "/" ? "" : path}`);
        }
      }
    }

    return [...urls];
  }

  set name(name: string) {
    this._name = name;
  }

  get name() {
    return this.isSwarm
      ? this.labels["com.docker.swarm.task.name"]
          .replace(`.${this.labels["com.docker.swarm.task.id"]}`, "")
          .replace(`.${this.labels["com.docker.swarm.node.id"]}`, "")
      : this._name;
  }

  set health(health: ContainerHealth | undefined) {
    this._health = health;
  }

  // A healthcheck only runs while the container does, and Docker keeps the last
  // result (flipped to unhealthy by the kill) after it stops. Neither describes a
  // stopped container, so every surface reads no health for one.
  get health(): ContainerHealth | undefined {
    return isStopped(this) ? undefined : this._health;
  }

  get swarmId() {
    return this.labels["com.docker.swarm.task.name"].replace(this.name + ".", "");
  }

  get isSwarm() {
    return Boolean(this.labels["com.docker.swarm.service.id"]);
  }

  public updateStat(stat: Stat) {
    // A Container in the store's reactive `containers` array is reached through a
    // proxy that unwraps every ref on the way out, which is why this used to branch
    // on `isRef` for each field in turn. The raw instance still holds the refs
    // themselves, so take it once and write through them directly.
    const self = toRaw(this);

    const previous = self._sampledStats.value > 0 ? self._stat.value : undefined;
    self._stat.value = stat;

    const history = self._statsHistory.value;
    history.push(stat);
    if (history.length > 300) {
      history.shift();
    }
    // The window is raw and was mutated in place, so nothing has been tracked. This
    // is the entire subscription mechanism for `statsHistory` -- drop it and every
    // chart freezes on its first frame.
    triggerRef(self._statsHistory);

    self._sampledStats.value = Math.min(history.length, self._sampledStats.value + 1);

    // Calculate EMA directly (no watcher needed)
    const alpha = 0.2;
    const prev = self.movingAverageStat.value;
    self.movingAverageStat.value = {
      cpu: alpha * stat.cpu + (1 - alpha) * prev.cpu,
      memory: alpha * stat.memory + (1 - alpha) * prev.memory,
      memoryUsage: alpha * stat.memoryUsage + (1 - alpha) * prev.memoryUsage,
      networkRxTotal: stat.networkRxTotal,
      networkTxTotal: stat.networkTxTotal,
      diskReadTotal: stat.diskReadTotal,
      diskWriteTotal: stat.diskWriteTotal,
    };

    if (previous) {
      const rates = ratesBetween(previous, stat);
      const prevRates = self.movingAverageRateRef.value;
      self.movingAverageRateRef.value = {
        networkRx: alpha * rates.networkRx + (1 - alpha) * prevRates.networkRx,
        networkTx: alpha * rates.networkTx + (1 - alpha) * prevRates.networkTx,
        diskRead: alpha * rates.diskRead + (1 - alpha) * prevRates.diskRead,
        diskWrite: alpha * rates.diskWrite + (1 - alpha) * prevRates.diskWrite,
      };
    }
  }

  public updateMountStats(mountStats: Record<string, MountStat>) {
    this.mountStats = mountStats ?? {};
  }

  static fromJSON(c: ContainerJson): Container {
    return new Container(
      c.id,
      new Date(c.created),
      new Date(c.startedAt),
      new Date(c.finishedAt),
      c.image,
      c.name,
      c.command,
      c.host,
      c.labels,
      c.state,
      c.cpuLimit,
      c.memoryLimit,
      decodeStats(c.stats),
      c.group,
      c.health,
      false,
      c.mounts ?? [],
      c.mountStats ?? {},
      c.ports ?? [],
      c.sizeRw,
      c.volumes ?? [],
    );
  }
}
