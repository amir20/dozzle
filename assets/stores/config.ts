import { type Settings } from "@/stores/settings";
import { Host } from "@/stores/hosts";
import type { CloudConfig } from "@/types/notifications";

const text = document.querySelector("script#config__json")?.textContent || "{}";

export interface Config {
  version: string;
  base: string;
  maxLogs: number;
  hostname: string;
  mode: "server" | "swarm" | "k8s";
  hosts: Host[];
  authProvider: "simple" | "none" | "forward-proxy" | "oidc";
  oauthProviders?: { name: string; loginUrl: string; icon: string }[];
  passwordLogin?: boolean;
  logoutUrl?: string;
  enableActions: boolean;
  enableShell: boolean;
  enableDownload: boolean;
  enableNotifications: boolean;
  enableCloud: boolean;
  canLinkCloud: boolean;
  dataPersisted?: boolean;
  // Full id of the container this Dozzle runs in, absent when it cannot tell.
  selfContainerId?: string;
  cloudUrl: string;
  // Null when this instance is not linked. Inlined so the page does not spend a
  // round trip asking, since everything else cloud-shaped waits on `linked`.
  cloudConfig?: CloudConfig | null;
  disableAvatars: boolean;
  releaseCheckMode: "automatic" | "manual";
  imageCheckMode: "automatic" | "manual" | "off";
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
  collapsedHostGroups?: Set<string>;
  cloudWelcomeShown?: boolean;
  dismissedImageUpdates?: Set<string>;
  dismissedLinkHint?: boolean;
  lastSeenAlertTs?: number;
  setupSeen?: boolean;
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 400,
  version: "v0.0.0",
  hosts: [],
  // The login page ships a config without the authorized keys, and unit tests
  // render components with no injected config at all.
  cloudUrl: "https://cloud.dozzle.dev",
  ...pageConfig,
};

export default Object.freeze(config);

export const withBase = (path: string) => `${config.base}${path}`;
