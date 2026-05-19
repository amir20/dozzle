export interface ContainerStat {
  readonly id: string;
  readonly cpu: number;
  readonly memory: number;
  readonly memoryUsage: number;
  readonly networkRxTotal: number;
  readonly networkTxTotal: number;
  readonly diskReadTotal: number;
  readonly diskWriteTotal: number;
}

export interface ContainerMount {
  readonly type: string;
  readonly source: string;
  readonly destination: string;
  readonly rw: boolean;
}

export interface MountStat {
  readonly destination: string;
  readonly total: number;
  readonly free: number;
  readonly used: number;
  readonly available: boolean;
  readonly lastChecked: string;
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
  readonly mounts?: ContainerMount[];
  readonly mountStats?: Record<string, MountStat>;
  readonly health?: ContainerHealth;
  readonly group?: string;
};

export type ContainerState = "created" | "running" | "exited" | "dead" | "paused" | "restarting" | "deleted";
export type ContainerHealth = "healthy" | "unhealthy" | "starting";
