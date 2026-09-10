/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import JsonValue from "./JsonValue.vue";

// indent < 0 forces the compact inline rendering, which is deterministic to assert.
function inline(value: unknown, highlight?: string) {
  return mount(JsonValue, { props: { value, indent: -1, highlight } });
}

// wrapper.text() drops inter-node whitespace, so compare structure without spaces.
function compact(wrapper: ReturnType<typeof inline>) {
  return wrapper.text().replace(/\s+/g, "");
}

describe("<JsonValue /> primitives", () => {
  test("null", () => {
    const wrapper = inline(null);
    expect(wrapper.text()).toBe("null");
    expect(wrapper.find(".json-null").exists()).toBe(true);
  });

  test("boolean", () => {
    expect(inline(true).text()).toBe("true");
    expect(inline(false).find(".json-boolean").exists()).toBe(true);
  });

  test("number", () => {
    const wrapper = inline(42);
    expect(wrapper.text()).toBe("42");
    expect(wrapper.find(".json-number").exists()).toBe(true);
  });

  test("string is quoted", () => {
    const wrapper = inline("hi");
    expect(wrapper.text()).toBe('"hi"');
    expect(wrapper.find(".json-string").exists()).toBe(true);
  });
});

describe("<JsonValue /> structures", () => {
  test("empty array and object", () => {
    expect(inline([]).text()).toBe("[]");
    expect(inline({}).text()).toBe("{}");
  });

  test("flat array", () => {
    expect(compact(inline([1, 2]))).toBe("[1,2]");
  });

  test("flat object", () => {
    expect(compact(inline({ a: 1 }))).toBe('{"a":1}');
  });

  test("nested object", () => {
    expect(compact(inline({ a: { b: 1 } }))).toBe('{"a":{"b":1}}');
  });

  test("indent mode emits newline spans, inline mode does not", () => {
    const indented = mount(JsonValue, { props: { value: { a: 1 }, indent: 0 } });
    expect(indented.findAll(".json-newline").length).toBeGreaterThan(0);
    expect(inline({ a: 1 }).findAll(".json-newline")).toHaveLength(0);
  });
});

describe("<JsonValue /> highlight", () => {
  test("passes highlight down to string values", () => {
    const wrapper = inline("hello", "ell");
    expect(wrapper.find("mark").text()).toBe("ell");
  });
});
