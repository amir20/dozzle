import { describe, expect, it } from "vitest";
import { formatDuration, readFields, repeatKey } from "./readable";

describe("readFields", () => {
  it("leads with the message and leaves everything else alone", () => {
    const r = readFields({ time: "x", level: "info", msg: "order processed", order_id: 1 });
    expect(r.title).toBe("order processed");
    expect([...r.used]).toEqual(["msg"]);
  });

  it("reads a request: method, path, status and duration in ms", () => {
    const r = readFields({
      "http.request.method": "POST",
      "url.path": "/api/feedback",
      "http.response.status_code": 201,
      "event.duration": 5039703,
      "client.ip": "1.2.3.4",
    });
    expect([r.method, r.path, r.status]).toEqual(["POST", "/api/feedback", 201]);
    expect(r.durationMs).toBeCloseTo(5.04, 2);
    expect(r.used.has("client.ip")).toBe(false);
  });

  it("leaves path and duration as fields on a line that isn't a request", () => {
    const r = readFields({ message: "retro review done", path: "/x", took: 2923 });
    expect(r.path).toBeUndefined();
    expect(r.durationMs).toBeUndefined();
    expect([...r.used]).toEqual(["message"]);
  });

  it("notices a message that only restates the request", () => {
    const r = readFields({
      msg: "POST /api/events ⇒ HTTP 400 (472ms)",
      method: "POST",
      path: "/api/events",
      status: 400,
    });
    expect(r.titleRepeatsRequest).toBe(true);
    expect(readFields({ msg: "order created", method: "POST", path: "/api/orders" }).titleRepeatsRequest).toBe(false);
  });

  it("lifts the error", () => {
    expect(readFields({ msg: "payment failed", error: "card declined" }).error).toBe("card declined");
  });
});

describe("repeatKey", () => {
  it("is the same for lines that say the same thing whatever their other fields", () => {
    expect(repeatKey({ msg: "order processed", order_id: 1 })).toBe(repeatKey({ msg: "order processed", order_id: 2 }));
  });

  it("keeps a failing request apart from successful ones with the same message", () => {
    expect(repeatKey({ msg: "request completed", status: 200 })).not.toBe(
      repeatKey({ msg: "request completed", status: 500 }),
    );
  });

  it("is undefined for a line with no message or error, so it never folds", () => {
    expect(repeatKey({ method: "GET", path: "/" })).toBeUndefined();
  });
});

describe("formatDuration", () => {
  it("picks a readable unit", () => {
    expect(formatDuration(3.4)).toBe("3.4ms");
    expect(formatDuration(42.4)).toBe("42ms");
    expect(formatDuration(1532)).toBe("1.5s");
  });
});
