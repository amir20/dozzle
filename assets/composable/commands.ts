import type { Component } from "vue";
import { Container } from "@/models/Container";
import { useContainerActions } from "@/composable/containerActions";
import config from "@/stores/config";
import {
  lightTheme,
  compact,
  showTimestamp,
  softWrap,
  showAllContainers,
  showStd,
  smallerScrollbars,
} from "@/stores/settings";

import mdiThemeLightDark from "~icons/mdi/theme-light-dark";
import mdiWhiteBalanceSunny from "~icons/mdi/white-balance-sunny";
import mdiWeatherNight from "~icons/mdi/weather-night";
import mdiFormatLineSpacing from "~icons/mdi/format-line-spacing";
import mdiClockOutline from "~icons/mdi/clock-outline";
import mdiWrap from "~icons/mdi/wrap";
import mdiEyeOutline from "~icons/mdi/eye-outline";
import mdiFormatListBulleted from "~icons/mdi/format-list-bulleted";
import mdiUnfoldMoreHorizontal from "~icons/mdi/unfold-more-horizontal";
import mdiCogOutline from "~icons/mdi/cog-outline";
import carbonRestart from "~icons/carbon/restart";
import mdiStop from "~icons/mdi/stop";
import mdiPlay from "~icons/mdi/play";
import mdiDownload from "~icons/mdi/download";

export type CommandSection = "container" | "settings" | "navigation";

export type Command = {
  id: string;
  title: string;
  section: CommandSection;
  icon: Component;
  keywords?: string;
  perform: () => unknown;
};

// Central registry for the Cmd+K command palette. Commands are recomputed on
// every access so context-sensitive entries (container actions, current
// toggle labels) stay in sync with the route and settings.
export function useCommands() {
  const { t } = useI18n();
  const router = useRouter();
  const route = useRoute();
  const containerStore = useContainerStore();

  const currentId = computed(() =>
    route?.name === "/container/[id]" && typeof route.params.id === "string" ? route.params.id : "",
  );
  // Null-safe: containerStore.currentContainer is a stubbed action under
  // @pinia/testing, so guard against it being absent.
  const currentContainerRef = containerStore.currentContainer?.(currentId);
  const currentContainer = computed(() => currentContainerRef?.value as Container | undefined);

  // Bound to the current container. The cast is safe because the action
  // handlers only read container.value when invoked, and container commands are
  // only pushed into the list when currentContainer is truthy — so the handlers
  // never run against an undefined container.
  const { start, stop, restart, update } = useContainerActions(currentContainer as Ref<Container>);

  const commands = computed<Command[]>(() => {
    const list: Command[] = [];

    const container = currentContainer.value;
    if (container && config.enableActions) {
      const name = container.name;
      list.push({
        id: "container.restart",
        section: "container",
        icon: carbonRestart,
        title: t("command-palette.restart-container", { name }),
        keywords: "restart reboot",
        perform: restart,
      });
      if (container.state === "running") {
        list.push({
          id: "container.stop",
          section: "container",
          icon: mdiStop,
          title: t("command-palette.stop-container", { name }),
          keywords: "stop kill halt",
          perform: stop,
        });
      } else {
        list.push({
          id: "container.start",
          section: "container",
          icon: mdiPlay,
          title: t("command-palette.start-container", { name }),
          keywords: "start run",
          perform: start,
        });
      }
      list.push({
        id: "container.update",
        section: "container",
        icon: mdiDownload,
        title: t("command-palette.update-container", { name }),
        keywords: "update pull recreate upgrade",
        perform: update,
      });
    }

    list.push(
      // lightTheme is tri-state, so expose each value as its own command rather
      // than a single toggle — that keeps "auto" (follow OS) reachable and makes
      // the target theme explicit instead of depending on the current state.
      {
        id: "settings.theme-auto",
        section: "settings",
        icon: mdiThemeLightDark,
        title: t("command-palette.theme-auto"),
        keywords: "theme auto system color mode appearance",
        perform: () => (lightTheme.value = "auto"),
      },
      {
        id: "settings.theme-light",
        section: "settings",
        icon: mdiWhiteBalanceSunny,
        title: t("command-palette.theme-light"),
        keywords: "theme light color mode appearance",
        perform: () => (lightTheme.value = "light"),
      },
      {
        id: "settings.theme-dark",
        section: "settings",
        icon: mdiWeatherNight,
        title: t("command-palette.theme-dark"),
        keywords: "theme dark color mode appearance",
        perform: () => (lightTheme.value = "dark"),
      },
      {
        id: "settings.toggle-compact",
        section: "settings",
        icon: mdiFormatLineSpacing,
        title: t("command-palette.toggle-compact"),
        keywords: "compact density spacing",
        perform: () => (compact.value = !compact.value),
      },
      {
        id: "settings.toggle-timestamps",
        section: "settings",
        icon: mdiClockOutline,
        title: t("command-palette.toggle-timestamps"),
        keywords: "timestamp time date",
        perform: () => (showTimestamp.value = !showTimestamp.value),
      },
      {
        id: "settings.toggle-soft-wrap",
        section: "settings",
        icon: mdiWrap,
        title: t("command-palette.toggle-soft-wrap"),
        keywords: "wrap soft line",
        perform: () => (softWrap.value = !softWrap.value),
      },
      {
        id: "settings.toggle-stopped",
        section: "settings",
        icon: mdiEyeOutline,
        title: t("command-palette.toggle-stopped"),
        keywords: "stopped hidden all containers exited",
        perform: () => (showAllContainers.value = !showAllContainers.value),
      },
      {
        id: "settings.toggle-std",
        section: "settings",
        icon: mdiFormatListBulleted,
        title: t("command-palette.toggle-std"),
        keywords: "stdout stderr std labels stream",
        perform: () => (showStd.value = !showStd.value),
      },
      {
        id: "settings.toggle-scrollbars",
        section: "settings",
        icon: mdiUnfoldMoreHorizontal,
        title: t("command-palette.toggle-scrollbars"),
        keywords: "scrollbar smaller thin",
        perform: () => (smallerScrollbars.value = !smallerScrollbars.value),
      },
      {
        id: "navigation.settings",
        section: "navigation",
        icon: mdiCogOutline,
        title: t("command-palette.open-settings"),
        keywords: "settings preferences options config",
        perform: () => router.push("/settings"),
      },
    );

    return list;
  });

  // Commands shown before the user types anything: the context-sensitive
  // container actions so e.g. Restart is one keystroke away on a container page.
  const contextCommands = computed(() => commands.value.filter((c) => c.section === "container"));

  return { commands, contextCommands };
}
