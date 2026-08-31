import type { MaybeRefOrGetter } from "vue";

export interface GraphBounds {
  minX: number;
  minY: number;
  maxX: number;
  maxY: number;
}

/**
 * Shared pan/zoom behavior for SVG graph views (topology, dependency map).
 * Returns a reactive transform for the root <g> plus handlers to wire on the <svg>.
 */
export function useGraphPanZoom(
  svgSource: MaybeRefOrGetter<SVGSVGElement | null | undefined>,
  bounds: () => GraphBounds,
) {
  const tx = ref(0);
  const ty = ref(0);
  const scale = ref(1);
  const panning = ref(false);
  let panStart: { x: number; y: number } | undefined;

  const transform = computed(() => `translate(${tx.value},${ty.value}) scale(${scale.value})`);
  const zoomPercent = computed(() => Math.round(scale.value * 100));

  function zoomAt(factor: number, cx?: number, cy?: number) {
    const svg = toValue(svgSource);
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    const mx = cx ?? box.width / 2;
    const my = cy ?? box.height / 2;
    const next = Math.min(Math.max(scale.value * factor, 0.2), 4);
    tx.value = mx - (mx - tx.value) * (next / scale.value);
    ty.value = my - (my - ty.value) * (next / scale.value);
    scale.value = next;
  }

  function fit() {
    const svg = toValue(svgSource);
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    if (box.width === 0 || box.height === 0) return;
    const b = bounds();
    const w = Math.max(b.maxX - b.minX, 1);
    const h = Math.max(b.maxY - b.minY, 1);
    const pad = 60;
    scale.value = Math.min((box.width - 40) / (w + pad * 2), (box.height - 40) / (h + pad * 2), 1.4);
    tx.value = box.width / 2 - (scale.value * (b.minX + b.maxX)) / 2;
    ty.value = box.height / 2 - (scale.value * (b.minY + b.maxY)) / 2;
  }

  function onWheel(e: WheelEvent) {
    e.preventDefault();
    const svg = toValue(svgSource);
    if (!svg) return;
    const box = svg.getBoundingClientRect();
    zoomAt(e.deltaY < 0 ? 1.12 : 1 / 1.12, e.clientX - box.left, e.clientY - box.top);
  }

  function onPointerDown(e: PointerEvent) {
    panStart = { x: e.clientX - tx.value, y: e.clientY - ty.value };
    panning.value = true;
  }

  function onPointerMove(e: PointerEvent) {
    if (!panStart) return;
    tx.value = e.clientX - panStart.x;
    ty.value = e.clientY - panStart.y;
  }

  function onPointerUp() {
    panStart = undefined;
    panning.value = false;
  }

  useEventListener(window, "pointermove", onPointerMove);
  useEventListener(window, "pointerup", onPointerUp);

  return {
    transform,
    zoomPercent,
    panning,
    fit,
    zoomIn: () => zoomAt(1.25),
    zoomOut: () => zoomAt(1 / 1.25),
    onWheel,
    onPointerDown,
  };
}
