export interface ContainerStat {
  readonly id: string;
  readonly cpu: number;
  readonly memory: number;
  readonly memoryUsage: number;
}

export type ContainerJson = {
  readonly id: string;
  readonly created: number;
  readonly image: string;
  readonly name: string;
  readonly command: string;
  readonly status: string;
  readonly state: ContainerState;
  readonly health?: ContainerHealth;
};

export type ContainerState = "created" | "running" | "exited" | "dead" | "paused" | "restarting";
export type ContainerHealth = "healthy" | "unhealthy" | "starting";
