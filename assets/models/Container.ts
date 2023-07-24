import type { ContainerHealth, ContainerStat, ContainerState } from "@/types/Container";
import type { UseThrottledRefHistoryReturn } from "@vueuse/core";
import { useExponentialMovingAverage } from "@/utils";
import { Ref } from "vue";

type Stat = Omit<ContainerStat, "id">;

const SWARM_ID_REGEX = /(\.[a-z0-9]{25})+$/i;

export class Container {
  public stat: Ref<Stat>;
  private readonly throttledStatHistory: UseThrottledRefHistoryReturn<Stat, Stat>;
  public readonly swarmId: string | null = null;
  public readonly isSwarm: boolean = false;
  public readonly movingAverageStat: Ref<Stat>;

  constructor(
    public readonly id: string,
    public readonly created: Date,
    public readonly image: string,
    public readonly name: string,
    public readonly command: string,
    public readonly host: string,
    public status: string,
    public state: ContainerState,
    public health?: ContainerHealth,
  ) {
    this.stat = ref({ cpu: 0, memory: 0, memoryUsage: 0 });
    this.throttledStatHistory = useThrottledRefHistory(this.stat, { capacity: 300, deep: true, throttle: 1000 });
    this.movingAverageStat = useExponentialMovingAverage(this.stat, 0.2);

    const match = name.match(SWARM_ID_REGEX);
    if (match) {
      this.swarmId = match[0];
      this.name = name.replace(`${this.swarmId}`, "");
      this.isSwarm = true;
    }
  }

  public getStatHistory() {
    return unref(this.throttledStatHistory.history);
  }

  public getLastStat() {
    return unref(this.throttledStatHistory.last);
  }
}
