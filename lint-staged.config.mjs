import path from "node:path";

export default {
  "*.{js,vue,css,ts,html,md}": ["prettier --write"],
  "*.go": (files) => {
    const dirs = [...new Set(files.map((f) => path.dirname(f)))];
    return [`go fix ${dirs.join(" ")}`, `gofmt -w ${files.join(" ")}`];
  },
};
