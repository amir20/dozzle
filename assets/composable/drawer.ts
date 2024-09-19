import SideDrawer from "@/components/common/SideDrawer.vue";
import { Component } from "vue";

export type DrawerWidth = "md" | "xl" | "lg";

export const drawerContext = Symbol("drawer") as InjectionKey<
  (c: Component, p: Record<string, any>, s?: DrawerWidth) => void
>;

export const createDrawer = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const component = shallowRef<Component | null>(null);
  const properties = shallowRef<Record<string, any>>({});
  const width = ref<DrawerWidth>("md");
  const showDrawer = (c: Component, p: Record<string, any>, w: DrawerWidth = "md") => {
    component.value = c;
    properties.value = p;
    width.value = w;
    drawer.value?.open();
  };

  provide(drawerContext, showDrawer);

  return { component, properties, showDrawer, width };
};

export const useDrawer = () =>
  inject(drawerContext, () => {
    console.error("No drawer context provided");
  });
