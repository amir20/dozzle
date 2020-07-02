const config = JSON.parse(document.querySelector("script#config__json").textContent);
if (config.version == "{{ .Version }}") {
  config.version = "dev";
  config.base = "";
}
export default config;
