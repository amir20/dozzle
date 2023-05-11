import { describe, expect, test } from "vitest";
import { Container } from "./Container";
describe("Container", () => {
  const names = [
    [
      "foo.gb1cto7gaq68fp4refnsr5hep.byqr1prci82zyfoos6gx1yhz0",
      "foo",
      ".gb1cto7gaq68fp4refnsr5hep.byqr1prci82zyfoos6gx1yhz0",
    ],
    ["bar.gb1cto7gaq68fp4refnsr5hep", "bar", ".gb1cto7gaq68fp4refnsr5hep"],
    ["baz", "baz", null],
  ];

  test.each(names)("name %s should be %s and %s", (name, expectedName, expectedSwarmId) => {
    const c = new Container("id", new Date(), "image", name!, "command", "status", "created");
    expect(c.name).toBe(expectedName);
    expect(c.swarmId).toBe(expectedSwarmId);
  });
});
