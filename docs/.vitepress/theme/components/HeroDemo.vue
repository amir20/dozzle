<script lang="ts" setup>
import { usePreferredReducedMotion } from "@vueuse/core";
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";

// The app's own logo, not a copy, so the hero can never drift from the product.
import logo from "../../../../assets/logo.svg";

// The same artwork the app bundles for container rows, imported from the same
// place, so the sidebar here shows exactly what a real install shows.
import bazarr from "../../../../assets/icons/apps/bazarr.svg";
import jellyfin from "../../../../assets/icons/apps/jellyfin.svg";
import lidarr from "../../../../assets/icons/apps/lidarr.webp";
import overseerr from "../../../../assets/icons/apps/overseerr.svg";
import prowlarr from "../../../../assets/icons/apps/prowlarr.svg";
import qbittorrent from "../../../../assets/icons/apps/qbittorrent.svg";
import radarr from "../../../../assets/icons/apps/radarr.svg";
import sabnzbd from "../../../../assets/icons/apps/sabnzbd.svg";
import sonarr from "../../../../assets/icons/apps/sonarr.svg";
import traefik from "../../../../assets/icons/apps/traefik.svg";

// Entries are taller and vary in height, so the overscan is generous.
const ROWS = 26;
const TICK_MS = 1100;
const CHAT_DELAY_MS = 2000;

type Kind = "raw" | "num" | "link";
type Pair = [key: string, value: string, kind?: Kind];
type Entry = {
  level: "info" | "warn" | "error";
  pairs: Pair[];
  /** Renders the bell in place of the level dot: an alert rule matched this line. */
  alert?: boolean;
};

// Cycled deterministically so the server and the client render the same first
// paint. Nothing here reads the clock or Math.random for the same reason.
const ENTRIES: Entry[] = [
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "RSS sync completed. 12 reports downloaded"],
      ["app", "sonarr"],
      ["indexers", "4", "num"],
      ["duration_ms", "812", "num"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Grabbed: The Expanse - S04E03 [WEBDL-1080p]"],
      ["indexer", "NZBgeek"],
      ["size", "3.4 GB"],
      ["client", "sabnzbd"],
    ],
  },
  {
    level: "warn",
    pairs: [
      ["level", "warn"],
      ["indexer", "Nyaa"],
      ["attempt", "2", "num"],
      ["maxRetries", "3", "num"],
      ["error", "request timed out after 15s"],
      ["message", "indexer query failed, falling back"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "GET /api/v3/queue => HTTP 200 (4.118ms)"],
      ["url.full", "http://sonarr.lan:8989/api/v3/queue", "link"],
      ["client.ip", "10.0.1.42"],
      ["http.response.status_code", "200", "num"],
    ],
  },
  {
    level: "error",
    alert: true,
    pairs: [
      ["level", "error"],
      ["error", "dial tcp 10.0.1.7:8080: connection refused"],
      ["component", "qbittorrent"],
      ["message", "download client unreachable, retrying in 30s"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Imported: Dune Part Two (2024) [Bluray-2160p]"],
      ["path", "/media/movies/Dune Part Two (2024)"],
      ["size", "64.2 GB"],
    ],
  },
  {
    level: "warn",
    pairs: [
      ["level", "warn"],
      ["free", "42.1 GB"],
      ["threshold", "50 GB"],
      ["mount", "/media"],
      ["message", "Disk space below threshold, pausing grabs"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Health check passed"],
      ["checks", "9", "num"],
      ["app", "prowlarr"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "POST /api/v3/command => HTTP 201 (18.302ms)"],
      ["url.full", "http://radarr.lan:7878/api/v3/command", "link"],
      ["command", "RefreshMovie"],
      ["http.request.method", "POST"],
      ["event.duration", "18302", "num"],
    ],
  },
  {
    level: "warn",
    pairs: [
      ["level", "warn"],
      ["release", "Breaking.Bad.S03.COMPLETE.1080p.BluRay"],
      ["reason", "no matching series in library"],
      ["message", "Import failed, moving to unsorted"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Renamed 24 episode files"],
      ["series", "Better Call Saul"],
      ["duration_ms", "412", "num"],
    ],
  },
  {
    level: "error",
    pairs: [
      ["level", "error"],
      ["error", "ffprobe exited with code 1"],
      ["file", "/media/movies/Arrival (2016)/Arrival.2016.mkv"],
      ["message", "failed to probe media file, skipping"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Transcoding started"],
      ["app", "jellyfin"],
      ["session", "8f6faccf-154a-408d"],
      ["profile", "1080p H.264"],
      ["client", "Apple TV"],
    ],
  },
  {
    level: "warn",
    alert: true,
    pairs: [
      ["level", "warn"],
      ["indexer", "1337x"],
      ["failures", "5", "num"],
      ["disabled_until", "2026-08-09T22:14:00Z"],
      ["message", "Indexer temporarily disabled after repeated failures"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Notification sent to Discord"],
      ["event", "Download"],
      ["app", "overseerr"],
    ],
  },
  {
    level: "info",
    pairs: [
      ["level", "INFO", "raw"],
      ["msg", "Refreshed metadata for 312 series"],
      ["duration_ms", "1840", "num"],
      ["app", "sonarr"],
    ],
  },
];

// 12:42:25 PM on 08/09/2026, advancing ~7s per entry. Pure arithmetic keeps it
// stable across SSR and hydration.
const BASE_SECONDS = 12 * 3600 + 42 * 60 + 25;

const pad = (n: number) => String(n).padStart(2, "0");

function entryAt(index: number) {
  const entry = ENTRIES[index % ENTRIES.length];
  const total = BASE_SECONDS + index * 7;
  const hours = Math.floor(total / 3600) % 24;
  const meridiem = hours >= 12 ? "PM" : "AM";
  const time = `${pad(hours % 12 || 12)}:${pad(Math.floor(total / 60) % 60)}:${pad(total % 60)} ${meridiem}`;
  return { ...entry, id: index, date: `08/09/2026 ${time}` };
}

type Status = "running" | "healthy" | "unhealthy";
type Item = { name: string; icon: string; status: Status; active?: boolean };
type Group = { icon: "pin" | "stack" | "box"; label: string; items: Item[] };

const GROUPS: Group[] = [
  {
    icon: "pin",
    label: "Pinned",
    items: [
      { name: "media_sonarr.1", icon: sonarr, status: "healthy", active: true },
      { name: "media_jellyfin.1", icon: jellyfin, status: "running" },
    ],
  },
  {
    icon: "stack",
    label: "media",
    items: [
      { name: "media_bazarr.1", icon: bazarr, status: "running" },
      { name: "media_lidarr.1", icon: lidarr, status: "running" },
      { name: "media_overseerr.1", icon: overseerr, status: "healthy" },
      { name: "media_prowlarr.1", icon: prowlarr, status: "healthy" },
      { name: "media_qbittorrent.1", icon: qbittorrent, status: "unhealthy" },
      { name: "media_radarr.1", icon: radarr, status: "running" },
      { name: "media_sabnzbd.1", icon: sabnzbd, status: "running" },
    ],
  },
  {
    icon: "box",
    label: "Running Containers",
    items: [{ name: "traefik_traefik.1", icon: traefik, status: "healthy" }],
  },
];

const cursor = ref(ROWS);
const rows = computed(() => Array.from({ length: ROWS }, (_, i) => entryAt(cursor.value - ROWS + i)));

// Deterministic pseudo-random series. The sparklines scroll a window across
// these instead of generating values, so the charts look live while staying
// identical between SSR and hydration.
const series = (seed: number, length: number, floor: number) =>
  Array.from({ length }, (_, i) => {
    const n = Math.sin((i + 1) * seed) * 10000;
    return floor + (n - Math.floor(n)) * (100 - floor);
  });

const BARS = 44;
const STAT_TICK_MS = 1000;
// Lengths are coprime-ish so the three readouts don't visibly loop together.
const CPU_SERIES = series(12.9898, 233, 12);
const MEM_SERIES = series(78.233, 197, 62);
const NET_SERIES = series(43.771, 149, 4);

const stat = ref(0);

const at = (src: number[], i: number) => src[i % src.length];
const windowed = (src: number[], offset: number) => Array.from({ length: BARS }, (_, i) => at(src, offset + i));

const cpuBars = computed(() => windowed(CPU_SERIES, stat.value));
const memBars = computed(() => windowed(MEM_SERIES, stat.value));

const rate = (kb: number) => (kb >= 1000 ? `${(kb / 1000).toFixed(1)}M/s` : `${kb.toFixed(1)}K/s`);

const cpuPct = computed(() => (cpuBars.value[BARS - 1] * 0.045).toFixed(1));
const memUsed = computed(() => (88 + memBars.value[BARS - 1] * 0.12).toFixed(1));
const netTx = computed(() => rate(at(NET_SERIES, stat.value) * 3.2));
const netRx = computed(() => rate(at(NET_SERIES, stat.value + 37) * 4.6));
const diskRx = computed(() => rate(at(NET_SERIES, stat.value + 91) * 0.7));

const list = ref<HTMLElement>();
// The panel is the last thing to arrive, so the stream is what the eye lands on
// first and the answer reads as something that happened rather than chrome that
// was always there. Closed on the server too, so hydration matches.
const chatOpen = ref(false);
const reducedMotion = usePreferredReducedMotion();
let timer: ReturnType<typeof setInterval> | undefined;
let statTimer: ReturnType<typeof setInterval> | undefined;
let chatTimer: ReturnType<typeof setTimeout> | undefined;

onMounted(() => {
  if (reducedMotion.value === "reduce") {
    // Nothing animates, so there is nothing to wait for.
    chatOpen.value = true;
    return;
  }

  chatTimer = setTimeout(() => (chatOpen.value = true), CHAT_DELAY_MS);

  timer = setInterval(async () => {
    // The list is bottom-anchored, so appending an entry shifts everything up by
    // the height of the entry dropping off the top. Measure it before the
    // re-render, then play that shift back as motion.
    const shift = list.value?.firstElementChild?.getBoundingClientRect().height ?? 0;
    cursor.value++;
    await nextTick();
    list.value?.animate([{ transform: `translateY(${shift}px)` }, { transform: "none" }], {
      duration: 260,
      easing: "cubic-bezier(0.22, 1, 0.36, 1)",
    });
  }, TICK_MS);

  // Deliberately out of step with the log tick so the panel doesn't pulse.
  statTimer = setInterval(() => stat.value++, STAT_TICK_MS);
});

onBeforeUnmount(() => {
  clearInterval(timer);
  clearInterval(statTimer);
  clearTimeout(chatTimer);
});

const alt = "The Dozzle interface streaming container logs in real time";
</script>

<template>
  <div class="hero-frame">
    <div class="hero-demo" role="img" :aria-label="alt">
      <div class="sidebar" aria-hidden="true">
        <div class="brand">
          <img :src="logo" alt="" class="logo" />
          <span class="wordmark">Dozzle</span>
        </div>

        <div class="rule"></div>

        <!-- A segmented control rather than dots: with a handful of sections,
             naming them is cheaper to read than a caption. -->
        <div class="tabs">
          <span class="tab on">Hosts</span>
          <span class="tab">Services</span>
        </div>

        <div class="nav-head">
          <i class="i i-chevron-left nav-back" />
          <i class="i i-box nav-host-icon" />
          <span class="nav-host">nas</span>
          <span class="head-btn"><i class="i i-merge" /></span>
          <span class="head-btn"><i class="i i-dots" /></span>
        </div>

        <div class="group" v-for="group in GROUPS" :key="group.label">
          <div class="group-head">
            <i class="i i-chevron-right group-caret" />
            <i class="i" :class="`i-${group.icon}`" />
            <span class="group-label">{{ group.label }}</span>
            <span class="group-count">{{ group.items.length }}</span>
          </div>
          <div class="items">
            <div class="item" v-for="item in group.items" :key="item.name" :class="{ active: item.active }">
              <span class="app-icon">
                <img :src="item.icon" alt="" />
                <span v-if="item.status === 'healthy'" class="badge glyph healthy"><i class="i i-check" /></span>
                <span v-else-if="item.status === 'unhealthy'" class="badge glyph unhealthy"
                  ><i class="i i-alert"
                /></span>
                <span v-else class="badge dot running" />
              </span>
              <span class="item-name">{{ item.name }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="stage" aria-hidden="true">
        <div class="topbar">
          <span class="title">media_sonarr.1</span>
          <i class="i i-pin pin" />
          <i class="i i-check health" />
          <span class="image-name">lscr.io/linuxserver/sonarr:4.0.15</span>

          <!-- One surface, hairline separated, the way the toolbar draws it now:
               the sparklines are the only colour so the numbers lead. -->
          <div class="stats">
            <div class="io">
              <span class="io-label">NET</span>
              <i class="i i-arrow-up io-arrow" /><span class="io-num">{{ netTx }}</span>
              <i class="i i-arrow-down io-arrow" /><span class="io-num">{{ netRx }}</span>
              <span class="io-label">DISK</span>
              <i class="i i-arrow-up io-arrow" /><span class="io-num idle">0B/s</span>
              <i class="i i-arrow-down io-arrow" /><span class="io-num">{{ diskRx }}</span>
            </div>

            <div class="stat-card cpu">
              <div class="stat-head">
                <span class="stat-label">CPU</span>
                <b>{{ cpuPct }}%</b><span class="stat-sub">/ 2</span>
              </div>
              <div class="chart">
                <i v-for="(h, i) in cpuBars" :key="i" :style="{ height: `${h}%` }" />
              </div>
            </div>

            <div class="stat-card mem">
              <div class="stat-head">
                <span class="stat-label">MEM</span>
                <b>{{ memUsed }}M</b><span class="stat-sub">/ 1.9G</span>
              </div>
              <div class="chart">
                <i v-for="(h, i) in memBars" :key="i" :style="{ height: `${h}%` }" />
              </div>
            </div>
          </div>

          <span class="std-btn"><i class="std stderr" /><i class="std stdout" /></span>
        </div>

        <div class="logs">
          <div class="log-list" ref="list">
            <div
              v-for="row in rows"
              :key="row.id"
              class="row"
              :class="{ zebra: row.id % 2 === 0 }"
              :data-level="row.level"
            >
              <span class="date">{{ row.date }}</span>
              <!-- A matched rule takes the level marker over rather than adding a
                   second glyph beside it, so one mark says "this fired" and what
                   level it was. -->
              <i v-if="row.alert" class="i i-bell alert-badge" :class="row.level" />
              <span v-else class="level" :class="row.level" />
              <span class="pairs">
                <span class="pair" v-for="[key, value, kind] in row.pairs" :key="key">
                  <span class="key">{{ key }}=</span><span class="value" :class="kind ?? 'string'">{{ value }}</span>
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- The rail's chat panel: what the assistant can answer about the view
           already on screen, sitting beside it rather than over it. -->
      <div class="panel" :class="{ open: chatOpen }" aria-hidden="true">
        <div class="panel-inner">
          <div class="panel-head">
            <i class="i i-message panel-icon" />
            <span class="panel-title">Ask Dozzle</span>
            <span class="panel-by"><i class="i i-cloud" />Dozzle Cloud</span>
            <span class="panel-close">×</span>
          </div>

          <div class="thread">
            <div class="ask">
              <p>why does qbittorrent keep dropping?</p>
              <div class="ctx"><i class="i i-eye" />media_sonarr.1 · last 30 min</div>
            </div>

            <div class="answer">
              <p>
                <b>media_qbittorrent.1</b> has failed its health check 3 times since 12:31 PM, and it is the same
                failure each time.
              </p>
              <ul>
                <li>
                  Memory held at <b>122M</b> of its <b>128M</b> limit for the last hour, then the process was OOM killed
                  and restarted.
                </li>
                <li>sonarr logged <b>connection refused</b> to 10.0.1.7:8080 fourteen times in the same window.</li>
              </ul>
              <p>Raising the memory limit, or capping the disk cache, stops the restarts.</p>
              <span class="answer-action"><i class="i i-arrow-out" />Show me the lines</span>
            </div>
          </div>

          <div class="composer-wrap">
            <div class="ctx"><i class="i i-eye" />media_sonarr.1 · 400 lines</div>
            <div class="composer">
              <span class="composer-hint">Ask about this view…</span>
              <span class="send"><i class="i i-send" /></span>
            </div>
          </div>
        </div>
      </div>

      <!-- The cloud rail: a strip on the right edge with the panels it opens,
           mounted only where Dozzle Cloud is linked. -->
      <div class="rail" aria-hidden="true">
        <span class="cloud-mark"><i class="i i-cloud" /></span>
        <div class="rail-rule"></div>
        <span class="rail-btn" :class="{ on: chatOpen }"><i class="i i-message" /></span>
        <span class="rail-btn"><i class="i i-chart" /></span>
        <span class="rail-btn"><i class="i i-bell" /><span class="rail-dot" /></span>
        <span class="rail-btn rail-hide"><i class="i i-chevron-right" /></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hero-frame {
  /* Inline-size containment means the width can't come from the contents, so
     the frame has to be sized by its parent. */
  width: 100%;
  container-type: inline-size;
  overflow: hidden;
  border: 1px solid oklch(90% 0 0);
  border-radius: 0.5rem;
  filter: drop-shadow(0 4px 3px rgb(0 0 0 / 0.07)) drop-shadow(0 2px 2px rgb(0 0 0 / 0.06));
}

.dark .hero-frame {
  border-color: oklch(22% 0 0);
}

/* The daisyUI palette Dozzle itself ships, so the mock matches the real app. */
.hero-demo {
  --base-100: oklch(100% 0 0);
  --base-200: oklch(97% 0 0);
  --base-300: oklch(90% 0 0);
  --content: oklch(33% 0 0);
  --primary: oklch(79.96% 0.143 176.65);
  --secondary: oklch(82.67% 0.1538 77.82);
  --page: var(--base-100);
  --chip: var(--base-200);
  --green: oklch(0.62 0.119722 158.82);
  --orange: oklch(72% 0.16 60);
  --red: oklch(64% 0.218 28.85);
  --blue: oklch(55% 0.171 249.5);
  --zebra: oklch(70.7% 0.022 261.3 / 0.09);

  /* 1u == 1px at the 1280x760 design size, so the whole mock scales as one unit. */
  --u: calc(100cqw / 1280);

  display: flex;
  aspect-ratio: 1280 / 760;
  /* Decorative: the mock should not behave like selectable page text. */
  user-select: none;
  -webkit-user-select: none;
  pointer-events: none;
  background: var(--page);
  color: var(--content);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: calc(11 * var(--u));
  text-align: left;
}

.dark .hero-demo {
  --base-100: oklch(25% 0 0);
  --base-200: oklch(18% 0 0);
  --base-300: oklch(11% 0 0);
  --content: oklch(89.23% 0 0);
  --primary: oklch(70.96% 0.143 176.65);
  --secondary: oklch(81.38% 0.1448 90.1243);
  --page: var(--base-300);
  --chip: var(--base-100);
  --orange: oklch(85% 0.186 48.13);
  --blue: oklch(65% 0.171 249.5);
  --zebra: oklch(70.7% 0.022 261.3 / 0.07);
}

/* Icons
   --------------------------------------------------------------------------
   Drawn as masks rather than glyphs: at this scale a font character loses its
   shape, and the stroke weight has to match the app's. */

.i {
  display: inline-block;
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
  flex: none;
  background: currentColor;
  mask: no-repeat center / contain var(--glyph);
}

.i-merge {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M12 21V11m0 0L6 5m6 6 6-6'/%3E%3C/svg%3E");
}

.i-pin {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M12 21.5s7-6.2 7-11a7 7 0 1 0-14 0c0 4.8 7 11 7 11Z'/%3E%3Ccircle cx='12' cy='10.5' r='2.6' fill='none' stroke='%23000' stroke-width='2'/%3E%3C/svg%3E");
}

.i-stack {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='m12 3 9 5-9 5-9-5 9-5Zm9 9-9 5-9-5m18 4-9 5-9-5'/%3E%3C/svg%3E");
}

.i-box {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M3 7h18v10H3zM7 7v10m10-10v10'/%3E%3C/svg%3E");
}

.i-chevron-right {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round' d='m9 5 7 7-7 7'/%3E%3C/svg%3E");
}

.i-chevron-left {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round' d='m15 5-7 7 7 7'/%3E%3C/svg%3E");
}

.i-check {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Ccircle cx='12' cy='12' r='9.2' fill='none' stroke='%23000' stroke-width='2.6'/%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.6' stroke-linecap='round' stroke-linejoin='round' d='m7.8 12.3 2.9 2.9 5.5-5.7'/%3E%3C/svg%3E");
}

.i-alert {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Ccircle cx='12' cy='12' r='9.2' fill='none' stroke='%23000' stroke-width='2.6'/%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.6' stroke-linecap='round' d='M12 7.2v5.4m0 3.4v.6'/%3E%3C/svg%3E");
}

.i-arrow-up {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round' d='M12 19V5m0 0-6 6m6-6 6 6'/%3E%3C/svg%3E");
}

.i-arrow-down {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round' d='M12 5v14m0 0-6-6m6 6 6-6'/%3E%3C/svg%3E");
}

.i-bell {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='%23000' d='M12 2.5a5.6 5.6 0 0 0-5.6 5.6c0 5.4-2.2 7-2.2 7h15.6s-2.2-1.6-2.2-7A5.6 5.6 0 0 0 12 2.5Zm2 14.6h-4a2 2 0 0 0 4 0Z'/%3E%3C/svg%3E");
}

.i-cloud {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='%23000' d='M17.6 19H6.8A4.3 4.3 0 0 1 6.2 10.4 6 6 0 0 1 17.9 9.8a4.6 4.6 0 0 1-.3 9.2Z'/%3E%3C/svg%3E");
}

.i-message {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linejoin='round' d='M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2Z'/%3E%3C/svg%3E");
}

.i-dots {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='%23000' d='M12 4.6a1.7 1.7 0 1 0 0 3.4 1.7 1.7 0 0 0 0-3.4Zm0 5.7a1.7 1.7 0 1 0 0 3.4 1.7 1.7 0 0 0 0-3.4Zm0 5.7a1.7 1.7 0 1 0 0 3.4 1.7 1.7 0 0 0 0-3.4Z'/%3E%3C/svg%3E");
}

.i-eye {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M2.5 12S6.5 5.5 12 5.5 21.5 12 21.5 12 17.5 18.5 12 18.5 2.5 12 2.5 12Z'/%3E%3Ccircle cx='12' cy='12' r='3' fill='none' stroke='%23000' stroke-width='2'/%3E%3C/svg%3E");
}

.i-send {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='%23000' d='M2.5 21 22 12 2.5 3l4.4 7.2 9.1 1.8-9.1 1.8Z'/%3E%3C/svg%3E");
}

.i-arrow-out {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M7 17 17 7m0 0H8.5M17 7v8.5'/%3E%3C/svg%3E");
}

.i-chart {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M3 3v18h18M7 15l4-5 3 3 5-7'/%3E%3C/svg%3E");
}

/* Sidebar
   -------------------------------------------------------------------------- */

.sidebar {
  display: flex;
  flex: 0 0 calc(232 * var(--u));
  flex-direction: column;
  gap: calc(10 * var(--u));
  padding: calc(12 * var(--u));
  border-right: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

.brand {
  display: flex;
  align-items: center;
  gap: calc(8 * var(--u));
}

.logo {
  width: calc(30 * var(--u));
  height: calc(30 * var(--u));
}

/* A brand mark, not a heading you read twice: it gives the space back to the
   tree underneath. */
.wordmark {
  font-family:
    ui-sans-serif,
    system-ui,
    -apple-system,
    "Segoe UI",
    Roboto,
    sans-serif;
  font-size: calc(24 * var(--u));
  font-weight: 300;
  letter-spacing: -0.01em;
  line-height: 1;
}

.tabs {
  display: flex;
  gap: calc(2 * var(--u));
  padding: calc(1.5 * var(--u));
  border-radius: calc(5 * var(--u));
  background: color-mix(in oklab, var(--content) 6%, transparent);
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size: calc(10.5 * var(--u));
  font-weight: 500;
  /* VitePress sets a 24px line-height on body, which this inherits: without a
     line-height of its own the strip came out half again as tall as the app's. */
  line-height: 1;
}

.tab {
  flex: 1;
  padding: calc(4 * var(--u)) calc(5 * var(--u));
  border-radius: calc(3.5 * var(--u));
  color: color-mix(in oklab, var(--content) 50%, transparent);
  text-align: center;
}

.tab.on {
  background: var(--base-100);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.06);
  color: var(--content);
}

.rule {
  height: 1px;
  background: color-mix(in oklab, var(--content) 10%, transparent);
}

.nav-head {
  display: flex;
  align-items: center;
  gap: calc(6 * var(--u));
  font-family: ui-sans-serif, system-ui, sans-serif;
}

.nav-back,
.nav-host-icon {
  width: calc(14 * var(--u));
  height: calc(14 * var(--u));
  color: color-mix(in oklab, var(--content) 50%, transparent);
}

.nav-host {
  font-size: calc(13 * var(--u));
  font-weight: 500;
}

.head-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(16 * var(--u));
  height: calc(16 * var(--u));
  border-radius: calc(3 * var(--u));
  color: color-mix(in oklab, var(--content) 45%, transparent);
}

.head-btn:first-of-type {
  margin-left: auto;
}

.head-btn .i {
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
}

/* Deliberately quieter than an item row: the group is scaffolding, the
   containers under it are the content. */
.group-head {
  display: flex;
  align-items: center;
  gap: calc(5 * var(--u));
  height: calc(24 * var(--u));
  padding-left: calc(2 * var(--u));
  color: color-mix(in oklab, var(--content) 55%, transparent);
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size: calc(11.5 * var(--u));
  font-weight: 600;
}

.group-caret {
  width: calc(11 * var(--u));
  height: calc(11 * var(--u));
  transform: rotate(90deg);
}

.group-head .i {
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
}

.group-label {
  white-space: nowrap;
}

.group-count {
  font-weight: 400;
  opacity: 0.6;
}

.items {
  margin-left: calc(10 * var(--u));
  padding-left: calc(4 * var(--u));
  border-left: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

.item {
  display: flex;
  align-items: center;
  gap: calc(8 * var(--u));
  height: calc(28 * var(--u));
  padding: 0 calc(7 * var(--u));
  border-radius: calc(5 * var(--u));
  color: color-mix(in oklab, var(--content) 85%, transparent);
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size: calc(12.5 * var(--u));
}

/* Tinted rather than filled: a solid block on the selected row shouted over the
   app icon and status badge sitting inside it. */
.item.active {
  background: color-mix(in oklab, var(--primary) 15%, transparent);
  color: var(--primary);
  font-weight: 500;
}

.app-icon {
  position: relative;
  display: inline-flex;
  flex: none;
  width: calc(19 * var(--u));
  height: calc(19 * var(--u));
}

.app-icon img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

/* Overhangs the logo's corner, filled and ringed in the page colour so the glyph
   stays legible on busy artwork. */
.badge {
  position: absolute;
  top: calc(-3 * var(--u));
  right: calc(-3 * var(--u));
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(10 * var(--u));
  height: calc(10 * var(--u));
  border-radius: 999px;
  background: var(--page);
  box-shadow: 0 0 0 calc(1.2 * var(--u)) var(--page);
}

.badge .i {
  width: 100%;
  height: 100%;
}

.badge.dot {
  width: calc(7 * var(--u));
  height: calc(7 * var(--u));
}

.badge.healthy {
  color: var(--green);
}

.badge.unhealthy {
  color: var(--red);
}

.badge.running {
  background: var(--green);
}

.item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Main panel
   --------------------------------------------------------------------------
   Named .stage, not .main: VitePress styles a global `.main` and its padding
   left a dead gutter between the stream and the rail. */

.stage {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  padding: 0;
}

.topbar {
  display: flex;
  flex: 0 0 calc(48 * var(--u));
  align-items: center;
  gap: calc(7 * var(--u));
  padding: 0 calc(12 * var(--u));
}

/* The name is the page title, so it carries no button frame of its own. */
.title {
  font-size: calc(13 * var(--u));
  white-space: nowrap;
}

.pin {
  width: calc(13 * var(--u));
  height: calc(13 * var(--u));
  color: color-mix(in oklab, var(--content) 40%, transparent);
}

.health {
  width: calc(13 * var(--u));
  height: calc(13 * var(--u));
  color: var(--green);
}

/* The image is reference material, not a control: dimmed text keeps it out of
   the name's way. */
.image-name {
  overflow: hidden;
  color: color-mix(in oklab, var(--content) 45%, transparent);
  font-size: calc(11 * var(--u));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stats {
  display: flex;
  flex: none;
  align-items: stretch;
  margin-left: auto;
  border-radius: calc(6 * var(--u));
  background: color-mix(in oklab, var(--content) 5.5%, transparent);
}

.stats > * + * {
  border-left: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

/* The app drops this card once the column gets narrow, and an open panel is
   exactly that. */
.io {
  display: none;
  align-items: center;
  gap: calc(5 * var(--u)) calc(5 * var(--u));
  grid-template-columns: auto auto calc(46 * var(--u)) auto calc(46 * var(--u));
  padding: calc(6 * var(--u)) calc(10 * var(--u));
  font-size: calc(11 * var(--u));
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.io-label {
  color: color-mix(in oklab, var(--content) 40%, transparent);
  font-size: calc(9.5 * var(--u));
  font-weight: 500;
  letter-spacing: 0.06em;
}

.io-arrow {
  width: calc(9 * var(--u));
  height: calc(9 * var(--u));
  color: color-mix(in oklab, var(--content) 35%, transparent);
}

.io-num {
  text-align: right;
}

.io-num.idle {
  color: color-mix(in oklab, var(--content) 40%, transparent);
}

.stat-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: calc(4 * var(--u));
  width: calc(168 * var(--u));
  padding: calc(6 * var(--u)) calc(10 * var(--u));
}

.stat-head {
  display: flex;
  align-items: baseline;
  gap: calc(5 * var(--u));
  font-size: calc(13 * var(--u));
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.stat-head b {
  font-weight: 600;
}

.stat-label {
  color: color-mix(in oklab, var(--content) 40%, transparent);
  font-size: calc(9.5 * var(--u));
  font-weight: 500;
  letter-spacing: 0.06em;
}

.stat-sub {
  color: color-mix(in oklab, var(--content) 45%, transparent);
  font-size: calc(11 * var(--u));
}

.chart {
  display: flex;
  align-items: flex-end;
  gap: calc(2 * var(--u));
  height: calc(16 * var(--u));
}

.chart i {
  flex: 1;
  min-height: 1px;
  transition: height 200ms linear;
  border-radius: calc(2 * var(--u)) calc(2 * var(--u)) 0 0;
  opacity: 0.7;
}

.cpu .chart i {
  background: var(--primary);
}

.mem .chart i {
  background: var(--secondary);
}

/* The stdout/stderr toggle, the one control that is always on the row. */
.std-btn {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: calc(3 * var(--u));
  padding: 0 calc(2 * var(--u));
}

.std {
  width: calc(7 * var(--u));
  height: calc(7 * var(--u));
  border-radius: 999px;
}

.std.stderr {
  background: var(--red);
}

.std.stdout {
  background: var(--blue);
}

/* Log stream
   -------------------------------------------------------------------------- */

.logs {
  position: relative;
  overflow: hidden;
  flex: 1;
  /* Fades the partially clipped overscan entry so the top edge reads as a
     scrolled-past tail rather than a cut-off row. */
  mask-image: linear-gradient(to bottom, transparent, black calc(34 * var(--u)));
}

/* Anchored to the bottom so the newest entry sits on the edge and the overscan
   clips off the top. */
.log-list {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
}

.row {
  display: flex;
  align-items: flex-start;
  gap: calc(8 * var(--u));
  padding: calc(4 * var(--u)) calc(14 * var(--u));
  line-height: calc(16 * var(--u));
}

.row.zebra {
  background: var(--zebra);
}

/* A level tint beats the zebra: warn and error rows are findable while
   scrolling past, without a saturated block for every line. */
.row[data-level="warn"] {
  background: color-mix(in oklab, var(--orange) 8%, transparent);
}

.row[data-level="error"] {
  background: color-mix(in oklab, var(--red) 9%, transparent);
}

/* Unboxed: in a wall of monospace the timestamp is a reference, not content, so
   it reads as a quiet left column. */
.date {
  flex: none;
  color: color-mix(in oklab, var(--content) 45%, transparent);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.level {
  flex: none;
  width: calc(5 * var(--u));
  height: calc(5 * var(--u));
  margin-top: calc(6 * var(--u));
  border-radius: 999px;
}

.level.info {
  background: var(--green);
}

.level.warn {
  background: var(--orange);
}

.level.error {
  background: var(--red);
}

.alert-badge {
  width: calc(11 * var(--u));
  height: calc(11 * var(--u));
  margin-top: calc(3 * var(--u));
}

.alert-badge.info {
  color: var(--green);
}

.alert-badge.warn {
  color: var(--orange);
}

.alert-badge.error {
  color: var(--red);
}

.pairs {
  display: flex;
  flex-wrap: wrap;
  column-gap: calc(14 * var(--u));
  min-width: 0;
}

.pair {
  white-space: nowrap;
}

.key {
  color: color-mix(in oklab, var(--content) 70%, transparent);
  font-weight: 300;
}

.value {
  color: var(--content);
  font-weight: 700;
}

.value.string::before,
.value.string::after,
.value.link::before,
.value.link::after {
  content: '"';
}

.value.link {
  color: var(--primary);
  text-decoration: underline;
  text-underline-offset: calc(3 * var(--u));
}

/* Chat panel
   -------------------------------------------------------------------------- */

.panel {
  display: flex;
  /* Slides open from the edge two seconds in. The width animates on the
     wrapper while the column inside holds its own, so the answer arrives
     already laid out instead of reflowing on the way in. */
  flex: 0 0 0;
  overflow: hidden;
  border-left: 1px solid transparent;
  background: var(--base-100);
  transition:
    flex-basis 420ms cubic-bezier(0.22, 1, 0.36, 1),
    border-color 420ms linear;
  font-family:
    ui-sans-serif,
    system-ui,
    -apple-system,
    "Segoe UI",
    Roboto,
    sans-serif;
}

.panel.open {
  flex-basis: calc(372 * var(--u));
  border-left-color: color-mix(in oklab, var(--content) 10%, transparent);
}

.panel-inner {
  display: flex;
  flex-direction: column;
  width: calc(372 * var(--u));
  margin-left: auto;
}

.panel-head {
  display: flex;
  flex: none;
  align-items: center;
  gap: calc(7 * var(--u));
  padding: calc(11 * var(--u)) calc(14 * var(--u));
  border-bottom: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

.panel-icon {
  width: calc(14 * var(--u));
  height: calc(14 * var(--u));
  color: color-mix(in oklab, var(--content) 60%, transparent);
}

.panel-title {
  font-size: calc(13 * var(--u));
  font-weight: 600;
}

/* Who answered. Muted, because the reader needs it once to trust the panel and
   never again while reading it. */
.panel-by {
  display: inline-flex;
  align-items: center;
  gap: calc(4 * var(--u));
  color: color-mix(in oklab, var(--content) 40%, transparent);
  font-size: calc(11 * var(--u));
}

.panel-by .i {
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
  color: color-mix(in oklab, var(--blue) 70%, transparent);
}

.panel-close {
  margin-left: auto;
  color: color-mix(in oklab, var(--content) 45%, transparent);
  font-size: calc(15 * var(--u));
  line-height: 1;
}

.thread {
  display: flex;
  flex: 1;
  flex-direction: column;
  /* Everything hugs the composer: centred, a short thread floats half a panel
     away from the box you type into. */
  justify-content: flex-end;
  gap: calc(14 * var(--u));
  min-height: 0;
  padding: calc(14 * var(--u));
  font-size: calc(12.5 * var(--u));
  line-height: 1.55;
}

/* The question is a neutral panel; the answer is the pane itself. */
.ask {
  padding: calc(10 * var(--u)) calc(11 * var(--u));
  border: 1px solid color-mix(in oklab, var(--content) 15%, transparent);
  border-radius: calc(8 * var(--u));
  background: color-mix(in oklab, var(--base-200) 40%, transparent);
}

.ctx {
  display: flex;
  align-items: center;
  gap: calc(5 * var(--u));
  margin-top: calc(7 * var(--u));
  color: color-mix(in oklab, var(--content) 40%, transparent);
  font-size: calc(11 * var(--u));
}

.ctx .i {
  width: calc(11 * var(--u));
  height: calc(11 * var(--u));
}

.answer p + p,
.answer ul + p,
.answer ul {
  margin-top: calc(9 * var(--u));
}

.answer ul {
  display: flex;
  flex-direction: column;
  gap: calc(5 * var(--u));
  padding-left: calc(14 * var(--u));
  list-style: disc;
}

.answer b {
  font-weight: 600;
}

/* Never a link out: driving the stream on screen to the window an answer
   describes is the whole reason this panel lives in Dozzle. */
.answer-action {
  display: inline-flex;
  align-items: center;
  gap: calc(5 * var(--u));
  margin-top: calc(11 * var(--u));
  padding: calc(4 * var(--u)) calc(8 * var(--u));
  border-radius: calc(5 * var(--u));
  background: color-mix(in oklab, var(--content) 8%, transparent);
  font-size: calc(11.5 * var(--u));
  font-weight: 500;
}

.answer-action .i {
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
  color: color-mix(in oklab, var(--content) 50%, transparent);
}

/* Sticky, opaque and full bleed, like every other footer in the app. */
.composer-wrap {
  flex: none;
  padding: calc(11 * var(--u)) calc(14 * var(--u));
  border-top: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

.composer-wrap .ctx {
  margin-top: 0;
  margin-bottom: calc(8 * var(--u));
}

.composer {
  display: flex;
  align-items: center;
  height: calc(38 * var(--u));
  padding: 0 calc(5 * var(--u)) 0 calc(12 * var(--u));
  border: 1px solid color-mix(in oklab, var(--content) 20%, transparent);
  border-radius: calc(7 * var(--u));
}

.composer-hint {
  color: color-mix(in oklab, var(--content) 40%, transparent);
  font-size: calc(12.5 * var(--u));
}

.send {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(28 * var(--u));
  height: calc(28 * var(--u));
  margin-left: auto;
  border-radius: 999px;
  background: var(--primary);
  color: oklch(20% 0 0);
}

.send .i {
  width: calc(13 * var(--u));
  height: calc(13 * var(--u));
}

/* Cloud rail
   -------------------------------------------------------------------------- */

.rail {
  display: flex;
  flex: 0 0 calc(48 * var(--u));
  flex-direction: column;
  align-items: center;
  gap: calc(4 * var(--u));
  padding: calc(12 * var(--u)) 0;
  border-left: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
  background: color-mix(in oklab, var(--base-200) 40%, transparent);
}

/* Says once where the answers come from, so no panel has to carry a slogan. */
.cloud-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(28 * var(--u));
  height: calc(28 * var(--u));
  border-radius: 999px;
  background: color-mix(in oklab, var(--blue) 12%, transparent);
  color: var(--blue);
}

.cloud-mark .i {
  width: calc(17 * var(--u));
  height: calc(17 * var(--u));
}

.rail-rule {
  width: calc(24 * var(--u));
  height: 1px;
  margin: calc(6 * var(--u)) 0;
  background: color-mix(in oklab, var(--content) 12%, transparent);
}

.rail-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(28 * var(--u));
  height: calc(28 * var(--u));
  border-radius: calc(5 * var(--u));
  color: color-mix(in oklab, var(--content) 60%, transparent);
}

.rail-btn.on {
  background: var(--base-300);
  color: var(--content);
}

.rail-btn .i {
  width: calc(17 * var(--u));
  height: calc(17 * var(--u));
}

.rail-dot {
  position: absolute;
  top: calc(4 * var(--u));
  right: calc(4 * var(--u));
  width: calc(5 * var(--u));
  height: calc(5 * var(--u));
  border-radius: 999px;
  background: var(--orange);
}

.rail-hide {
  margin-top: auto;
  color: color-mix(in oklab, var(--content) 40%, transparent);
}

/* Narrow hero: drop the sidebar, the stat cards and the rail, and scale the
   type against a smaller design width so the entries stay legible instead of
   collapsing into texture. */
@container (max-width: 620px) {
  .hero-demo {
    --u: calc(100cqw / 520);
  }

  .sidebar,
  .stats,
  .panel,
  .rail {
    display: none;
  }

  /* No room to keep a pair intact at this width; let long values break rather
     than run off the edge. */
  .pair {
    white-space: normal;
    overflow-wrap: anywhere;
  }
}

@media (prefers-reduced-motion: reduce) {
  .chart i,
  .panel {
    transition: none;
  }
}
</style>
