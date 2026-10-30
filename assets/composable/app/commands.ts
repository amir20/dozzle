import { defineAsyncComponent, shallowReactive, type Component } from "vue";
import { Container, powerAction } from "@/models/Container";
import { canUpdate, isSelf, useImageUpdates } from "@/composable/containers/imageUpdate";
import { useDrawer } from "@/composable/app/drawer";
import { useContainerActions } from "@/composable/containers/containerActions";
import { settingsPages, SETTINGS_PAGE_ICONS, SETTINGS_PAGE_LABELS } from "@/composable/app/settingsPages";
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
import mdiBellOutline from "~icons/mdi/bell-outline";
import mdiBellOffOutline from "~icons/mdi/bell-off-outline";
import mdiMessageOutline from "~icons/mdi/message-outline";
import mdiConsole from "~icons/mdi/console";
import mdiLinkVariant from "~icons/mdi/link-variant";
import carbonRestart from "~icons/carbon/restart";
import mdiStop from "~icons/mdi/stop";
import mdiPlay from "~icons/mdi/play";
import mdiDownload from "~icons/mdi/download";
import mdiUpdate from "~icons/mdi/update";
import mdiPackageDown from "~icons/mdi/package-down";

const ContainerUpdatesDrawer = defineAsyncComponent(() => import("@/components/containers/ContainerUpdatesDrawer.vue"));
const Terminal = defineAsyncComponent(() => import("@/components/containers/Terminal.vue"));

export type CommandSection = "container" | "updates" | "view" | "settings" | "navigation";

export type Command = {
  id: string;
  title: string;
  section: CommandSection;
  icon: Component;
  keywords?: string;
  /** Keys that run this outside the palette, shown on the row so people learn them. */
  shortcut?: string[];
  /** The current value of an on/off setting, shown as a pill. */
  state?: boolean;
  /** Marks the option already in effect, for a choice between several commands. */
  active?: boolean;
  /** Muted trailing text, such as the versions an update moves between. */
  detail?: string;
  /** Asks for Enter a second time, showing this text, before running. */
  confirm?: string;
  perform: () => unknown;
};

/** The platform's command key, as the palette prints it. */
export const modKey = () => (isMac ? "⌘" : "Ctrl");

// Commands only a mounted view can offer, such as analytics over the logs on
// screen. The view registers a getter for as long as it is mounted, so what the
// palette lists always matches what the view's own shortcuts would do.
const viewSources = shallowReactive(new Set<() => Command[]>());

export function provideViewCommands(source: () => Command[]) {
  viewSources.add(source);
  onScopeDispose(() => viewSources.delete(source));
}

/**
 * Everything the palette can do to one container. Used for the container on
 * screen and for any container picked in the palette. `onPage` adds the
 * shortcut hints, which only work on that container's own page.
 */
export function useContainerCommands(containerRef: Ref<Container | undefined>, onPage = false) {
  const { t } = useI18n();
  // The cast is safe because the handlers only read container.value when
  // invoked, and no command is listed while the container is undefined.
  const { start, stop, restart, update } = useContainerActions(containerRef as Ref<Container>);
  const { resultFor, hasUpdate, isDismissed, dismiss, check } = useImageUpdates();
  const { showToast } = useToast();
  const showDrawer = useDrawer();

  async function checkNow(container: Container) {
    await check(container, true);
    const status = resultFor(container)?.status;
    if (status === "update-available") {
      showToast({ message: t("toolbar.update-available"), type: "info" }, { expire: 5000 });
    } else if (status === "up-to-date") {
      showToast({ message: t("toolbar.update-up-to-date"), type: "info" }, { expire: 5000 });
    }
  }

  return computed<Command[]>(() => {
    const container = containerRef.value;
    // A deleted container has nothing left to act on.
    if (!container || container.state === "deleted") return [];
    const list: Command[] = [];
    const name = container.name;

    if (config.enableActions && powerAction(container)) {
      list.push({
        id: "container.restart",
        section: "container",
        icon: carbonRestart,
        title: t("command-palette.restart-container", { name }),
        keywords: "restart reboot",
        perform: restart,
      });
      // Kubernetes has no stop, start or image update for one container, only restart.
      if (config.mode !== "k8s") {
        const power = powerAction(container);
        if (power === "stop") {
          list.push({
            id: "container.stop",
            section: "container",
            icon: mdiStop,
            title: t("command-palette.stop-container", { name }),
            keywords: "stop kill halt",
            confirm: t("command-palette.confirm-stop", { name }),
            perform: stop,
          });
        } else if (power === "start") {
          list.push({
            id: "container.start",
            section: "container",
            icon: mdiPlay,
            title: t("command-palette.start-container", { name }),
            keywords: "start run",
            perform: start,
          });
        }
        // Same gate and self-update handling as the toolbar, so the palette cannot
        // update Dozzle out from under the page or offer what the backend refuses.
        if (canUpdate(container)) {
          const result = resultFor(container);
          list.push({
            id: "container.update",
            section: "container",
            icon: mdiDownload,
            title: t("command-palette.update-container", { name }),
            keywords: "update pull recreate upgrade",
            detail:
              result?.status === "update-available" && result.latestVersion
                ? [result.currentVersion, result.latestVersion].filter(Boolean).join(" → ")
                : undefined,
            confirm: t("command-palette.confirm-update", { name }),
            perform: () => update({ self: isSelf(container) }),
          });
        }
      }
    }

    if (config.imageCheckMode !== "off") {
      list.push({
        id: "container.check-update",
        section: "container",
        icon: mdiUpdate,
        title: t("command-palette.check-container-updates", { name }),
        keywords: "check update image newer outdated",
        perform: () => checkNow(container),
      });
      if (hasUpdate(container) && !isDismissed(container)) {
        list.push({
          id: "container.skip-update",
          section: "container",
          icon: mdiBellOffOutline,
          title: t("command-palette.skip-update", { name }),
          keywords: "skip dismiss ignore update",
          perform: () => dismiss(container),
        });
      }
    }

    if (config.enableShell && container.state === "running") {
      list.push(
        {
          id: "container.attach",
          section: "container",
          icon: mdiLinkVariant,
          title: t("command-palette.attach-shell", { name }),
          keywords: "attach terminal tty console",
          shortcut: onPage ? ["A"] : undefined,
          perform: () => showDrawer(Terminal, { container, action: "attach" }, "lg"),
        },
        {
          id: "container.exec",
          section: "container",
          icon: mdiConsole,
          title: t("command-palette.exec-shell", { name }),
          keywords: "exec shell terminal bash sh console",
          shortcut: onPage ? ["E"] : undefined,
          perform: () => showDrawer(Terminal, { container, action: "exec" }, "lg"),
        },
      );
    }

    return list;
  });
}

// Central registry for the Cmd+K command palette. Commands are recomputed on
// every access so context-sensitive entries (container actions, current
// toggle values) stay in sync with the route and settings.
export function useCommands() {
  const { t } = useI18n();
  const router = useRouter();
  const route = useRoute();
  const containerStore = useContainerStore();
  const { visibleContainers, allContainersById } = storeToRefs(containerStore);

  const currentId = computed(() =>
    route?.name === "/container/[id]" && typeof route.params.id === "string" ? route.params.id : "",
  );
  // From every container, not only the visible ones, so a stopped container's
  // page still offers Start while stopped containers are hidden.
  const currentContainer = computed<Container | undefined>(() => allContainersById.value?.[currentId.value]);
  const containerCommands = useContainerCommands(currentContainer, true);

  const { checkAll, hasUpdate, isDismissed } = useImageUpdates();
  const showDrawer = useDrawer();
  const { available: railAvailable, openRail } = useCloudRail();

  // What the updates drawer would list, minus what the user already skipped.
  const updatable = computed(() =>
    config.enableUpdates && config.imageCheckMode !== "off" && config.mode !== "k8s"
      ? visibleContainers.value.filter((c) => hasUpdate(c) && !isDismissed(c))
      : [],
  );

  const commands = computed<Command[]>(() => {
    const list: Command[] = [...containerCommands.value];

    for (const source of viewSources) list.push(...source());

    // The `.` key and Cmd+Shift+K open the same pane, wherever it can show.
    if (railAvailable.value) {
      list.push({
        id: "view.assistant",
        section: "view",
        icon: mdiMessageOutline,
        title: t("command-palette.open-assistant"),
        keywords: "ask assistant chat ai question",
        shortcut: ["."],
        perform: () => openRail("chat"),
      });
    }

    // Same gate as the dashboard's updates button. Opens the drawer so the
    // check has somewhere to report, even when it finds nothing.
    if (config.enableUpdates && config.imageCheckMode !== "off" && config.mode !== "k8s") {
      if (updatable.value.length) {
        list.push({
          id: "updates.review",
          section: "updates",
          icon: mdiPackageDown,
          title: t("command-palette.review-updates", updatable.value.length),
          keywords: "review updates images upgrade pull outdated all",
          perform: () => showDrawer(ContainerUpdatesDrawer, {}, "lg"),
        });
      }
      list.push({
        id: "updates.check",
        section: "updates",
        icon: mdiUpdate,
        title: t("toolbar.check-for-updates"),
        keywords: "check updates images upgrade pull outdated",
        perform: () => {
          showDrawer(ContainerUpdatesDrawer, {}, "lg");
          return checkAll(true);
        },
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
        active: lightTheme.value === "auto",
        perform: () => (lightTheme.value = "auto"),
      },
      {
        id: "settings.theme-light",
        section: "settings",
        icon: mdiWhiteBalanceSunny,
        title: t("command-palette.theme-light"),
        keywords: "theme light color mode appearance",
        active: lightTheme.value === "light",
        perform: () => (lightTheme.value = "light"),
      },
      {
        id: "settings.theme-dark",
        section: "settings",
        icon: mdiWeatherNight,
        title: t("command-palette.theme-dark"),
        keywords: "theme dark color mode appearance",
        active: lightTheme.value === "dark",
        perform: () => (lightTheme.value = "dark"),
      },
      {
        id: "settings.toggle-compact",
        section: "settings",
        icon: mdiFormatLineSpacing,
        title: t("command-palette.toggle-compact"),
        keywords: "compact density spacing",
        state: compact.value,
        perform: () => (compact.value = !compact.value),
      },
      {
        id: "settings.toggle-timestamps",
        section: "settings",
        icon: mdiClockOutline,
        title: t("command-palette.toggle-timestamps"),
        keywords: "timestamp time date",
        state: showTimestamp.value,
        perform: () => (showTimestamp.value = !showTimestamp.value),
      },
      {
        id: "settings.toggle-soft-wrap",
        section: "settings",
        icon: mdiWrap,
        title: t("command-palette.toggle-soft-wrap"),
        keywords: "wrap soft line",
        state: softWrap.value,
        perform: () => (softWrap.value = !softWrap.value),
      },
      {
        id: "settings.toggle-stopped",
        section: "settings",
        icon: mdiEyeOutline,
        title: t("command-palette.toggle-stopped"),
        keywords: "stopped hidden all containers exited",
        state: showAllContainers.value,
        perform: () => (showAllContainers.value = !showAllContainers.value),
      },
      {
        id: "settings.toggle-std",
        section: "settings",
        icon: mdiFormatListBulleted,
        title: t("command-palette.toggle-std"),
        keywords: "stdout stderr std labels stream",
        state: showStd.value,
        perform: () => (showStd.value = !showStd.value),
      },
      {
        id: "settings.toggle-scrollbars",
        section: "settings",
        icon: mdiUnfoldMoreHorizontal,
        title: t("command-palette.toggle-scrollbars"),
        keywords: "scrollbar smaller thin",
        state: smallerScrollbars.value,
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

    // The same pages the settings nav shows on this install.
    for (const page of settingsPages(config)) {
      list.push({
        id: `navigation.settings.${page}`,
        section: "navigation",
        icon: SETTINGS_PAGE_ICONS[page],
        title: t("command-palette.open-settings-page", { page: t(SETTINGS_PAGE_LABELS[page]) }),
        keywords: `settings ${page}`,
        perform: () => router.push(`/settings/${page}`),
      });
    }

    list.push({
      id: "navigation.notifications",
      section: "navigation",
      icon: mdiBellOutline,
      title: t("command-palette.open-notifications"),
      keywords: "notifications alerts rules destinations",
      perform: () => router.push({ name: "/notifications" }),
    });

    return list;
  });

  // Commands shown before the user types anything: the context-sensitive
  // container actions so e.g. Restart is one keystroke away on a container page.
  const contextCommands = computed(() => commands.value.filter((c) => c.section === "container"));

  return { commands, contextCommands, updatable };
}
