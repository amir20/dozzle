<script lang="ts" setup>
import { usePreferredReducedMotion } from "@vueuse/core";
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";

// The app's own logo, not a copy, so the hero can never drift from the product.
import logo from "../../../../assets/logo.svg";

// Entries are taller and vary in height, so the overscan is generous.
const ROWS = 28;
const TICK_MS = 1100;

type Kind = "raw" | "num" | "link";
type Pair = [key: string, value: string, kind?: Kind];
type Entry = { level: "info" | "warn" | "error"; pairs: Pair[] };

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

const cursor = ref(ROWS);
const rows = computed(() => Array.from({ length: ROWS }, (_, i) => entryAt(cursor.value - ROWS + i)));

// Deterministic pseudo-random bars for the CPU and memory sparklines.
const bars = (seed: number, count: number, floor: number) =>
  Array.from({ length: count }, (_, i) => {
    const n = Math.sin((i + 1) * seed) * 10000;
    return floor + (n - Math.floor(n)) * (100 - floor);
  });

const cpuBars = bars(12.9898, 44, 18);
const memBars = bars(78.233, 44, 55);

const list = ref<HTMLElement>();
const reducedMotion = usePreferredReducedMotion();
let timer: ReturnType<typeof setInterval> | undefined;

onMounted(() => {
  if (reducedMotion.value === "reduce") return;
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
});

onBeforeUnmount(() => clearInterval(timer));

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

        <div class="crumbs">
          <span class="crumb-link">Hosts</span>
          <span class="crumb-sep">›</span>
          <span class="host-pill"><i class="i-merge" />nas</span>
          <span class="crumb-more">⋮</span>
        </div>

        <div
          class="group"
          v-for="group in [
            {
              icon: 'pin',
              label: 'Pinned',
              items: [{ name: 'media_sonarr.1', active: true }, { name: 'media_jellyfin.1' }],
            },
            {
              icon: 'stack',
              label: 'media',
              items: [
                { name: 'media_bazarr.1' },
                { name: 'media_lidarr.1' },
                { name: 'media_overseerr.1', healthy: true },
                { name: 'media_prowlarr.1', healthy: true },
                { name: 'media_qbittorrent.1' },
                { name: 'media_radarr.1' },
                { name: 'media_sabnzbd.1' },
              ],
            },
            { icon: 'box', label: 'Running Containers', items: [{ name: 'traefik_traefik.1' }] },
          ]"
          :key="group.label"
        >
          <div class="group-head">
            <i class="i-group" :class="`i-${group.icon}`" />
            <span class="group-label">{{ group.label }} ({{ group.items.length }})</span>
            <span class="merge-btn"><i class="i-merge" /></span>
            <span class="chevron">⌃</span>
          </div>
          <div class="item" v-for="item in group.items" :key="item.name" :class="{ active: item.active }">
            <span class="dot" />
            <span class="item-name">{{ item.name }}</span>
            <span class="health" v-if="item.healthy">✓</span>
          </div>
        </div>
      </div>

      <div class="main" aria-hidden="true">
        <div class="topbar">
          <span class="star">★</span>
          <span class="name-btn">media_sonarr.1 <span class="caret">▾</span></span>
          <span class="health topbar-health">✓</span>
          <span class="image-tag">
            <span class="image-name">lscr.io/linuxserver/sonarr:4.0.15</span>
            <span class="copy">⧉</span>
          </span>

          <div class="cards">
            <div class="io-card">
              <span class="io-icon">◇</span>
              <span class="up">↑</span><span class="io-num">235.5K/s</span> <span class="down">↓</span
              ><span class="io-num">355.1K/s</span>
              <span class="io-icon">▤</span>
              <span class="up">↑</span><span class="io-num">0B/s</span> <span class="down">↓</span
              ><span class="io-num">56K/s</span>
            </div>

            <div class="stat-card cpu">
              <div class="stat-head">
                <span class="stat-icon">▢</span><b>1.7%</b><span class="stat-sub">/ 2 CPU</span>
              </div>
              <div class="chart">
                <i v-for="(h, i) in cpuBars" :key="i" :style="{ height: `${h}%` }" />
              </div>
            </div>

            <div class="stat-card mem">
              <div class="stat-head">
                <span class="stat-icon">▤</span><b>95.6M</b><span class="stat-sub">/ 1.9G</span>
              </div>
              <div class="chart">
                <i v-for="(h, i) in memBars" :key="i" :style="{ height: `${h}%` }" />
              </div>
            </div>
          </div>
        </div>

        <div class="logs">
          <div class="log-list" ref="list">
            <div v-for="row in rows" :key="row.id" class="row" :class="{ zebra: row.id % 2 === 0 }">
              <span class="date">{{ row.date }}</span>
              <span class="level" :class="row.level" />
              <span class="pairs">
                <span class="pair" v-for="[key, value, kind] in row.pairs" :key="key">
                  <span class="key">{{ key }}=</span><span class="value" :class="kind ?? 'string'">{{ value }}</span>
                </span>
              </span>
            </div>
          </div>
        </div>
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

/* Sidebar
   -------------------------------------------------------------------------- */

.sidebar {
  display: flex;
  flex: 0 0 calc(228 * var(--u));
  flex-direction: column;
  padding: calc(12 * var(--u));
  border-right: 1px solid color-mix(in oklab, var(--content) 10%, transparent);
}

.brand {
  display: flex;
  align-items: center;
  gap: calc(10 * var(--u));
}

.logo {
  width: calc(34 * var(--u));
  height: calc(34 * var(--u));
}

.wordmark {
  font-family:
    ui-sans-serif,
    system-ui,
    -apple-system,
    "Segoe UI",
    Roboto,
    sans-serif;
  font-size: calc(34 * var(--u));
  font-weight: 100;
  line-height: 1;
}

.crumbs {
  display: flex;
  align-items: center;
  gap: calc(6 * var(--u));
  margin-top: calc(18 * var(--u));
  font-size: calc(11.5 * var(--u));
}

.crumb-link {
  color: var(--primary);
}

.crumb-sep {
  opacity: 0.4;
}

.host-pill {
  display: inline-flex;
  align-items: center;
  gap: calc(4 * var(--u));
  padding: calc(2 * var(--u)) calc(7 * var(--u));
  border: 1px solid var(--primary);
  border-radius: calc(4 * var(--u));
  color: var(--primary);
  font-size: calc(10 * var(--u));
}

.crumb-more {
  margin-left: auto;
  opacity: 0.5;
}

.group {
  margin-top: calc(14 * var(--u));
}

.group-head {
  display: flex;
  align-items: center;
  gap: calc(7 * var(--u));
  padding: calc(2 * var(--u)) calc(4 * var(--u));
  color: color-mix(in oklab, var(--content) 80%, transparent);
  font-size: calc(11.5 * var(--u));
  font-weight: 300;
}

.group-label {
  white-space: nowrap;
}

.merge-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(15 * var(--u));
  height: calc(15 * var(--u));
  border: 1px solid var(--primary);
  border-radius: calc(3 * var(--u));
  color: var(--primary);
}

.chevron {
  margin-left: auto;
  opacity: 0.45;
  font-size: calc(9 * var(--u));
}

/* ph:arrows-merge, drawn small enough that a glyph would not survive. */
.i-merge {
  width: calc(9 * var(--u));
  height: calc(9 * var(--u));
  background: currentColor;
  mask: no-repeat center / contain
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M12 21V11m0 0L6 5m6 6 6-6'/%3E%3C/svg%3E");
}

.i-group {
  width: calc(12 * var(--u));
  height: calc(12 * var(--u));
  background: currentColor;
  mask: no-repeat center / contain var(--glyph);
}

.i-pin {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M12 22v-8m0 0a5 5 0 1 0 0-10 5 5 0 0 0 0 10Z'/%3E%3C/svg%3E");
}

.i-stack {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='m12 3 9 5-9 5-9-5 9-5Zm9 9-9 5-9-5m18 4-9 5-9-5'/%3E%3C/svg%3E");
}

.i-box {
  --glyph: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'%3E%3Cpath fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' d='M3 7h18v10H3zM7 7v10m10-10v10'/%3E%3C/svg%3E");
}

.item {
  display: flex;
  align-items: center;
  gap: calc(8 * var(--u));
  height: calc(28 * var(--u));
  padding: 0 calc(10 * var(--u));
  border-radius: calc(5 * var(--u));
  font-size: calc(12 * var(--u));
}

.item.active {
  background: linear-gradient(
    100deg,
    color-mix(in oklab, var(--primary) 85%, transparent),
    color-mix(in oklab, var(--primary) 55%, transparent)
  );
  color: oklch(20% 0 0);
}

.dot {
  width: calc(6 * var(--u));
  height: calc(6 * var(--u));
  border-radius: 999px;
  background: var(--green);
}

.item.active .dot {
  background: oklch(30% 0.08 158);
}

.item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.health {
  margin-left: auto;
  color: var(--green);
  font-size: calc(10 * var(--u));
}

.item.active .health {
  color: oklch(25% 0.06 158);
}

/* Main panel
   -------------------------------------------------------------------------- */

.main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.topbar {
  display: flex;
  flex: 0 0 calc(46 * var(--u));
  align-items: center;
  gap: calc(8 * var(--u));
  padding: 0 calc(12 * var(--u));
}

.star {
  color: var(--secondary);
  font-size: calc(13 * var(--u));
}

.name-btn {
  display: inline-flex;
  align-items: center;
  gap: calc(5 * var(--u));
  padding: calc(3 * var(--u)) calc(8 * var(--u));
  border-radius: calc(4 * var(--u));
  background: var(--chip);
  font-size: calc(11 * var(--u));
}

.caret {
  font-size: calc(8 * var(--u));
  opacity: 0.7;
}

.topbar-health {
  margin-left: 0;
}

.image-tag {
  display: inline-flex;
  overflow: hidden;
  align-items: center;
  min-width: 0;
  gap: calc(6 * var(--u));
  white-space: nowrap;
  padding: calc(3 * var(--u)) calc(4 * var(--u)) calc(3 * var(--u)) calc(8 * var(--u));
  border-radius: calc(3 * var(--u));
  background: var(--chip);
  font-size: calc(10.5 * var(--u));
}

.image-name {
  overflow: hidden;
  text-overflow: ellipsis;
}

.copy {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: calc(14 * var(--u));
  height: calc(14 * var(--u));
  border-radius: calc(3 * var(--u));
  background: color-mix(in oklab, var(--content) 10%, transparent);
  color: color-mix(in oklab, var(--content) 45%, transparent);
  font-size: calc(9 * var(--u));
}

.cards {
  display: flex;
  flex: none;
  align-items: stretch;
  gap: calc(10 * var(--u));
  margin-left: auto;
  padding-left: calc(10 * var(--u));
}

.io-card {
  display: grid;
  align-items: center;
  gap: calc(3 * var(--u)) calc(6 * var(--u));
  grid-template-columns: auto auto calc(44 * var(--u)) auto calc(44 * var(--u));
  padding: calc(4 * var(--u)) calc(8 * var(--u));
  border-radius: calc(5 * var(--u));
  background: color-mix(in oklab, var(--content) 6%, transparent);
  font-size: calc(10.5 * var(--u));
  line-height: 1;
}

.io-icon {
  color: color-mix(in oklab, var(--content) 60%, transparent);
}

.io-num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.up {
  color: var(--primary);
}

.down {
  color: var(--secondary);
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: calc(2 * var(--u));
  width: calc(196 * var(--u));
  padding: calc(4 * var(--u)) calc(8 * var(--u));
  border-radius: calc(5 * var(--u));
}

.cpu {
  background: color-mix(in oklab, var(--primary) 12%, transparent);
}

.mem {
  background: color-mix(in oklab, var(--secondary) 12%, transparent);
}

.stat-head {
  display: flex;
  align-items: center;
  gap: calc(6 * var(--u));
  font-size: calc(10.5 * var(--u));
  line-height: 1;
}

.cpu .stat-icon {
  color: var(--primary);
}

.mem .stat-icon {
  color: var(--secondary);
}

.stat-sub {
  color: color-mix(in oklab, var(--content) 45%, transparent);
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
  border-radius: calc(2 * var(--u)) calc(2 * var(--u)) 0 0;
  opacity: 0.8;
}

.cpu .chart i {
  background: var(--primary);
}

.mem .chart i {
  background: var(--secondary);
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

.date {
  flex: none;
  padding: calc(1 * var(--u)) calc(7 * var(--u));
  border-radius: calc(3 * var(--u));
  background: var(--chip);
  color: var(--blue);
  white-space: nowrap;
}

.level {
  flex: none;
  width: calc(9 * var(--u));
  height: calc(9 * var(--u));
  margin-top: calc(4 * var(--u));
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

/* Narrow hero: drop the sidebar and the stat cards, and scale the type against a
   smaller design width so the entries stay legible instead of collapsing into
   texture. */
@container (max-width: 620px) {
  .hero-demo {
    --u: calc(100cqw / 520);
  }

  .sidebar,
  .cards {
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
  .chart i {
    transition: none;
  }
}
</style>
