import { type Settings } from "@/stores/settings";
import { Host } from "@/stores/hosts";

const text = document.querySelector("script#config__json")?.textContent || "{}";

export interface Config {
  version: string;
  base: string;
  maxLogs: number;
  hostname: string;
  hosts: Host[];
  authProvider: "simple" | "none" | "forward-proxy";
  enableActions: boolean;
  enableShell: boolean;
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
  visibleKeys?: Map<string, Map<string[], boolean>>;
  releaseSeen?: string;
  collapsedGroups?: Set<string>;
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 400,
  version: "v0.0.0",
  hosts: [],
  ...pageConfig,
};

export default Object.freeze(config);

export const withBase = (path: string) => `${config.base}${path}`;
