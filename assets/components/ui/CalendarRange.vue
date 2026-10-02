<template>
  <div class="flex flex-col gap-2 select-none">
    <div class="flex items-center justify-between">
      <button
        type="button"
        class="btn btn-ghost btn-square btn-sm"
        :aria-label="previousLabel"
        :title="previousLabel"
        @click="shift(-1)"
      >
        <mdi:chevron-left />
      </button>
      <span class="text-sm font-semibold">{{ monthLabel }}</span>
      <button
        type="button"
        class="btn btn-ghost btn-square btn-sm"
        :aria-label="nextLabel"
        :title="nextLabel"
        :disabled="atLastMonth"
        @click="shift(1)"
      >
        <mdi:chevron-right />
      </button>
    </div>
    <div class="grid grid-cols-7 gap-y-0.5 text-center" role="grid">
      <span v-for="day in weekdays" :key="day" class="text-base-content/40 py-1 text-xs">{{ day }}</span>
      <button
        v-for="day in days"
        :key="day.key"
        type="button"
        class="h-8 text-sm tabular-nums transition-colors"
        :class="day.classes"
        :disabled="day.disabled"
        :aria-pressed="day.endpoint"
        :aria-label="day.title"
        @click="pick(day.date)"
      >
        {{ day.date.getDate() }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
// Two clicks pick a range of days: the first sets the start, the second the end,
// and a click after that starts over. Days are local midnights; the time of day
// is someone else's business.
const start = defineModel<Date | undefined>("start");
const end = defineModel<Date | undefined>("end");

const {
  max,
  previousLabel = "Previous month",
  nextLabel = "Next month",
} = defineProps<{
  /** The last day that can be picked; later days are disabled. */
  max?: Date;
  previousLabel?: string;
  nextLabel?: string;
}>();

const midnight = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate());
const same = (a?: Date, b?: Date) => !!a && !!b && a.getTime() === b.getTime();

const shown = ref(midnight(end.value ?? start.value ?? new Date()));
watch([start, end], () => {
  const anchor = end.value ?? start.value;
  if (anchor && (anchor.getMonth() !== shown.value.getMonth() || anchor.getFullYear() !== shown.value.getFullYear())) {
    shown.value = midnight(anchor);
  }
});

const monthLabel = computed(() => shown.value.toLocaleDateString(undefined, { month: "long", year: "numeric" }));

// Where the week starts follows the browser's locale when it says.
const firstDay = (() => {
  try {
    const locale = new Intl.Locale(navigator.language) as Intl.Locale & {
      getWeekInfo?: () => { firstDay: number };
      weekInfo?: { firstDay: number };
    };
    return (locale.getWeekInfo?.() ?? locale.weekInfo)?.firstDay ?? 7;
  } catch {
    return 7;
  }
})();
const firstWeekday = firstDay % 7; // 0 is Sunday, as Date.getDay counts

const weekdays = computed(() => {
  const format = new Intl.DateTimeFormat(undefined, { weekday: "narrow" });
  // 2023-01-01 was a Sunday.
  return Array.from({ length: 7 }, (_, i) => format.format(new Date(2023, 0, 1 + ((firstWeekday + i) % 7))));
});

const atLastMonth = computed(() => {
  if (!max) return false;
  return shown.value.getFullYear() * 12 + shown.value.getMonth() >= max.getFullYear() * 12 + max.getMonth();
});

function shift(months: number) {
  shown.value = new Date(shown.value.getFullYear(), shown.value.getMonth() + months, 1);
}

const days = computed(() => {
  const year = shown.value.getFullYear();
  const month = shown.value.getMonth();
  const first = new Date(year, month, 1);
  const lead = (first.getDay() - firstWeekday + 7) % 7;
  const today = midnight(new Date());
  const limit = max ? midnight(max) : undefined;
  const titleFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "full" });
  const from = start.value;
  const to = end.value;

  return Array.from({ length: 42 }, (_, i) => {
    const date = new Date(year, month, 1 - lead + i);
    const inMonth = date.getMonth() === month;
    const isStart = same(date, from);
    const isEnd = same(date, to);
    const endpoint = isStart || isEnd;
    const between = !!from && !!to && date > from && date < to;
    const disabled = !!limit && date > limit;
    const classes = [
      endpoint
        ? "bg-primary text-primary-content font-semibold"
        : between
          ? "bg-primary/15"
          : "hover:bg-base-300 rounded-md",
      isStart && to && !isEnd ? "rounded-l-md" : "",
      isEnd && from && !isStart ? "rounded-r-md" : "",
      endpoint && (!to || isStart === isEnd) ? "rounded-md" : "",
      !inMonth && !endpoint ? "text-base-content/30" : "",
      same(date, today) && !endpoint ? "underline decoration-2 underline-offset-4" : "",
      disabled ? "opacity-30" : "",
    ];
    return { key: date.getTime(), date, classes, disabled, endpoint, title: titleFormat.format(date) };
  });
});

function pick(date: Date) {
  if (!start.value || end.value) {
    start.value = date;
    end.value = undefined;
  } else if (date < start.value) {
    start.value = date;
  } else {
    end.value = date;
  }
}
</script>
