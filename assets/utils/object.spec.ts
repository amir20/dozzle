import { describe, expect, test } from "vitest";
import { flattenJSON, getDeep, isObject, arrayEquals } from "./object";

describe("flattenJSON", () => {
  test("flattens nested objects to dot keys", () => {
    expect(flattenJSON({ a: { b: 1 } })).toEqual({ "a.b": 1 });
    expect(flattenJSON({ a: { b: { c: 2 } } })).toEqual({ "a.b.c": 2 });
  });

  test("keeps top-level primitives", () => {
    expect(flattenJSON({ a: 1, b: "x" })).toEqual({ a: 1, b: "x" });
  });

  test("keeps arrays as values (not recursed)", () => {
    expect(flattenJSON({ a: [1, 2] })).toEqual({ a: [1, 2] });
  });

  test("empty object", () => {
    expect(flattenJSON({})).toEqual({});
  });
});

describe("getDeep", () => {
  test("reads a nested path", () => {
    expect(getDeep({ a: { b: { c: 5 } } }, ["a", "b", "c"])).toBe(5);
  });

  test("returns undefined for a missing path without throwing", () => {
    expect(getDeep({ a: {} }, ["a", "b", "c"])).toBeUndefined();
    expect(getDeep({}, ["x"])).toBeUndefined();
  });
});

describe("isObject", () => {
  test("true for plain objects", () => {
    expect(isObject({})).toBe(true);
  });

  test("false for arrays, null, and primitives", () => {
    expect(isObject([])).toBe(false);
    expect(isObject(null)).toBe(false);
    expect(isObject(5)).toBe(false);
    expect(isObject("s")).toBe(false);
  });
});

describe("arrayEquals", () => {
  test("equal arrays", () => {
    expect(arrayEquals(["a", "b"], ["a", "b"])).toBe(true);
    expect(arrayEquals([], [])).toBe(true);
  });

  test("different length or order", () => {
    expect(arrayEquals(["a"], ["a", "b"])).toBe(false);
    expect(arrayEquals(["a", "b"], ["b", "a"])).toBe(false);
  });
});
