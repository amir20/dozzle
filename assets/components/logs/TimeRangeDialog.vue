<template>
  <dialog ref="dialog" class="modal max-md:modal-bottom" @close="request = undefined">
    <form class="modal-box flex w-full max-w-2xl flex-col p-0" @submit.prevent="apply">
      <div class="flex flex-col gap-1 px-6 pt-6">
        <h2 class="text-2xl font-bold">{{ $t("time-range.custom") }}</h2>
        <p class="text-base-content/60 text-sm">{{ $t("time-range.custom-hint", { zone: timeZone }) }}</p>
      </div>

      <div class="flex gap-6 p-6 max-md:flex-col" v-if="request">
        <CalendarRange
          v-model:start="startDay"
          v-model:end="endDay"
          :max="new Date()"
          :previous-label="$t('time-range.previous-month')"
          :next-label="$t('time-range.next-month')"
          class="md:w-64 md:shrink-0"
        />

        <div class="flex flex-1 flex-col gap-4">
          <label class="flex flex-col gap-1.5">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
              {{ $t("time-range.from") }}
            </span>
            <span class="flex items-center gap-2">
              <span class="text-base-content/60 w-16 shrink-0 font-mono text-sm">{{ dayLabel(startDay) }}</span>
              <input
                v-model="fromTime"
                autofocus
                class="input input-sm w-full font-mono"
                :class="{ 'input-error': fromTime && !fromParsed }"
                placeholder="00:00:00"
                autocomplete="off"
                spellcheck="false"
              />
            </span>
          </label>

          <label class="flex flex-col gap-1.5">
            <span class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
              {{ $t("time-range.to") }}
            </span>
            <span class="flex items-center gap-2" :class="{ 'opacity-40': follow }">
              <span class="text-base-content/60 w-16 shrink-0 font-mono text-sm">{{
                dayLabel(endDay ?? startDay)
              }}</span>
              <input
                v-model="toTime"
                class="input input-sm w-full font-mono"
                :class="{ 'input-error': !follow && toTime && !toParsed }"
                placeholder="23:59:59"
                autocomplete="off"
                spellcheck="false"
                :disabled="follow"
              />
            </span>
          </label>

          <label class="flex items-center justify-between gap-3 text-sm">
            <span class="flex flex-col">
              <span>{{ $t("time-range.keep-following") }}</span>
              <span class="text-base-content/50 text-xs">{{ $t("time-range.keep-following-hint") }}</span>
            </span>
            <input v-model="follow" type="checkbox" class="toggle toggle-sm toggle-primary" />
          </label>

          <p v-if="invalid && (fromTime || toTime)" class="text-error text-xs">{{ $t("time-range.invalid") }}</p>
        </div>
      </div>

      <div v-if="request && !unsupported" class="flex flex-col gap-1 px-6 pb-6">
        <div class="text-base-content/50 flex justify-between text-xs">
          <span>{{ $t("time-range.volume") }}</span>
          <span class="flex items-center gap-1.5">
            <span class="bg-error size-2 rounded-xs"></span>
            {{ $t("time-range.errors") }}
          </span>
        </div>
        <LogVolumeChart :data="volume" :selection="draft" selectable height-class="h-12" @select="chartSelect" />
      </div>

      <div class="border-base-content/10 flex justify-end gap-2 border-t px-6 py-4">
        <button type="button" class="btn btn-sm" @click="dialog?.close()">{{ $t("button.cancel") }}</button>
        <button type="submit" class="btn btn-primary btn-sm" :disabled="invalid">{{ $t("time-range.show") }}</button>
      </div>
    </form>
    <form method="dialog" class="modal-backdrop">
      <button>{{ $t("button.cancel") }}</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import { formatTime, parseTime, timeRangeRoute, type TimeRange } from "@/composable/logs/timeRange";

const request = useCustomRangeRequest();
const router = useRouter();
const dialog = useTemplateRef<HTMLDialogElement>("dialog");

const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;

const midnight = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate());
const dayLabel = (d?: Date) => (d ? d.toLocaleDateString(undefined, { month: "short", day: "numeric" }) : "—");

const startDay = ref<Date>();
const endDay = ref<Date>();
const fromTime = ref("");
const toTime = ref("");
const follow = ref(false);

// Opens on whatever the view shows, so a small change is a small edit.
function fill(from: Date, until: Date, following: boolean) {
  startDay.value = midnight(from);
  endDay.value = midnight(until);
  fromTime.value = formatTime(from);
  toTime.value = formatTime(until);
  follow.value = following;
}

watch(request, (r) => {
  if (!r) return;
  const now = new Date();
  if (r.anchor) {
    fill(new Date(r.anchor.getTime() - 5 * 60_000), new Date(Math.min(+now, r.anchor.getTime() + 5 * 60_000)), false);
  } else if (r.range.kind === "range") {
    fill(r.range.from, r.range.until, false);
  } else if (r.range.kind === "since") {
    fill(r.range.since, now, true);
  } else {
    fill(new Date(+now - 60 * 60_000), now, false);
  }
  nextTick(() => dialog.value?.showModal());
});

const fromParsed = computed(() => parseTime(fromTime.value));
const toParsed = computed(() => parseTime(toTime.value));

function at(day: Date | undefined, time: { h: number; m: number; s: number } | undefined) {
  if (!day || !time) return undefined;
  return new Date(day.getFullYear(), day.getMonth(), day.getDate(), time.h, time.m, time.s);
}

const draft = computed(() => {
  const from = at(startDay.value, fromParsed.value);
  const to = follow.value ? new Date() : at(endDay.value ?? startDay.value, toParsed.value);
  return from && to && to > from ? { from, to } : undefined;
});
const invalid = computed(() => draft.value === undefined);

// The chart covers the pick plus as much again on each side, up to now.
const volumeWindow = computed(() => {
  const d = draft.value;
  if (!request.value || !d) return undefined;
  const span = d.to.getTime() - d.from.getTime();
  return { from: new Date(d.from.getTime() - span), to: new Date(Math.min(Date.now(), d.to.getTime() + span)) };
});
const { data: volume, unsupported } = useLogHistogram(
  toRef(() => request.value?.container ?? ({ host: "", id: "" } as never)),
  volumeWindow,
  60,
);

function chartSelect(from: Date, to: Date) {
  const following = to.getTime() >= Date.now();
  fill(from, following ? new Date() : to, following);
}

function apply() {
  const r = request.value;
  const d = draft.value;
  if (!r || !d) return;
  const next: TimeRange = follow.value
    ? { kind: "since", since: d.from }
    : { kind: "range", from: d.from, until: d.to };
  const to = timeRangeRoute(r.container.id, next);
  if (next.kind === r.range.kind && !r.anchor) router.replace(to);
  else router.push(to);
  dialog.value?.close();
}
</script>
