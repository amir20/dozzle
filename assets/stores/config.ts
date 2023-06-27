const text = document.querySelector("script#config__json")?.textContent || "{}";

interface Config {
  version: string;
  base: string;
  authorizationNeeded: boolean;
  secured: boolean;
  maxLogs: number;
  hostname: string;
  hosts: string[];
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 600,
  ...pageConfig,
};

config.version = config.version.replace(/^v/, "");

export default config;
