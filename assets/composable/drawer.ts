import SideDrawer from "@/components/common/SideDrawer.vue";
import { Component } from "vue";

export type DrawerWidth = "md" | "xl" | "lg";

/** Returns true when the drawer may close. A guard that returns false owns telling the user why. */
export type DrawerCloseGuard = () => boolean;

export const drawerContext = Symbol("drawer") as InjectionKey<
  (c: Component, p: Record<string, any>, s?: DrawerWidth) => void
>;

interface DrawerGuardApi {
  set: (guard: DrawerCloseGuard | null) => void;
  current: () => DrawerCloseGuard | null;
}

export const drawerGuardContext = Symbol("drawer-guard") as InjectionKey<DrawerGuardApi>;

export const createDrawer = (drawer: Ref<InstanceType<typeof SideDrawer>>) => {
  const component = shallowRef<Component | null>(null);
  const properties = shallowRef<Record<string, any>>({});
  const width = ref<DrawerWidth>("md");
  const closeGuard = shallowRef<DrawerCloseGuard | null>(null);
  const showDrawer = (c: Component, p: Record<string, any>, w: DrawerWidth = "md") => {
    // A new occupant never inherits the previous one's guard.
    closeGuard.value = null;
    component.value = c;
    properties.value = p;
    width.value = w;
    drawer.value?.open();
  };

  provide(drawerContext, showDrawer);
  provide(drawerGuardContext, {
    set: (guard: DrawerCloseGuard | null) => (closeGuard.value = guard),
    current: () => closeGuard.value,
  });

  return { component, properties, showDrawer, width, closeGuard };
};

export const useDrawer = () =>
  inject(drawerContext, () => {
    console.error("No drawer context provided");
  });

/**
 * Lets drawer content veto a close (Esc, backdrop click, the close button) — for example to
 * confirm discarding an unsaved form. The guard is dropped when the component goes away.
 */
export const useDrawerCloseGuard = () => {
  const api = inject(drawerGuardContext, { set: () => {}, current: () => null });
  let mine: DrawerCloseGuard | null = null;

  const register = (guard: DrawerCloseGuard | null) => {
    mine = guard;
    api.set(guard);
  };

  // Unmounting after the drawer's next occupant has already registered its own guard must
  // not clear that guard, so only retract one that is still ours.
  onScopeDispose(() => {
    if (mine && api.current() === mine) api.set(null);
    mine = null;
  });

  return register;
};
