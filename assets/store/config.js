const config = JSON.parse(document.querySelector("script#config__json").textContent);
if (config.version == "{{ .Version }}") {
  config.version = "dev";
  config.base = "";
  config.authorizationNeeded = false;
} else {
  config.version = config.version.replace(/^v/, "");
  config.authorizationNeeded = config.authorizationNeeded === "true";
}

export default config;
