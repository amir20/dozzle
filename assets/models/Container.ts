import type {
  ContainerHealth,
  ContainerJson,
  ContainerMount,
  ContainerStat,
  ContainerState,
  MountStat,
} from "@/types/Container";
import { Ref } from "vue";

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

const hosts = computed(() =>
  config.hosts.reduce(
    (acc, item) => {
      acc[item.id] = item;
      return acc;
    },
    {} as Record<string, { name: string; id: string }>,
  ),
);

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
  ) {}
}

export class Container {
  private _stat: Ref<Stat>;
  private _name: string;
  private readonly _statsHistory: Ref<Stat[]>;
  private readonly movingAverageStat: Ref<Stat>;

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
    public readonly group?: string,
    public health?: ContainerHealth,
    public isNew: boolean = false,
    mounts: ContainerMount[] = [],
    mountStats: Record<string, MountStat> = {},
    public readonly ports: string[] = [],
  ) {
    this.mounts = mounts;
    this.mountStats = mountStats;
    const defaultStat = emptyStat();
    this._stat = ref(stats.at(-1) || defaultStat);
    const recentStats = stats.slice(-300);
    const padding = Array(300 - recentStats.length).fill(defaultStat);
    this._statsHistory = ref([...padding, ...recentStats]);
    this.movingAverageStat = ref(stats.at(-1) || defaultStat);

    this._name = name;
  }

  get statsHistory() {
    return unref(this._statsHistory);
  }

  get movingAverage() {
    return unref(this.movingAverageStat);
  }

  get stat() {
    return unref(this._stat);
  }

  get hostLabel() {
    return hosts.value[this.host]?.name;
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
   * overrides the guess, and `none` opts a container out when the guess is wrong.
   */
  get icon() {
    const override = this.labels["dev.dozzle.icon"]?.trim().toLowerCase();
    if (override) return override === "none" || !hasIcon(override) ? undefined : override;
    return iconSlugForImage(this.image);
  }

  /**
   * Opt-in link to whatever web UI this container serves, from `dev.dozzle.url`.
   * Only absolute http(s) URLs are honored so a label can't smuggle in `javascript:`.
   */
  get url() {
    const raw = this.labels["dev.dozzle.url"]?.trim();
    if (!raw) return undefined;
    try {
      const { protocol } = new URL(raw);
      return protocol === "http:" || protocol === "https:" ? raw : undefined;
    } catch {
      return undefined;
    }
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

  get swarmId() {
    return this.labels["com.docker.swarm.task.name"].replace(this.name + ".", "");
  }

  get isSwarm() {
    return Boolean(this.labels["com.docker.swarm.service.id"]);
  }

  public updateStat(stat: Stat) {
    // When Container is inside a reactive array, refs get unwrapped
    if (isRef(this._stat)) {
      this._stat.value = stat;
    } else {
      (this._stat as unknown as Stat) = stat;
    }

    // Update history directly (no watcher needed)
    const history = isRef(this._statsHistory) ? this._statsHistory.value : (this._statsHistory as unknown as Stat[]);
    history.push(stat);
    if (history.length > 300) {
      history.shift();
    }

    // Calculate EMA directly (no watcher needed)
    const alpha = 0.2;
    const prev = isRef(this.movingAverageStat)
      ? this.movingAverageStat.value
      : (this.movingAverageStat as unknown as Stat);
    const newEma = {
      cpu: alpha * stat.cpu + (1 - alpha) * prev.cpu,
      memory: alpha * stat.memory + (1 - alpha) * prev.memory,
      memoryUsage: alpha * stat.memoryUsage + (1 - alpha) * prev.memoryUsage,
      networkRxTotal: stat.networkRxTotal,
      networkTxTotal: stat.networkTxTotal,
      diskReadTotal: stat.diskReadTotal,
      diskWriteTotal: stat.diskWriteTotal,
    };
    if (isRef(this.movingAverageStat)) {
      this.movingAverageStat.value = newEma;
    } else {
      (this.movingAverageStat as unknown as Stat) = newEma;
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
      c.stats ?? [],
      c.group,
      c.health,
      false,
      c.mounts ?? [],
      c.mountStats ?? {},
      c.ports ?? [],
    );
  }
}
