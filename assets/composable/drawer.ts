import SideDrawer from "@/components/common/SideDrawer.vue";
import { Component } from "vue";

export const drawerContext = Symbol("drawer") as InjectionKey<(c: Component, p: Record<string, any>) => void>;

export const createDrawer = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const component = shallowRef<Component | null>(null);
  const properties = shallowRef<Record<string, any>>({});
  const showDrawer = (c: Component, p: Record<string, any>) => {
    component.value = c;
    properties.value = p;
    drawer.value?.open();
  };

  provide(drawerContext, showDrawer);

  return { component, properties, showDrawer };
};

export const useDrawer = () =>
  inject(drawerContext, () => {
    console.error("No drawer context provided");
  });
