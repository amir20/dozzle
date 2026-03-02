import type { ContainerHealth, ContainerJson, ContainerStat, ContainerState } from "@/types/Container";
import { Ref } from "vue";

export type Stat = Omit<ContainerStat, "id">;

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
  ) {
    const defaultStat = { cpu: 0, memory: 0, memoryUsage: 0, networkRxTotal: 0, networkTxTotal: 0 } as Stat;
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
    };
    if (isRef(this.movingAverageStat)) {
      this.movingAverageStat.value = newEma;
    } else {
      (this.movingAverageStat as unknown as Stat) = newEma;
    }
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
    );
  }
}
