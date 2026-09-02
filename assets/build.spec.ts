import { build } from "vite";
import { describe, expect, it } from "vitest";

describe("production build", () => {
  it("emits font URLs relative to the generated stylesheet", async () => {
    const result = await build({
      configFile: "vite.config.ts",
      logLevel: "silent",
      build: { write: false },
    });
    const outputs = Array.isArray(result) ? result : [result];
    // Routes are code split, so the build emits several stylesheets. The fonts live in
    // whichever one carries the @font-face rules.
    const stylesheets = outputs
      .flatMap((output) => ("output" in output ? output.output : []))
      .filter((item) => item.type === "asset" && item.fileName.endsWith(".css"))
      .map((item) => String((item as { source: string | Uint8Array }).source));

    expect(stylesheets.length).toBeGreaterThan(0);
    expect(stylesheets.some((css) => /url\(\.\/jetbrains-mono/.test(css))).toBe(true);
    expect(stylesheets.some((css) => /url\(\/assets\//.test(css))).toBe(false);
  }, 15_000);
});
