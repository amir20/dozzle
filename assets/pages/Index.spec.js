import { shallowMount } from "@vue/test-utils";
import Index from "./Index";

describe("<Index />", () => {
  test("is a Vue instance", () => {
    const wrapper = shallowMount(Index);
    expect(wrapper.isVueInstance()).toBeTruthy();
  });

  test("renders correctly", () => {
    const wrapper = shallowMount(Index);
    expect(wrapper.element).toMatchSnapshot();
  });
});
