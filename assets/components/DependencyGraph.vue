<template>
  <div class="border-base-content/20 bg-base-200 relative h-full min-h-0 overflow-hidden rounded-lg border">
    <svg
      ref="svg"
      class="h-full w-full"
      :class="panning ? 'cursor-grabbing' : 'cursor-grab'"
      @wheel="onWheel"
      @pointerdown.self="onPointerDown"
    >
      <g :transform="transform">
        <g>
          <g v-for="card in cards" :key="card.name">
            <rect
              :x="card.x"
              :y="card.y"
              :width="card.w"
              :height="card.h"
              rx="8"
              class="fill-base-300/80"
              :stroke="
                hoveredStack === card.name
                  ? 'var(--color-primary)'
                  : 'color-mix(in oklab, var(--color-base-content) 18%, transparent)'
              "
              stroke-width="1"
            />
            <text :x="card.x + 12" :y="card.y + 16" class="fill-primary font-mono text-[10px] font-semibold">
              {{ card.name }}
            </text>
          </g>
        </g>
        <g>
          <path
            v-for="(link, i) in links"
            :key="i"
            :d="linkPath(link)"
            fill="none"
            :stroke-dasharray="link.kind === 'network' ? '5 4' : undefined"
            :stroke="
              isLinkHighlighted(link)
                ? 'var(--color-primary)'
                : link.kind === 'network'
                  ? 'color-mix(in oklab, var(--color-base-content) 35%, transparent)'
                  : 'color-mix(in oklab, var(--color-primary) 55%, transparent)'
            "
            :stroke-width="isLinkHighlighted(link) ? 2 : 1.5"
          />
        </g>
        <g>
          <g
            v-for="node in nodes"
            :key="node.id"
            :transform="`translate(${node.x},${node.y})`"
            class="cursor-pointer"
            @pointerenter="hovered = node.id"
            @pointerleave="hovered = undefined"
            @click.stop="openContainer(node.id)"
          >
            <circle :r="13" class="fill-base-300" :stroke="strokeFor(node)" stroke-width="2.2" />
            <path
              :d="GLYPH_PATHS[node.glyph]"
              fill="none"
              :stroke="strokeFor(node)"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
              transform="scale(0.8)"
              class="pointer-events-none"
            />
            <text
              y="24"
              text-anchor="middle"
              class="pointer-events-none font-mono text-[9px]"
              :class="hovered === node.id ? 'fill-base-content' : 'fill-base-content/70'"
            >
              {{ node.name.length > 14 ? node.name.slice(0, 13) + "…" : node.name }}
            </text>
          </g>
        </g>
      </g>
    </svg>

    <!-- legend -->
    <div
      class="border-base-content/20 bg-base-300 absolute bottom-3 left-3 flex items-center gap-4 rounded-md border px-3.5 py-2 text-xs"
    >
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-green size-2 rounded-full"></span>Running
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-red size-2 rounded-full"></span>Stopped
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-base-content/40 size-2 rounded-full"></span>Other
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-primary/60 h-0.5 w-4 rounded"></span>depends_on
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="border-base-content/40 w-4 border-t-2 border-dashed"></span>Shared network
      </div>
    </div>

    <!-- zoom controls -->
    <div class="absolute top-3 right-3 flex flex-col gap-1.5">
      <button class="btn btn-square btn-sm" @click="zoomIn" aria-label="Zoom in"><mdi:plus /></button>
      <button class="btn btn-square btn-sm" @click="zoomOut" aria-label="Zoom out"><mdi:minus /></button>
      <button class="btn btn-square btn-sm" @click="fit" aria-label="Fit"><mdi:fit-to-screen /></button>
    </div>
    <div
      class="border-base-content/20 bg-base-300/85 text-base-content/70 absolute right-3 bottom-3 rounded-md border px-2.5 py-1 font-mono text-xs"
    >
      {{ zoomPercent }}%
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { useGraphPanZoom } from "@/composable/graphPanZoom";
import type { GraphNode } from "@/composable/graphModel";
import { GLYPH_PATHS, glyphFor, graphBounds, linkableNetworks, seededRandom } from "@/composable/graphModel";

const { containers } = defineProps<{ containers: Container[] }>();

const router = useRouter();
const hovered = ref<string>();

interface StackCard {
  name: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

interface DepLink {
  a: string;
  b: string;
  kind: "depends" | "network";
}

const NODE_CELL_W = 96;
const NODE_CELL_H = 64;
const CARD_PAD = 18;
const CARD_TITLE_H = 24;

const model = computed(() => {
  const random = seededRandom(1337);

  const stacks = new Map<string, Container[]>();
  const singles: Container[] = [];
  for (const c of containers) {
    const ns = c.namespace;
    if (ns) {
      if (!stacks.has(ns)) stacks.set(ns, []);
      stacks.get(ns)!.push(c);
    } else {
      singles.push(c);
    }
  }

  const cards: StackCard[] = [];
  const nodes: GraphNode[] = [];
  const stackNames = [...stacks.keys()];
  const total = stackNames.length + Math.ceil(singles.length / 4);
  const cols = Math.max(Math.min(Math.ceil(Math.sqrt(total * 1.6)), 5), 1);
  const colW = 340;
  const rowH = 260;

  stackNames.forEach((name, i) => {
    const cs = stacks.get(name)!;
    const perRow = cs.length <= 4 ? 2 : 3;
    const rows = Math.ceil(cs.length / perRow);
    const w = CARD_PAD * 2 + perRow * NODE_CELL_W;
    const h = CARD_TITLE_H + CARD_PAD + rows * NODE_CELL_H + 6;
    const x = (i % cols) * colW + 40 + (random() - 0.5) * 60;
    const y = Math.floor(i / cols) * rowH + 40 + (random() - 0.5) * 50;
    cards.push({ name, x, y, w, h });
    cs.forEach((c, j) => {
      nodes.push({
        id: c.id,
        name: c.name,
        state: c.state,
        glyph: glyphFor(c),
        networks: linkableNetworks(c),
        stack: name,
        x: x + CARD_PAD + (j % perRow) * NODE_CELL_W + NODE_CELL_W / 2,
        y: y + CARD_TITLE_H + CARD_PAD + Math.floor(j / perRow) * NODE_CELL_H + 14,
      });
    });
  });
  // Standalone containers pack densely below the stack cards instead of one
  // per grid cell, which sprawled the canvas.
  const singlesPerRow = Math.max(cols * 2, 4);
  const singlesTop = Math.ceil(stackNames.length / cols) * rowH + 60;
  singles.forEach((c, i) => {
    nodes.push({
      id: c.id,
      name: c.name,
      state: c.state,
      glyph: glyphFor(c),
      networks: linkableNetworks(c),
      x: (i % singlesPerRow) * 165 + 110 + (random() - 0.5) * 40,
      y: singlesTop + Math.floor(i / singlesPerRow) * 110 + (random() - 0.5) * 30,
    });
  });

  // depends_on edges from compose labels, resolved within the same project
  const links: DepLink[] = [];
  const seen = new Set<string>();
  const addLink = (a: string, b: string, kind: DepLink["kind"]) => {
    if (a === b) return;
    const key = `${kind}:${a < b ? a + b : b + a}`;
    if (seen.has(key)) return;
    seen.add(key);
    links.push({ a, b, kind });
  };
  const byService = new Map<string, string>();
  for (const c of containers) {
    const project = c.labels["com.docker.compose.project"];
    const service = c.labels["com.docker.compose.service"];
    if (project && service) byService.set(`${project}/${service}`, c.id);
  }
  for (const c of containers) {
    const project = c.labels["com.docker.compose.project"];
    const dependsOn = c.labels["com.docker.compose.depends_on"];
    if (!project || !dependsOn) continue;
    for (const entry of dependsOn.split(",")) {
      const service = entry.split(":")[0].trim();
      if (!service) continue;
      const target = byService.get(`${project}/${service}`);
      if (target) addLink(c.id, target, "depends");
    }
  }

  // dashed edges between stacks sharing a network (only real docker networks)
  const stackOf = new Map(nodes.map((n) => [n.id, n.stack]));
  const netMembers = new Map<string, string[]>();
  for (const c of containers) {
    for (const net of c.networks) {
      if (["bridge", "host", "none", "ingress"].includes(net)) continue;
      if (!netMembers.has(net)) netMembers.set(net, []);
      netMembers.get(net)!.push(c.id);
    }
  }
  for (const ids of netMembers.values()) {
    const byStack = new Map<string | undefined, string>();
    for (const id of ids) {
      const stack = stackOf.get(id);
      if (!byStack.has(stack)) byStack.set(stack, id);
    }
    const representatives = [...byStack.values()];
    for (let i = 1; i < representatives.length; i++) addLink(representatives[0], representatives[i], "network");
  }

  return { cards, nodes, links };
});

const cards = computed(() => model.value.cards);
const nodes = computed(() => model.value.nodes);
const links = computed(() => model.value.links);
const nodeById = computed(() => Object.fromEntries(nodes.value.map((n) => [n.id, n])));
const hoveredStack = computed(() => (hovered.value ? nodeById.value[hovered.value]?.stack : undefined));

function strokeFor(node: GraphNode) {
  if (node.state === "running") return "var(--color-green)";
  if (node.state === "exited" || node.state === "dead") return "var(--color-red)";
  return "color-mix(in oklab, var(--color-base-content) 40%, transparent)";
}

function isLinkHighlighted(link: DepLink) {
  return hovered.value !== undefined && (link.a === hovered.value || link.b === hovered.value);
}

function linkPath(link: DepLink) {
  const a = nodeById.value[link.a];
  const b = nodeById.value[link.b];
  if (!a || !b) return "";
  const mx = (a.x + b.x) / 2;
  const my = (a.y + b.y) / 2;
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const n = Math.hypot(dx, dy) || 1;
  const off = Math.min(n * 0.18, 55);
  return `M${a.x},${a.y} Q${mx - (dy / n) * off},${my + (dx / n) * off} ${b.x},${b.y}`;
}

function openContainer(id: string) {
  router.push({ name: "/container/[id]", params: { id } });
}

const svg = useTemplateRef<SVGSVGElement>("svg");
const { transform, zoomPercent, panning, fit, zoomIn, zoomOut, onWheel, onPointerDown } = useGraphPanZoom(svg, () => {
  const points = nodes.value.map(({ x, y }) => ({ x, y }));
  for (const c of cards.value) {
    points.push({ x: c.x, y: c.y }, { x: c.x + c.w, y: c.y + c.h });
  }
  return graphBounds(points);
});

watch(
  () => nodes.value.length,
  () => nextTick(fit),
);
onMounted(() => nextTick(fit));
</script>
