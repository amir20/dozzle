import type { RouteLocationRaw } from "vue-router";

/**
 * A moment in a container's log stream. `logId` pinpoints an exact line when
 * the caller knows it (Dozzle stamps the same FNV-32a hash Cloud returns);
 * `query` pre-fills the search box so the reader lands with the term already
 * highlighted.
 */
export type LogMoment = {
  containerId: string;
  date: Date;
  logId?: string | number;
  query?: string;
};

/**
 * Build the route for "show me the lines": the historical view of one
 * container, scrolled to the moment something happened.
 *
 * This is the one move Dozzle can make that Cloud cannot, so alerts, findings
 * and assistant answers all end here rather than linking back out. Everything
 * that wants it builds the destination the same way, through this.
 */
export function logMomentRoute({ containerId, date, logId, query }: LogMoment): RouteLocationRaw {
  const params: Record<string, string> = {};
  // A logId of 0 means "not log-anchored" — metric and event alerts carry no
  // line — so it is dropped rather than sent as a target that matches nothing.
  if (logId !== undefined && logId !== 0 && logId !== "0") {
    params.logId = String(logId);
  }
  if (query) {
    params.q = query;
  }

  return {
    name: "/container/[id].time.[datetime]",
    params: { id: containerId, datetime: date.toISOString() },
    query: params,
  };
}

/**
 * Navigate to a moment, and return the resolved absolute URL so callers that
 * want to copy or share it don't rebuild the route themselves.
 */
export function useLogJump() {
  const router = useRouter();

  return {
    jumpTo(moment: LogMoment) {
      return router.push(logMomentRoute(moment));
    },
    hrefFor(moment: LogMoment) {
      const { href } = router.resolve(logMomentRoute(moment));
      return new URL(href, window.location.origin).href;
    },
  };
}
