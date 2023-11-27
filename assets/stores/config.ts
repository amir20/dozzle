import { type Settings } from "@/stores/settings";

const text = document.querySelector("script#config__json")?.textContent || "{}";

export interface Config {
  version: string;
  base: string;
  authorizationNeeded: boolean;
  secured: boolean;
  maxLogs: number;
  hostname: string;
  hosts: { name: string; id: string }[];
  authProvider: "simple" | "none" | "forward-proxy";
  user?: {
    username: string;
    email: string;
    name: string;
    avatar: string;
  };
  profile?: Profile;
  pages?: { id: string; title: string }[];
}

export interface Profile {
  settings?: Settings;
  pinned?: string[];
  visibleKeys?: { [key: string]: string[][] };
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 600,
  ...pageConfig,
};

config.version = config.version.replace(/^v/, "");

export default Object.freeze(config);

export const withBase = (path: string) => `${config.base}${path}`;
