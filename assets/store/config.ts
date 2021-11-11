const text = document.querySelector("script#config__json")?.textContent || "{}";

const config = JSON.parse(text);
if (config.version == "{{ .Version }}") {
  config.version = "master";
  config.base = "";
  config.authorizationNeeded = false;
  config.secured = false;
} else {
  config.version = config.version.replace(/^v/, "");
  config.authorizationNeeded = config.authorizationNeeded === "true";
  config.secured = config.secured === "true";
}
export default config;
