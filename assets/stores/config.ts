import { type Settings } from "@/stores/settings";
import { Host } from "@/stores/hosts";

const text = document.querySelector("script#config__json")?.textContent || "{}";

type HostWithoutAvailable = Omit<Host, "available">;

export interface Config {
  version: string;
  base: string;
  maxLogs: number;
  hostname: string;
  hosts: HostWithoutAvailable[];
  authProvider: "simple" | "none" | "forward-proxy";
  enableActions: boolean;
  user?: {
    username: string;
    email: string;
    name: string;
  };
  profile?: Profile;
}

export interface Profile {
  settings?: Settings;
  pinned?: Set<string>;
  visibleKeys?: { [key: string]: string[][] };
  releaseSeen?: string;
  collapsedGroups?: Set<string>;
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 600,
  version: "v0.0.0",
  hosts: [],
  ...pageConfig,
};

config.version = config.version.replace(/^v/, "");

export default Object.freeze(config);

export const withBase = (path: string) => `${config.base}${path}`;
