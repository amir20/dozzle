import SideDrawer from "@/components/common/SideDrawer.vue";
import { Component } from "vue";

export type DrawerSize = "md" | "xl" | "lg";

export const drawerContext = Symbol("drawer") as InjectionKey<
  (c: Component, p: Record<string, any>, s?: DrawerSize) => void
>;

export const createDrawer = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const component = shallowRef<Component | null>(null);
  const properties = shallowRef<Record<string, any>>({});
  const size = ref<DrawerSize>("md");
  const showDrawer = (c: Component, p: Record<string, any>, s: DrawerSize = "md") => {
    component.value = c;
    properties.value = p;
    size.value = s;
    drawer.value?.open();
  };

  provide(drawerContext, showDrawer);

  return { component, properties, showDrawer, size };
};

export const useDrawer = () =>
  inject(drawerContext, () => {
    console.error("No drawer context provided");
  });
