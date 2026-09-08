<template>
  <aside ref="root" class="flex h-[calc(100svh-50px)] flex-col gap-4">
    <header class="flex items-center gap-3 pe-20">
      <ri:terminal-window-fill v-if="action === 'attach'" class="text-primary size-7 shrink-0" />
      <material-symbols:terminal v-else class="text-primary size-7 shrink-0" />
      <div class="flex min-w-0 flex-col">
        <div class="flex items-center gap-2">
          <h1 class="text-xl leading-tight font-semibold">
            {{ action === "attach" ? $t("toolbar.attach") : $t("toolbar.shell") }}
          </h1>
          <span class="badge badge-sm bg-base-300 gap-1.5 border-none">
            <span class="status size-1.5" :class="statusDotClass"></span>
            {{ statusLabel }}
          </span>
        </div>
        <p class="text-base-content/60 flex items-center gap-1.5 text-sm">
          <span class="truncate font-mono">{{ container.name }}</span>
          <span class="opacity-40">·</span>
          <RelativeTime :date="container.created" />
        </p>
      </div>

      <div class="ms-auto flex shrink-0 items-center gap-1">
        <button
          class="btn btn-ghost btn-xs btn-square"
          :class="{ 'btn-active': searchOpen }"
          :title="$t('terminal.search')"
          :aria-label="$t('terminal.search')"
          @click="searchOpen ? closeSearch() : openSearch()"
        >
          <mdi:magnify />
        </button>
        <button
          class="btn btn-ghost btn-xs btn-square"
          :title="$t('terminal.decrease-font')"
          :aria-label="$t('terminal.decrease-font')"
          :disabled="fontSize <= MIN_FONT_SIZE"
          @click="fontSize--"
        >
          <mdi:format-font-size-decrease />
        </button>
        <button
          class="btn btn-ghost btn-xs btn-square"
          :title="$t('terminal.increase-font')"
          :aria-label="$t('terminal.increase-font')"
          :disabled="fontSize >= MAX_FONT_SIZE"
          @click="fontSize++"
        >
          <mdi:format-font-size-increase />
        </button>
        <button
          class="btn btn-ghost btn-xs btn-square"
          :title="$t('toolbar.clear')"
          :aria-label="$t('toolbar.clear')"
          @click="clear()"
        >
          <octicon:trash-24 />
        </button>
      </div>
    </header>

    <div class="shell relative min-h-0 flex-1">
      <div ref="host" class="size-full"></div>

      <Transition
        enter-active-class="transition duration-150"
        enter-from-class="-translate-y-2 opacity-0"
        leave-active-class="transition duration-100"
        leave-to-class="-translate-y-2 opacity-0"
      >
        <div
          v-if="searchOpen"
          class="bg-base-300 border-base-content/20 absolute end-3 top-3 z-10 flex items-center gap-1 rounded border p-1 shadow-md"
        >
          <input
            ref="searchInput"
            v-model="searchTerm"
            type="text"
            class="input input-xs w-44 focus:outline-none"
            :class="{ 'input-error': searchTerm && results.resultCount === 0 }"
            :placeholder="$t('terminal.search')"
            :aria-label="$t('terminal.search')"
            spellcheck="false"
            autocapitalize="off"
            autocomplete="off"
            @keydown.esc.prevent.stop="closeSearch()"
            @keydown.enter.prevent.stop="$event.shiftKey ? findPrevious() : findNext()"
            @blur="searchAddon.clearActiveDecoration()"
          />
          <span class="text-base-content/50 w-16 shrink-0 text-center text-xs tabular-nums">{{ resultLabel }}</span>
          <button
            class="btn btn-ghost btn-xs btn-square"
            :title="$t('terminal.previous-match')"
            :aria-label="$t('terminal.previous-match')"
            :disabled="!results.resultCount"
            @click="findPrevious()"
          >
            <mdi:chevron-up />
          </button>
          <button
            class="btn btn-ghost btn-xs btn-square"
            :title="$t('terminal.next-match')"
            :aria-label="$t('terminal.next-match')"
            :disabled="!results.resultCount"
            @click="findNext()"
          >
            <mdi:chevron-down />
          </button>
          <button
            class="btn btn-ghost btn-xs btn-square"
            :title="$t('terminal.close-search')"
            :aria-label="$t('terminal.close-search')"
            @click="closeSearch()"
          >
            <mdi:close />
          </button>
        </div>
      </Transition>

      <Transition
        enter-active-class="transition duration-200"
        enter-from-class="translate-y-full"
        leave-active-class="transition duration-150"
        leave-to-class="translate-y-full"
      >
        <div
          v-if="notice"
          class="bg-base-300/95 border-base-content/20 absolute inset-x-0 bottom-0 flex items-center justify-center gap-3 border-t px-3 py-2 text-sm"
        >
          <span class="text-base-content/70">{{ notice }}</span>
          <button class="btn btn-primary btn-xs" v-if="isRunning" @click="connect()">
            <carbon:restart /> {{ $t("terminal.reconnect") }}
          </button>
        </div>
      </Transition>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import { terminalFontSize } from "@/stores/settings";
import { useResolvedTheme } from "@/composable/theme";
import type { ISearchOptions } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";

const { container, action } = defineProps<{ container: Container; action: "attach" | "exec" }>();

const [{ Terminal }, { WebLinksAddon }, { FitAddon }, { SearchAddon }] = await Promise.all([
  import("@xterm/xterm"),
  import("@xterm/addon-web-links"),
  import("@xterm/addon-fit"),
  import("@xterm/addon-search"),
]);

const MIN_FONT_SIZE = 10;
const MAX_FONT_SIZE = 22;

const { t } = useI18n();
const { theme } = useResolvedTheme();
const root = useTemplateRef<HTMLElement>("root");
const host = useTemplateRef<HTMLDivElement>("host");
const searchInput = useTemplateRef<HTMLInputElement>("searchInput");

const clamp = (value: number) => Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, value));
const fontSize = computed({
  get: () => clamp(terminalFontSize.value),
  set: (value) => (terminalFontSize.value = clamp(value)),
});

const status = ref<"connecting" | "connected" | "closed">("connecting");
const isRunning = computed(() => container.state === "running");

const statusLabel = computed(() => (isRunning.value ? t(`terminal.${status.value}`) : container.state));

const statusDotClass = computed(() => {
  if (!isRunning.value) return "status-error";
  return { connecting: "status-warning animate-pulse", connected: "status-success", closed: "status-error" }[
    status.value
  ];
});

// The bar is the only place a dead session can be revived from, so it stays until the
// socket is back up. A stopped container has nothing to connect to, so it says so instead.
const notice = computed(() => {
  if (!isRunning.value) return t("terminal.not-running");
  if (status.value === "closed") return t("terminal.connection-closed");
  return null;
});

const terminal = new Terminal({
  cursorBlink: true,
  cursorStyle: "block",
  fontSize: fontSize.value,
  scrollback: 5000,
});
terminal.loadAddon(new WebLinksAddon());
const fitAddon = new FitAddon();
terminal.loadAddon(fitAddon);
const searchAddon = new SearchAddon();
terminal.loadAddon(searchAddon);

const searchOpen = ref(false);
const searchTerm = ref("");
const results = ref({ resultIndex: -1, resultCount: 0 });
// Filled by applyTheme, so the highlights follow the palette rather than xterm's defaults.
const decorations = shallowRef<ISearchOptions["decorations"]>();
const searchOptions = computed(() => ({ decorations: decorations.value }));

const resultLabel = computed(() => {
  if (!searchTerm.value) return "";
  if (results.value.resultCount === 0) return t("terminal.no-results");
  return `${results.value.resultIndex + 1}/${results.value.resultCount}`;
});

searchAddon.onDidChangeResults((event) => (results.value = event));

const findNext = () => searchAddon.findNext(searchTerm.value, searchOptions.value);
const findPrevious = () => searchAddon.findPrevious(searchTerm.value, searchOptions.value);

function openSearch() {
  searchOpen.value = true;
  // Reopening on an existing term selects it, so typing replaces rather than appends.
  nextTick(() => searchInput.value?.select());
}

function closeSearch() {
  searchOpen.value = false;
  searchAddon.clearDecorations();
  results.value = { resultIndex: -1, resultCount: 0 };
  terminal.focus();
}

watch(searchTerm, (term) => {
  if (term) searchAddon.findNext(term, { ...searchOptions.value, incremental: true });
  else {
    searchAddon.clearDecorations();
    results.value = { resultIndex: -1, resultCount: 0 };
  }
});

let ws: WebSocket | null = null;

function sendEvent(type: "userinput" | "resize", data?: string, width?: number, height?: number) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;

  const event: { type: string; data?: string; width?: number; height?: number } = { type };
  if (data !== undefined) event.data = data;
  if (width !== undefined) event.width = width;
  if (height !== undefined) event.height = height;

  ws.send(JSON.stringify(event));
}

function connect() {
  ws?.close();
  status.value = "connecting";

  const socket = new WebSocket(withBase(`/api/hosts/${container.host}/containers/${container.id}/${action}`));
  ws = socket;

  socket.onopen = () => {
    if (ws !== socket) return;
    status.value = "connected";
    terminal.writeln(t("terminal.attaching", { name: container.name }) + " 🚀");
    sendEvent("resize", undefined, terminal.cols, terminal.rows);
    if (action === "attach") sendEvent("userinput", "\r");
    terminal.focus();
  };

  socket.onmessage = (event) => terminal.write(event.data);

  // A socket already replaced by a reconnect must not report the new session as closed.
  socket.addEventListener("close", () => {
    if (ws === socket) status.value = "closed";
  });
}

function clear() {
  terminal.clear();
  terminal.focus();
}

/** Resolves theme custom properties to sRGB triples. xterm needs literal colors, and the
 * app's palette is authored in oklch, which xterm's color parser cannot read. */
function resolveColors(element: HTMLElement, names: string[]) {
  const context = document.createElement("canvas").getContext("2d", { willReadFrequently: true });
  const probe = document.createElement("span");
  probe.style.cssText = "position:absolute;visibility:hidden;pointer-events:none";
  element.appendChild(probe);

  const colors: Record<string, [number, number, number]> = {};
  for (const name of names) {
    probe.style.color = `var(${name})`;
    const resolvedColor = getComputedStyle(probe).color;
    if (context) {
      context.fillStyle = "#000000";
      context.fillStyle = resolvedColor;
      context.fillRect(0, 0, 1, 1);
      const [r, g, b] = context.getImageData(0, 0, 1, 1).data;
      colors[name] = [r, g, b];
    } else {
      colors[name] = [0, 0, 0];
    }
  }

  probe.remove();
  return colors;
}

const ANSI = [
  "black",
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "white",
  "brightBlack",
  "brightRed",
  "brightGreen",
  "brightYellow",
  "brightBlue",
  "brightMagenta",
  "brightCyan",
  "brightWhite",
] as const;

const ansiVar = (name: string) => `--ansi-${name.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase())}`;

function applyTheme() {
  if (!host.value) return;

  const vars = [
    "--color-base-200",
    "--color-base-content",
    "--color-primary",
    "--color-secondary",
    ...ANSI.map(ansiVar),
  ];
  const resolved = resolveColors(host.value, vars);
  const toHex = (rgb: [number, number, number]) => "#" + rgb.map((v) => v.toString(16).padStart(2, "0")).join("");
  const hex = (name: string) => toHex(resolved[name]);

  terminal.options.theme = {
    background: hex("--color-base-200"),
    foreground: hex("--color-base-content"),
    cursor: hex("--color-primary"),
    cursorAccent: hex("--color-base-200"),
    selectionBackground: `rgba(${resolved["--color-primary"].join(",")},0.35)`,
    ...Object.fromEntries(ANSI.map((name) => [name, hex(ansiVar(name))])),
  };

  // Decorations only take #RRGGBB, so the highlight is blended against the background here
  // rather than left translucent — a solid secondary would bury the text in either theme.
  const blend = (ratio: number) =>
    toHex(
      resolved["--color-secondary"].map((v, i) =>
        Math.round(v * ratio + resolved["--color-base-200"][i] * (1 - ratio)),
      ) as [number, number, number],
    );
  const match = blend(0.35);
  const activeMatch = blend(0.75);
  decorations.value = {
    matchBackground: match,
    matchOverviewRuler: match,
    activeMatchBackground: activeMatch,
    activeMatchColorOverviewRuler: activeMatch,
  };
}

function fit() {
  if (host.value?.clientHeight) fitAddon.fit();
}

onMounted(() => {
  terminal.open(host.value!);
  applyTheme();
  fit();

  // Fires for the window, the drawer's maximize toggle and the font-size buttons alike,
  // none of which the old window-size watch saw.
  useResizeObserver(host, () => requestAnimationFrame(fit));

  // Capture phase on the drawer root so the shortcut beats both xterm's own key handling
  // and the log viewer's window-level Cmd+F, which would otherwise open behind the drawer.
  useEventListener(
    root,
    "keydown",
    (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey) || event.shiftKey || event.key.toLowerCase() !== "f") return;
      event.preventDefault();
      event.stopPropagation();
      openSearch();
    },
    { capture: true },
  );

  const disposables = [
    terminal.onData((data) => sendEvent("userinput", data)),
    terminal.onResize(({ cols, rows }) => sendEvent("resize", undefined, cols, rows)),
  ];
  onUnmounted(() => disposables.forEach((d) => d.dispose()));

  if (isRunning.value) connect();
  else status.value = "closed";
});

watch(theme, () => {
  applyTheme();
  // Existing decorations keep the old palette's colors until they are redrawn.
  if (searchTerm.value) searchAddon.findNext(searchTerm.value, { ...searchOptions.value, incremental: true });
});
watch(fontSize, (value) => {
  terminal.options.fontSize = value;
  requestAnimationFrame(fit);
});

// A container that comes back up should be one click away rather than a reopened drawer.
watch(isRunning, (running) => {
  if (running && status.value === "closed") connect();
});

onUnmounted(() => {
  terminal.dispose();
  ws?.close();
  ws = null;
});
</script>
<style scoped>
@reference "@/main.css";

.shell {
  @apply border-base-content/20 bg-base-200 overflow-hidden rounded border p-2 transition-colors;

  &:has(.terminal.focus) {
    @apply border-primary;
  }

  & :deep(.terminal) {
    @apply size-full;
  }

  & :deep(.xterm-cursor-block.xterm-cursor-blink) {
    animation-name: blink !important;
  }
}

@keyframes blink {
  0% {
    background-color: var(--color-base-content);
    color: var(--color-base-200);
  }

  50% {
    background-color: inherit;
    color: var(--color-base-content);
  }
}
</style>
