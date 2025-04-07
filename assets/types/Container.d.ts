export interface ContainerStat {
  readonly id: string;
  readonly cpu: number;
  readonly memory: number;
  readonly memoryUsage: number;
}

export type ContainerJson = {
  readonly id: string;
  readonly created: string;
  readonly startedAt: string;
  readonly finishedAt: string;
  readonly image: string;
  readonly name: string;
  readonly command: string;
  readonly status: string;
  readonly state: ContainerState;
  readonly host: string;
  readonly cpuLimit: number;
  readonly memoryLimit: number;
  readonly labels: Record<string, string>;
  readonly stats: ContainerStat[];
  readonly health?: ContainerHealth;
  readonly group?: string;
};

export type ContainerState = "created" | "running" | "exited" | "dead" | "paused" | "restarting" | "deleted";
export type ContainerHealth = "healthy" | "unhealthy" | "starting";
