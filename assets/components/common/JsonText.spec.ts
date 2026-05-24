/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import JsonText from "./JsonText.vue";

function mountText(text: string, highlight?: string) {
  return mount(JsonText, { props: { text, highlight } });
}

describe("<JsonText />", () => {
  test("renders plain text without a highlight", () => {
    const wrapper = mountText("hello");
    expect(wrapper.text()).toBe("hello");
    expect(wrapper.find("mark").exists()).toBe(false);
  });

  test("marks the matching substring", () => {
    const wrapper = mountText("hello", "ell");
    const marks = wrapper.findAll("mark");
    expect(marks).toHaveLength(1);
    expect(marks[0].text()).toBe("ell");
    expect(wrapper.text()).toBe("hello");
  });

  test("matches case-insensitively", () => {
    const wrapper = mountText("Hello", "ELL");
    expect(wrapper.find("mark").text()).toBe("ell");
  });

  test("treats regex metacharacters literally", () => {
    // Without escaping, "." would match every character; escaped it matches only dots.
    const wrapper = mountText("a.b.c", ".");
    const marks = wrapper.findAll("mark");
    expect(marks).toHaveLength(2);
    expect(marks.every((m) => m.text() === ".")).toBe(true);
  });

  test("marks every occurrence", () => {
    const wrapper = mountText("xax", "x");
    expect(wrapper.findAll("mark")).toHaveLength(2);
  });
});
