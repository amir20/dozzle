import type { ContainerStat, ContainerState } from "@/types/Container";
import type { UseThrottledRefHistoryReturn } from "@vueuse/core";
import { Ref } from "vue";

type Stat = Omit<ContainerStat, "id">;

export class Container {
  public stat: Ref<Stat>;
  private readonly throttledStatHistory: UseThrottledRefHistoryReturn<Stat, Stat>;

  constructor(
    public readonly id: string,
    public readonly created: number,
    public readonly image: string,
    public readonly name: string,
    public readonly command: string,
    public status: string,
    public state: ContainerState
  ) {
    this.stat = ref({ cpu: 0, memory: 0, memoryUsage: 0 });
    this.throttledStatHistory = useThrottledRefHistory(this.stat, { capacity: 300, deep: true, throttle: 1000 });
  }

  public getStatHistory() {
    return unref(this.throttledStatHistory.history);
  }

  public getLastStat() {
    return unref(this.throttledStatHistory.last);
  }
}
