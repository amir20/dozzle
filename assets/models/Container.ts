import type { ContainerHealth, ContainerStat, ContainerState } from "@/types/Container";
import { useExponentialMovingAverage, useSimpleRefHistory } from "@/utils";
import { Ref } from "vue";

export type Stat = Omit<ContainerStat, "id">;

const SWARM_ID_REGEX = /(\.[a-z0-9]{25})+$/i;

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

export class Container {
  private _stat: Ref<Stat>;
  private readonly _statsHistory: Ref<Stat[]>;
  public readonly swarmId: string | null = null;
  public readonly isSwarm: boolean = false;
  private readonly movingAverageStat: Ref<Stat>;

  constructor(
    public readonly id: string,
    public readonly created: Date,
    public readonly image: string,
    public readonly name: string,
    public readonly command: string,
    public readonly host: string,
    public readonly labels = {} as Record<string, string>,
    public status: string,
    public state: ContainerState,
    stats: Stat[],
    public readonly group?: string,
    public health?: ContainerHealth,
  ) {
    this._stat = ref(stats.at(-1) || ({ cpu: 0, memory: 0, memoryUsage: 0 } as Stat));
    this._statsHistory = useSimpleRefHistory(this._stat, { capacity: 300, deep: true, initial: stats });
    this.movingAverageStat = useExponentialMovingAverage(this._stat, 0.2);

    const match = name.match(SWARM_ID_REGEX);
    if (match) {
      this.swarmId = match[0];
      this.name = name.replace(`${this.swarmId}`, "");
      this.isSwarm = true;
    }
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

  public updateStat(stat: Stat) {
    if (isRef(this._stat)) {
      this._stat.value = stat;
    } else {
      // @ts-ignore
      this._stat = stat;
    }
  }
}
