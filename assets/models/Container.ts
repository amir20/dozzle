import type { ContainerHealth, ContainerStat, ContainerState } from "@/types/Container";
import { useExponentialMovingAverage, useSimpleRefHistory } from "@/utils";
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
  ) {
    this._stat = ref(stats.at(-1) || ({ cpu: 0, memory: 0, memoryUsage: 0 } as Stat));
    const { history } = useSimpleRefHistory(this._stat, { capacity: 300, deep: true, initial: stats });
    this._statsHistory = history;
    this.movingAverageStat = useExponentialMovingAverage(this._stat, 0.2);

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
    return this.labels["com.docker.stack.namespace"] || this.labels["com.docker.compose.project"];
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
    if (isRef(this._stat)) {
      this._stat.value = stat;
    } else {
      // @ts-ignore
      this._stat = stat;
    }
  }
}
