const text = document.querySelector("script#config__json")?.textContent || "{}";

interface Config {
  version: string;
  base: string;
  authorizationNeeded: boolean | "false" | "true";
  secured: boolean | "false" | "true";
  maxLogs: number;
  hostname: string;
  hosts: string[] | string;
}

const pageConfig = JSON.parse(text);

const config: Config = {
  maxLogs: 600,
  ...pageConfig,
};

if (config.version == "{{ .Version }}") {
  config.version = "master";
  config.base = "";
  config.authorizationNeeded = false;
  config.secured = false;
  config.hostname = "localhost";
  config.hosts = ["localhost"];
} else {
  config.version = config.version.replace(/^v/, "");
  config.authorizationNeeded = config.authorizationNeeded === "true";
  config.secured = config.secured === "true";
  config.hosts = (config.hosts as string).split(",");
}

export default config as Config;
