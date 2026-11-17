import { describe, expect, it } from "vitest";
import { isQuestionLike } from "./questionLike";

describe("isQuestionLike", () => {
  it.each([
    "why is postgres slow",
    "Which containers logged errors in the last hour",
    "anything new since the deploy",
    "is redis ok?",
    "  what changed today  ",
  ])("treats %j as a question", (q) => {
    expect(isQuestionLike(q)).toBe(true);
  });

  it.each(["", "postgres", "why", "why postgres", "doligence_api.1", "api errors today", "redis?"])(
    "treats %j as a search",
    (q) => {
      expect(isQuestionLike(q)).toBe(false);
    },
  );
});
