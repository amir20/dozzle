import SideDrawer from "@/components/common/SideDrawer.vue";
import { Component } from "vue";

export const drawerContext = Symbol("drawer") as InjectionKey<(c: Component, p: Record<string, any>) => void>;

export const createDrawer = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const component = shallowRef<Component | null>(null);
  const properties = shallowRef<Record<string, any>>({});

  provide(drawerContext, (c: Component, p: Record<string, any>) => {
    component.value = c;
    properties.value = p;
    drawer.value?.open();
  });

  return { component, properties };
};

export const useDrawer = () => inject(drawerContext, () => {});
