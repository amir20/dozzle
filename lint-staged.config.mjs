import path from "node:path";

export default {
  // eslint first, prettier last: --fix rewrites classes (`min-w-[4px]` to
  // `min-w-1`) and prettier then re-sorts and re-wraps whatever it produced.
  // Kept as one entry rather than two globs so the two tools never run
  // concurrently on the same file.
  //
  // --no-warn-ignored because this glob matches generated files the config
  // ignores (auto-imports.d.ts and friends), and a staged one would otherwise
  // fail the commit for being ignored.
  // Same extensions eslint.config.js lints. mjs matters: this file is one.
  "*.{js,mjs,ts,mts,vue}": ["eslint --fix --no-warn-ignored", "prettier --write"],
  "*.{css,html,md}": ["prettier --write"],
  "*.go": (files) => {
    const dirs = [...new Set(files.map((f) => path.dirname(f)))];
    return [`go fix ${dirs.join(" ")}`, `gofmt -w ${files.join(" ")}`];
  },
};
