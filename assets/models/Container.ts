import type { ContainerStat, ContainerState } from "@/types/Container";
import type { UseRefHistoryRecord, UseThrottledRefHistoryReturn } from "@vueuse/core";
import { ComputedRef, Ref, WritableComputedRef } from "vue";

type Stat = Omit<ContainerStat, "id">;

export class Container {
  private _stat: Ref<Stat> = ref({ cpu: 0, memory: 0, memoryUsage: 0 });
  private _throttledStatHistory: UseThrottledRefHistoryReturn<Stat, Stat>;

  constructor(
    public readonly id: string,
    public readonly created: number,
    public readonly image: string,
    public readonly name: string,
    public readonly command: string,
    public status: string,
    public state: ContainerState
  ) {
    this._throttledStatHistory = useThrottledRefHistory(this._stat, { capacity: 300, deep: true, throttle: 1000 });
  }

  public stat: WritableComputedRef<Stat> = computed({
    get: () => this._stat.value,
    set: (stat) => {
      this._stat.value = stat;
    },
  });

  public statHistory: ComputedRef<UseRefHistoryRecord<Stat>[]> = computed(
    () => this._throttledStatHistory.history.value
  );
}
