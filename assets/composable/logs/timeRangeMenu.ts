import { isStopped, type Container } from "@/models/Container";
import { RELATIVE_SPANS, formatRange, timeRangeRoute, type RelativeSpan, type TimeRange } from "./timeRange";

export type CustomRangeRequest = { container: Container; range: TimeRange; anchor?: Date };

// The custom range dialog lives in the view, outside the hover menu that opens
// it (a menu closes on pointer leave, and a dialog inside it would go too).
const customRequest = shallowRef<CustomRangeRequest>();

export const useCustomRangeRequest = () => customRequest;

/**
 * The time range rows the chip's hover menu and the phone's toolbar menu both
 * list, and the label the chip shows. `anchor` is the moment a frozen view was
 * opened on when it is not a range, so nothing reads as active there.
 */
export function useTimeRangeMenu(
  container: MaybeRefOrGetter<Container>,
  range: MaybeRefOrGetter<TimeRange>,
  anchor?: MaybeRefOrGetter<Date | undefined>,
) {
  const { t } = useI18n();
  const router = useRouter();

  // Adjusting a range is the same visit; entering or leaving one is a new one,
  // so Back returns to where the person was before.
  function go(next: TimeRange) {
    const to = timeRangeRoute(toValue(container).id, next);
    if (next.kind === toValue(range).kind && !toValue(anchor)) router.replace(to);
    else router.push(to);
  }

  // A stopped container has nothing left to stream, so its "live" view is just
  // the logs it wrote and nothing should claim otherwise.
  const stopped = computed(() => isStopped(toValue(container)));

  const rows = computed(() => {
    const current = toValue(range);
    const frozen = !!toValue(anchor);
    return [
      {
        key: "live",
        label: t("time-range.live"),
        active: current.kind === "live" && !frozen && !stopped.value,
        run: () => go({ kind: "live" }),
      },
      ...(Object.keys(RELATIVE_SPANS) as RelativeSpan[]).map((span) => ({
        key: span,
        label: t(`time-range.last-${span}`),
        active: current.kind === "since" && current.relative === span,
        run: () => go({ kind: "since", since: new Date(Date.now() - RELATIVE_SPANS[span]), relative: span }),
      })),
    ];
  });

  const label = computed(() => {
    const current = toValue(range);
    switch (current.kind) {
      case "live":
        return toValue(anchor) || stopped.value ? t("time-range.title") : t("time-range.live");
      case "since":
        return current.relative
          ? t("time-range.last-short", { span: current.relative })
          : t("time-range.since-short", {
              time: current.since.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" }),
            });
      case "range":
        return formatRange(current.from, current.until);
    }
  });

  /** Live and "since" both follow new lines, which the chip marks with a dot. */
  const following = computed(() => toValue(range).kind !== "range" && !toValue(anchor) && !stopped.value);

  function custom() {
    customRequest.value = { container: toValue(container), range: toValue(range), anchor: toValue(anchor) };
  }

  return { rows, label, following, custom, go };
}
