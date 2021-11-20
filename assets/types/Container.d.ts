export interface Container {
  readonly id: string;
  readonly created: number;
  readonly image: string;
  readonly name: string;
  readonly state: string;
  readonly status: string;
  stat: ContainerStat;
}

export interface ContainerStat {
  readonly cpu: number;
  readonly memory: number;
  readonly memoryUsage: number;
}
