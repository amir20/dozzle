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
    const stylesheet = outputs
      .flatMap((output) => ("output" in output ? output.output : []))
      .find((item) => item.type === "asset" && item.fileName.endsWith(".css"));

    expect(stylesheet).toBeDefined();
    if (!stylesheet || stylesheet.type !== "asset") {
      throw new Error("Production build did not emit a stylesheet");
    }
    const css = String(stylesheet.source);
    expect(css).toMatch(/url\(\.\/jetbrains-mono/);
    expect(css).not.toMatch(/url\(\/assets\/jetbrains-mono/);
  }, 15_000);
});
