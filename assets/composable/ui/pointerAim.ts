/**
 * A "safe triangle" for hover panels that hang off a row in a list.
 *
 * The straight line from a row to its panel usually crosses the rows below or above it, and
 * each of those wants to open its own panel on hover, which closes the one the pointer was
 * heading for. While the pointer is still moving inside the triangle between the point it
 * left its row and the near edge of the panel, it counts as aiming at that panel: the owner
 * keeps it open and every other row holds off. Once the pointer stops or strays outside,
 * the aim lapses and normal hover takes over.
 *
 * One aim is shared by the whole page, since only one hover panel is open at a time.
 */

/** How long the pointer may rest before it no longer counts as travelling. Also a sensible
 * interval for re-checking while held off. */
export const AIM_IDLE = 150;
/** Pushes the apex away from the panel, so a pointer that leaves the row heading nearly
 * straight up or down still lands inside the triangle. */
const SLACK = 8;

type Point = { x: number; y: number };

let aim: { owner: object; apex: Point; rect: DOMRect } | null = null;
let last = { x: 0, y: 0, at: -Infinity };
let tracking = false;

function track(event: MouseEvent) {
  last = { x: event.clientX, y: event.clientY, at: performance.now() };
}

/** Starts an aim from where `from` left the trigger toward `target`, owned by `owner`. */
export function aimAt(owner: object, from: MouseEvent, target: HTMLElement) {
  track(from);
  // Never removed: one passive listener for the life of the page costs less than
  // adding and dropping it every time the pointer crosses a row.
  if (!tracking) {
    window.addEventListener("pointermove", track, { passive: true });
    tracking = true;
  }
  const rect = target.getBoundingClientRect();
  const towardRight = rect.left >= from.clientX;
  aim = { owner, apex: { x: from.clientX + (towardRight ? -SLACK : SLACK), y: from.clientY }, rect };
}

/** Drops the aim if `owner` holds it: the pointer arrived, or the panel closed. */
export function clearAim(owner: object) {
  if (aim?.owner === owner) aim = null;
}

/** The owner the pointer is travelling toward right now, if any. */
export function aimOwner(): object | null {
  if (!aim || performance.now() - last.at > AIM_IDLE) return null;
  const { apex, rect } = aim;
  const edge = rect.left >= apex.x ? rect.left : rect.right;
  return inTriangle(last, apex, { x: edge, y: rect.top }, { x: edge, y: rect.bottom }) ? aim.owner : null;
}

function inTriangle(p: Point, a: Point, b: Point, c: Point) {
  const side = (u: Point, v: Point) => (v.x - u.x) * (p.y - u.y) - (v.y - u.y) * (p.x - u.x);
  const d1 = side(a, b);
  const d2 = side(b, c);
  const d3 = side(c, a);
  return !((d1 < 0 || d2 < 0 || d3 < 0) && (d1 > 0 || d2 > 0 || d3 > 0));
}
