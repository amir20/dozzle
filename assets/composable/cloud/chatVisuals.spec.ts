import { describe, expect, test, vi } from "vitest";

// The words are the locale's business; these tests check which facts are said.
vi.mock("@/modules/i18n", () => ({
  i18n: {
    global: {
      t: (key: string, params?: Record<string, unknown>) =>
        `${key.replace("cloud-chat.visual.", "")}${params ? JSON.stringify(params) : ""}`,
    },
  },
}));

const { cpuBars, cpuPeak, factsLine, memoryBars } = await import("./chatVisuals");

describe("factsLine", () => {
  // Rook: 24 SIGKILL-heavy exits and no out-of-memory event stored.
  test("lists every exit code with its count, and that no OOM kill was recorded", () => {
    const line = factsLine({
      exits: 24,
      exitCodes: [
        { code: "137", count: 17 },
        { code: "1", count: 7 },
      ],
      ooms: 0,
      restarts: 0,
      eventsRecorded: true,
      capped: false,
    });
    expect(line).toBe('exits{"n":24}, exit-codes{"codes":"137 (17), 1 (7)"} · no-oom');
  });

  test("a single code needs no count", () => {
    const line = factsLine({
      exits: 2,
      exitCodes: [{ code: "137", count: 2 }],
      ooms: 0,
      restarts: 0,
      eventsRecorded: true,
      capped: false,
    });
    expect(line).toBe('exits{"n":2}, exit-codes{"codes":"137"} · no-oom');
  });

  test("never claims no OOM kill when events are not stored or were cut off", () => {
    const base = { exits: 3, ooms: 0, restarts: 0 };
    expect(factsLine({ ...base, eventsRecorded: false, capped: false })).not.toContain("no-oom");
    expect(factsLine({ ...base, eventsRecorded: true, capped: true })).not.toContain("no-oom");
  });

  test("says OOM kills and restarts when there were some", () => {
    expect(factsLine({ exits: 0, ooms: 1, restarts: 4, eventsRecorded: true, capped: false })).toBe(
      'ooms{"n":1} · restarts{"n":4}',
    );
  });
});

describe("memoryBars", () => {
  const GiB = 1024 ** 3;
  test("with a limit, bars are a share of it and the peak keeps its own height", () => {
    const [bar] = memoryBars([{ ts: 0, memory: 0, memoryUsage: 8 * GiB, memoryUsageMax: 12 * GiB }], 16 * GiB);
    expect(bar).toEqual({ percent: 50, value: 8 * GiB, peak: 12 * GiB, peakPercent: 75 });
  });

  test("without a limit, bars are bytes so the chart scales to its own peak", () => {
    const [bar] = memoryBars([{ ts: 0, memory: 0, memoryUsage: 300 }], 0);
    expect(bar).toEqual({ percent: 300, value: 300, peak: 300, peakPercent: 300 });
  });
});

describe("cpuBars", () => {
  test("bars are the average and the peak is the bucket's highest sample", () => {
    expect(cpuBars([{ ts: 1, cpu: 12.5, cpuMax: 87.5, memory: 0, memoryUsage: 0 }])).toEqual([
      { percent: 12.5, value: 12.5, peak: 87.5, peakPercent: 87.5 },
    ]);
  });

  test("a cloud that sends no max peaks at the average", () => {
    expect(cpuBars([{ ts: 1, cpu: 4, memory: 0, memoryUsage: 0 }])[0].peak).toBe(4);
  });
});

describe("cpuPeak", () => {
  test("is the highest sample across the window", () => {
    expect(
      cpuPeak([
        { ts: 1, cpu: 0.1, cpuMax: 2.1, memory: 0, memoryUsage: 0 },
        { ts: 2, cpu: 0.2, memory: 0, memoryUsage: 0 },
      ]),
    ).toBe(2.1);
  });

  test("is undefined with no series", () => {
    expect(cpuPeak([])).toBeUndefined();
  });
});
