<template>
  <div class="border-base-content/20 bg-base-200 relative h-full min-h-0 overflow-hidden rounded-lg border">
    <svg
      ref="svg"
      class="h-full w-full"
      :class="panning ? 'cursor-grabbing' : 'cursor-grab'"
      @wheel="onWheel"
      @pointerdown.self="onPointerDown"
      @click.self="selected = undefined"
    >
      <g :transform="transform">
        <g>
          <line
            v-for="(e, i) in edges"
            :key="i"
            :x1="nodeById[e.a].x"
            :y1="nodeById[e.a].y"
            :x2="nodeById[e.b].x"
            :y2="nodeById[e.b].y"
            class="stroke-primary"
            :stroke-width="isEdgeHighlighted(e) ? 1.6 : 1.2"
            :opacity="hovered ? (isEdgeHighlighted(e) ? 0.85 : 0.07) : 0.28"
          />
        </g>
        <g>
          <text
            v-for="(e, i) in edges"
            :key="i"
            :x="(nodeById[e.a].x + nodeById[e.b].x) / 2"
            :y="(nodeById[e.a].y + nodeById[e.b].y) / 2 - 3"
            text-anchor="middle"
            class="fill-primary pointer-events-none font-mono text-[9px]"
            :opacity="hovered ? (isEdgeHighlighted(e) ? 0.9 : 0.1) : 0.45"
          >
            {{ e.label }}
          </text>
        </g>
        <g>
          <g
            v-for="node in nodes"
            :key="node.id"
            :transform="`translate(${node.x},${node.y})`"
            class="cursor-pointer"
            :opacity="hovered && !isNodeNear(node.id) ? 0.18 : 1"
            @pointerenter="hovered = node.id"
            @pointerleave="hovered = undefined"
            @click.stop="selected = node.id"
          >
            <circle
              :r="22"
              fill="none"
              :stroke="strokeFor(node)"
              stroke-width="1"
              :opacity="hovered === node.id || selected === node.id ? 0.45 : 0"
            />
            <circle :r="17" class="fill-base-300" :stroke="strokeFor(node)" stroke-width="2.5" />
            <path
              :d="GLYPH_PATHS[node.glyph]"
              fill="none"
              :stroke="strokeFor(node)"
              stroke-width="1.6"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="pointer-events-none"
            />
            <text
              y="31"
              text-anchor="middle"
              class="fill-base-content pointer-events-none font-mono text-xs font-semibold"
            >
              {{ node.name.length > 16 ? node.name.slice(0, 15) + "…" : node.name }}
            </text>
          </g>
        </g>
      </g>
    </svg>

    <!-- legend -->
    <div
      class="border-base-content/20 bg-base-300 absolute top-3 left-3 flex flex-col gap-1.5 rounded-md border px-3.5 py-2.5 text-xs"
    >
      <div class="flex items-center gap-2 font-bold">Network Topology</div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-green size-2 rounded-full"></span>Running
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-base-content/40 size-2 rounded-full"></span>Stopped
      </div>
      <div class="text-base-content/70 flex items-center gap-2">
        <span class="bg-primary h-0.5 w-4 rounded"></span>Network link
      </div>
    </div>

    <!-- zoom controls -->
    <div class="absolute top-3 right-3 flex flex-col gap-1.5">
      <button class="btn btn-square btn-sm" @click="zoomIn" aria-label="Zoom in"><mdi:plus /></button>
      <button class="btn btn-square btn-sm" @click="zoomOut" aria-label="Zoom out"><mdi:minus /></button>
      <button class="btn btn-square btn-sm" @click="fit" aria-label="Fit"><mdi:fit-to-screen /></button>
    </div>

    <!-- counters -->
    <div
      class="border-base-content/20 bg-base-300/85 text-base-content/70 absolute bottom-3 left-3 flex gap-4 rounded-md border px-3 py-1.5 font-mono text-xs"
    >
      <span>
        <span class="text-base-content font-semibold">{{ nodes.length }}</span>
        containers
      </span>
      <span>
        <span class="text-base-content font-semibold">{{ networkCount }}</span>
        networks
      </span>
    </div>
    <div
      class="border-base-content/20 bg-base-300/85 text-base-content/70 absolute right-3 bottom-3 rounded-md border px-2.5 py-1 font-mono text-xs"
    >
      {{ zoomPercent }}%
    </div>

    <!-- selected container card -->
    <div
      v-if="selectedNode"
      class="border-base-content/20 bg-base-300 absolute top-3 right-14 w-64 rounded-md border p-3.5 text-xs"
    >
      <div class="mb-2 flex items-center gap-2 font-mono text-sm font-semibold">
        <span
          class="size-2 rounded-full"
          :class="selectedNode.state === 'running' ? 'bg-green' : 'bg-base-content/40'"
        ></span>
        {{ selectedNode.name }}
      </div>
      <div class="mb-2 flex flex-wrap gap-1.5">
        <span
          v-for="net in selectedNode.networks"
          :key="net"
          class="text-primary bg-primary/10 rounded px-1.5 py-0.5 font-mono text-[10px]"
        >
          {{ net }}
        </span>
      </div>
      <router-link class="btn btn-primary btn-xs" :to="{ name: '/container/[id]', params: { id: selectedNode.id } }">
        View logs
      </router-link>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Container } from "@/models/Container";
import { useGraphPanZoom } from "@/composable/graphPanZoom";
import type { GraphEdge, GraphNode } from "@/composable/graphModel";
import {
  GLYPH_PATHS,
  edgesForNetwork,
  glyphFor,
  graphBounds,
  linkableNetworks,
  runForceLayout,
  seededRandom,
} from "@/composable/graphModel";

const { containers } = defineProps<{ containers: Container[] }>();

const hovered = ref<string>();
const selected = ref<string>();

const model = computed(() => {
  const random = seededRandom();
  const nodes: GraphNode[] = containers.map((c) => ({
    id: c.id,
    name: c.name,
    state: c.state,
    glyph: glyphFor(c),
    networks: linkableNetworks(c),
    x: 0,
    y: 0,
  }));

  const members = new Map<string, string[]>();
  for (const n of nodes) {
    for (const net of n.networks) {
      if (!members.has(net)) members.set(net, []);
      members.get(net)!.push(n.id);
    }
  }

  const edges: GraphEdge[] = [];
  const seen = new Set<string>();
  const addEdge = (a: string, b: string, label: string) => {
    if (a === b) return;
    const key = a < b ? `${a}|${b}` : `${b}|${a}`;
    if (seen.has(key)) return;
    seen.add(key);
    edges.push({ a, b, label });
  };
  for (const [net, ids] of members) edgesForNetwork(net, ids, addEdge);

  const neighbors = new Map<string, Set<string>>();
  for (const e of edges) {
    if (!neighbors.has(e.a)) neighbors.set(e.a, new Set());
    if (!neighbors.has(e.b)) neighbors.set(e.b, new Set());
    neighbors.get(e.a)!.add(e.b);
    neighbors.get(e.b)!.add(e.a);
  }

  // Cluster nodes around their first network so stacks stay together. Nodes
  // without a linkable network (bridge-only, host, none) each get their own
  // anchor so repulsion can't fling them far away and wreck the fit.
  const clusterOf = (n: GraphNode) => n.networks[0] ?? `__solo__${n.id}`;
  const clusterNames = [...new Set([...members.keys(), ...nodes.map(clusterOf)])];
  const cols = Math.max(Math.ceil(Math.sqrt(clusterNames.length || 1)), 1);
  const rows = Math.max(Math.ceil(clusterNames.length / cols), 1);
  const width = cols * 280;
  const height = rows * 240;
  const centers = new Map<string, { x: number; y: number }>();
  clusterNames.forEach((name, i) => {
    centers.set(name, {
      x: ((i % cols) + 0.5) * (width / cols),
      y: (Math.floor(i / cols) + 0.5) * (height / rows),
    });
  });
  for (const n of nodes) {
    const center = centers.get(clusterOf(n))!;
    n.x = center.x + (random() - 0.5) * 160;
    n.y = center.y + (random() - 0.5) * 160;
  }
  runForceLayout(nodes, edges, centers, clusterOf, { width, height });

  return { nodes, edges, neighbors, networkCount: members.size };
});

const nodes = computed(() => model.value.nodes);
const edges = computed(() => model.value.edges);
const networkCount = computed(() => model.value.networkCount);
const nodeById = computed(() => Object.fromEntries(nodes.value.map((n) => [n.id, n])));
const selectedNode = computed(() => (selected.value ? nodeById.value[selected.value] : undefined));

function strokeFor(node: GraphNode) {
  return node.state === "running"
    ? "var(--color-green)"
    : "color-mix(in oklab, var(--color-base-content) 40%, transparent)";
}

function isEdgeHighlighted(e: GraphEdge) {
  return hovered.value !== undefined && (e.a === hovered.value || e.b === hovered.value);
}

function isNodeNear(id: string) {
  if (!hovered.value) return true;
  if (id === hovered.value) return true;
  return model.value.neighbors.get(hovered.value)?.has(id) ?? false;
}

const svg = useTemplateRef<SVGSVGElement>("svg");
const { transform, zoomPercent, panning, fit, zoomIn, zoomOut, onWheel, onPointerDown } = useGraphPanZoom(svg, () =>
  graphBounds(nodes.value),
);

watch(
  () => nodes.value.length,
  () => nextTick(fit),
  { immediate: false },
);
onMounted(() => nextTick(fit));
</script>
