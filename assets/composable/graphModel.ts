import { Container } from "@/models/Container";

// Networks that connect everything (or nothing) and would only add noise as links.
const EXCLUDED_NETWORKS = new Set(["bridge", "host", "none", "ingress"]);

export type NodeGlyph = "db" | "cache" | "web" | "worker" | "code" | "app";

export interface GraphNode {
  id: string;
  name: string;
  state: string;
  glyph: NodeGlyph;
  networks: string[];
  stack?: string;
  x: number;
  y: number;
}

export interface GraphEdge {
  a: string; // node id
  b: string; // node id
  label: string;
}

export function glyphFor(container: Container): NodeGlyph {
  const subject = `${container.image} ${container.name}`.toLowerCase();
  if (/mariadb|mysql|postgres|mongo|clickhouse|sqlite|elasticsearch|minio|sftp|db\b|database/.test(subject))
    return "db";
  if (/redis|valkey|memcache|rabbitmq|nats|kafka/.test(subject)) return "cache";
  if (/nginx|caddy|traefik|haproxy|httpd|apache/.test(subject)) return "web";
  if (/worker|cron|beat|scheduler|runner/.test(subject)) return "worker";
  if (/api|backend|php|fpm|gateway/.test(subject)) return "code";
  return "app";
}

/** Networks a container should be linked by; falls back to its compose project when
 * the backend didn't report networks (agents and k8s hosts). */
export function linkableNetworks(container: Container): string[] {
  const nets = container.networks.filter((n) => !EXCLUDED_NETWORKS.has(n));
  if (nets.length > 0) return nets;
  return container.namespace ? [container.namespace] : [];
}

/** Pairwise for small networks, hub-and-chain for large ones to keep the graph readable. */
export function edgesForNetwork(net: string, members: string[], addEdge: (a: string, b: string, net: string) => void) {
  if (members.length <= 4) {
    for (let i = 0; i < members.length; i++)
      for (let j = i + 1; j < members.length; j++) addEdge(members[i], members[j], net);
  } else {
    for (let i = 1; i < members.length; i++) addEdge(members[0], members[i], net);
    for (let i = 1; i < members.length - 1; i++) addEdge(members[i], members[i + 1], net);
  }
}

/** Deterministic seeded PRNG so the layout is stable across renders. */
export function seededRandom(seed = 42) {
  let state = seed;
  return () => {
    state = (state * 16807) % 2147483647;
    return state / 2147483647;
  };
}

export interface ForceLayoutOptions {
  width: number;
  height: number;
  clusterStrength?: number;
  springLength?: number;
  repulsionRadius?: number;
}

/** One-shot force simulation: repulsion + edge springs + pull toward per-cluster centers. */
export function runForceLayout(
  nodes: GraphNode[],
  edges: GraphEdge[],
  clusterCenters: Map<string, { x: number; y: number }>,
  clusterOf: (node: GraphNode) => string | undefined,
  { width, height, clusterStrength = 0.02, springLength = 100, repulsionRadius = 175 }: ForceLayoutOptions,
) {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const velocity = new Map(nodes.map((n) => [n.id, { vx: 0, vy: 0 }]));
  const iterations = 320;

  for (let it = 0; it < iterations; it++) {
    const alpha = 1 - it / iterations;
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const a = nodes[i];
        const b = nodes[j];
        let dx = a.x - b.x;
        let dy = a.y - b.y;
        const d2 = dx * dx + dy * dy || 1;
        if (d2 < repulsionRadius * repulsionRadius) {
          const f = (2200 / d2) * alpha * 4;
          dx *= f;
          dy *= f;
          const va = velocity.get(a.id)!;
          const vb = velocity.get(b.id)!;
          va.vx += dx;
          va.vy += dy;
          vb.vx -= dx;
          vb.vy -= dy;
        }
      }
    }
    for (const e of edges) {
      const a = byId.get(e.a)!;
      const b = byId.get(e.b)!;
      const dx = b.x - a.x;
      const dy = b.y - a.y;
      const d = Math.sqrt(dx * dx + dy * dy) || 1;
      const f = ((d - springLength) / d) * 0.05 * alpha * 4;
      const va = velocity.get(a.id)!;
      const vb = velocity.get(b.id)!;
      va.vx += dx * f;
      va.vy += dy * f;
      vb.vx -= dx * f;
      vb.vy -= dy * f;
    }
    for (const n of nodes) {
      const v = velocity.get(n.id)!;
      const cluster = clusterOf(n);
      const center = cluster ? clusterCenters.get(cluster) : undefined;
      if (center) {
        v.vx += (center.x - n.x) * clusterStrength * alpha;
        v.vy += (center.y - n.y) * clusterStrength * alpha;
      }
      v.vx += (width / 2 - n.x) * 0.001 * alpha;
      v.vy += (height / 2 - n.y) * 0.001 * alpha;
      n.x += v.vx * 0.5;
      n.y += v.vy * 0.5;
      v.vx *= 0.4;
      v.vy *= 0.4;
    }
  }
}

export function graphBounds(points: { x: number; y: number }[]): {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
} {
  if (points.length === 0) return { minX: 0, minY: 0, maxX: 1, maxY: 1 };
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const p of points) {
    minX = Math.min(minX, p.x);
    maxX = Math.max(maxX, p.x);
    minY = Math.min(minY, p.y);
    maxY = Math.max(maxY, p.y);
  }
  return { minX, minY, maxX, maxY };
}

/** Small stroke-based glyph paths, drawn inside a node circle. */
export const GLYPH_PATHS: Record<NodeGlyph, string> = {
  db: "M-5,-5 a5,2.4 0 0,0 10,0 a5,2.4 0 0,0 -10,0 v10 a5,2.4 0 0,0 10,0 v-10 M-5,0 a5,2.4 0 0,0 10,0",
  code: "M-4.5,-3 l-2.6,3 2.6,3 M4.5,-3 l2.6,3 -2.6,3 M1.5,-5 l-3,10",
  web: "M0,-6 a6,6 0 1,0 0,12 a6,6 0 1,0 0,-12 M-6,0 h12 M0,-6 a9,9 0 0,0 0,12 a9,9 0 0,0 0,-12",
  worker:
    "M0,-2.2 a2.2,2.2 0 1,0 0,4.4 a2.2,2.2 0 1,0 0,-4.4 M0,-6.4 v2.4 M0,4 v2.4 M-6.4,0 h2.4 M4,0 h2.4 M-4.5,-4.5 l1.7,1.7 M2.8,2.8 l1.7,1.7 M-4.5,4.5 l1.7,-1.7 M2.8,-2.8 l1.7,-1.7",
  cache: "M1.5,-7 L-4,1 h3 L-1.5,7 L4,-1 h-3 Z",
  app: "M0,-6.5 l6,3.2 v6.6 l-6,3.2 -6,-3.2 v-6.6 Z M-6,-3.3 l6,3.2 6,-3.2 M0,-0.1 v6.6",
};
