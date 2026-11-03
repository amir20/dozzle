// The API's container types are generated from Go by tygo (tygo.yaml). This file
// holds only what Go cannot describe, plus the names the frontend already imports.
import type { Container, ContainerStat, Mount } from "./generated/container";

export type { ContainerStat, MountStat, VolumeUsage, ContainerNetwork, Network, Volume } from "./generated/container";
export type { ContainerInspect } from "./generated/web";

export type ContainerJson = Readonly<Container>;
export type ContainerMount = Readonly<Mount>;

// The stats history as one array per field, oldest first. memoryUsage and the four
// totals are delta-encoded: the first value is absolute and each one after it is the
// difference from the previous. See statColumns in internal/container/types.go.
export type StatColumns = { readonly [K in Exclude<keyof ContainerStat, "id">]: number[] };

export type ContainerState = "created" | "running" | "exited" | "dead" | "paused" | "restarting" | "deleted";
export type ContainerHealth = "healthy" | "unhealthy" | "starting";
