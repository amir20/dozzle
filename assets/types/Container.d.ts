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

// The stats history as one array per field, oldest first. memoryUsage and the four
// totals are delta-encoded: the first value is absolute and each one after it is the
// difference from the previous. See statColumns in internal/container/types.go.
export type StatColumns = { readonly [K in Exclude<keyof ContainerStat, "id">]: number[] };

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

export interface VolumeUsage {
  readonly name: string;
  readonly destination: string;
  readonly size: number;
  // containers using the volume, this one included
  readonly links: number;
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
  readonly stats?: StatColumns;
  readonly mounts?: ContainerMount[];
  readonly ports?: string[];
  readonly mountStats?: Record<string, MountStat>;
  // bytes in the writable layer; absent until the server has measured it
  readonly sizeRw?: number;
  // Docker-managed volumes the container mounts; absent until measured or when it has none
  readonly volumes?: VolumeUsage[];
  readonly health?: ContainerHealth;
  readonly group?: string;
};

export type ContainerState = "created" | "running" | "exited" | "dead" | "paused" | "restarting" | "deleted";
export type ContainerHealth = "healthy" | "unhealthy" | "starting";
