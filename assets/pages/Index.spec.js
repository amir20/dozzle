import { shallowMount } from "@vue/test-utils";
import Index from "./Index";

describe("<Index />", () => {
  test("renders correctly", () => {
    const wrapper = shallowMount(Index);
    expect(wrapper.element).toMatchSnapshot();
  });
});
