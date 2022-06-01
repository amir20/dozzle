export interface Container {
  readonly id: string;
  readonly created: number;
  readonly image: string;
  readonly name: string;
  readonly status: string;
  readonly command: string;
  state: "created" | "running" | "exited" | "dead" | "paused" | "restarting";
  stat?: ContainerStat;
}

export interface ContainerStat {
  readonly id: string;
  readonly cpu: number;
  readonly memory: number;
  readonly memoryUsage: number;
}
