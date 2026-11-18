<template>
  <!--
    What Cloud's daily review noticed about the containers on screen: the rules
    nobody wrote. Never listed beside alerts, which are the rules someone did.
  -->
  <div class="h-full overflow-y-auto p-4">
    <!-- When the review runs, on every page: an empty list reads as "nothing
         yet" only if the reader knows when "yet" ends. -->
    <div
      v-if="lastScanAt || nextScanAt"
      class="border-base-content/15 bg-base-200/40 mb-4 flex items-start gap-3 rounded-lg border p-4"
      data-schedule
    >
      <mdi:calendar-clock class="mt-0.5 size-4 shrink-0 opacity-60" />
      <div class="min-w-0 text-sm">
        <div v-if="nextScanAt">
          {{ $t("cloud-findings.next-review", { when: relative(nextScanAt), time: clock(nextScanAt) }) }}
        </div>
        <div class="text-base-content/60" :class="{ 'text-xs': nextScanAt }">
          {{
            lastScanAt
              ? $t("cloud-findings.last-review", { when: relative(lastScanAt) })
              : $t("cloud-findings.not-reviewed")
          }}
        </div>
      </div>
    </div>

    <div role="tablist" class="tabs tabs-box tabs-xs mb-4 w-fit">
      <button
        v-for="option in filters"
        :key="option"
        role="tab"
        class="tab"
        :class="{ 'tab-active': filter === option }"
        @click="filter = option"
      >
        {{ $t(`cloud-findings.filter-${option}`) }}
      </button>
    </div>

    <p v-if="loading && !findings.length" class="text-base-content/60 text-sm">{{ $t("cloud-findings.loading") }}</p>
    <p v-else-if="failed" class="text-base-content/60 text-sm">{{ $t("cloud-findings.failed") }}</p>
    <p v-else-if="!findings.length" class="text-base-content/60 text-sm">
      {{ emptyLine }}
    </p>

    <div v-else class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <div v-for="finding in findings" :key="finding.id">
        <button
          type="button"
          class="hover:bg-base-300 flex w-full items-start gap-3 p-3 text-left transition-colors"
          :aria-expanded="expanded === finding.id"
          @click="expanded = expanded === finding.id ? undefined : finding.id"
        >
          <!-- Severity rides the icon; the row behind it stays neutral. -->
          <span
            class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full"
            :class="tint[finding.severity]"
          >
            <mdi:lightbulb-on-outline class="size-4" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block text-sm font-semibold wrap-anywhere" :class="{ 'text-base-content/60': finding.locked }">
              {{ finding.locked ? $t("cloud-findings.locked") : finding.headline }}
            </span>
            <span class="text-base-content/60 mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <span v-if="!single" class="font-mono">{{ finding.containers.map((c) => c.name).join(", ") }}</span>
              <RelativeTime :date="new Date(finding.lastSeenAt / 1e6)" />
              <span v-if="finding.status === 'resolved'" class="status-pill status-pill-success">
                {{ $t("cloud-findings.resolved") }}
              </span>
            </span>
          </span>
        </button>

        <div v-if="expanded === finding.id" class="space-y-3 px-3 pb-3 pl-13">
          <p v-if="finding.why" class="text-base-content/60 text-sm wrap-anywhere">{{ finding.why }}</p>
          <p v-if="finding.fix" class="text-sm wrap-anywhere">{{ finding.fix }}</p>
          <div class="flex flex-wrap items-center gap-2">
            <!-- A finding on several containers on screen opens the first; the
                 others are one tap away by name. -->
            <button
              v-for="(container, i) in finding.containers"
              :key="container.id"
              type="button"
              class="btn btn-sm"
              :class="{ 'btn-primary': i === 0 }"
              @click="showLines(finding, container)"
            >
              <material-symbols:eye-tracking v-if="i === 0" class="size-4" />
              <template v-if="i === 0">{{ $t("cloud-findings.show-lines") }}</template>
              <span v-else class="font-mono">{{ container.name }}</span>
            </button>
            <!-- The server says whether the ignore route would let this reader in. -->
            <template v-if="canIgnore">
              <button v-if="filter === 'ignored'" type="button" class="btn btn-sm" @click="unignore(finding)">
                {{ $t("cloud-findings.unignore") }}
              </button>
              <button v-else type="button" class="btn btn-sm" @click="ignore(finding)">
                {{ $t("cloud-findings.ignore") }}
              </button>
            </template>
            <a
              v-if="finding.url"
              :href="finding.url"
              target="_blank"
              rel="noreferrer noopener"
              class="hover:bg-base-300 ml-auto flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm transition-colors"
            >
              {{ $t("cloud-findings.investigate") }}
              <mdi:open-in-new class="size-3.5 opacity-40" />
            </a>
          </div>
        </div>
      </div>
    </div>

    <!-- The rest is in Cloud, which the footer links to. -->
    <p v-if="truncated && findings.length" class="text-base-content/40 mt-3 text-xs">
      {{ $t("cloud-findings.truncated") }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import {
  findingMoment,
  useFindingsScope,
  type CloudFinding,
  type FindingContainer,
  type FindingsFilter,
} from "@/composable/cloud/cloudFindings";

const filters: FindingsFilter[] = ["new", "all", "ignored"];
const filter = ref<FindingsFilter>("new");
const expanded = ref<string>();

// The containers asked about: the ones on screen, or every one this instance knows.
const containers = useFindingsScope();
const single = computed(() => containers.value.length === 1);
const { findings, truncated, canIgnore, lastScanAt, nextScanAt, loading, failed, ignore, unignore, markSeen } =
  useCloudFindings(filter);
const { t, locale } = useI18n();

// Nanoseconds, like every other time on a finding. The shared tick keeps
// "in 3 hours" honest on a panel left open.
function relative(ns: number) {
  void relativeTimeTick.value;
  return toRelativeTime(new Date(ns / 1e6), locale.value || undefined);
}
// The reader's own clock, with the day when it is not today.
function clock(ns: number) {
  const date = new Date(ns / 1e6);
  const today = date.toDateString() === new Date().toDateString();
  return new Intl.DateTimeFormat(locale.value || undefined, {
    weekday: today ? undefined : "short",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: hourStyle.value === "12" ? "h12" : hourStyle.value === "24" ? "h23" : undefined,
  }).format(date);
}

// Once a review has run, an empty list is its answer, and says so by name.
const emptyLine = computed(() => {
  if (!lastScanAt.value || filter.value === "ignored") return t(`cloud-findings.empty-${filter.value}`);
  const name = single.value ? containers.value[0].name : t("cloud-rail.scope-n", { n: containers.value.length });
  return t("cloud-findings.empty-reviewed", { name });
});
const { jumpTo } = useLogJump();
const { closeRail } = useCloudRail();

const tint: Record<string, string> = {
  high: "bg-error/10 text-error",
  medium: "bg-warning/10 text-warning",
  low: "bg-info/10 text-info",
};

// Open on New is what the tab's dot promised, so that is what clears it.
watchEffect(() => {
  if (filter.value === "new" && findings.value.length) markSeen();
});

// The lines are the reader's own local logs, so a locked finding gets them too:
// a time is not the diagnosis its plan withholds.
function showLines(finding: CloudFinding, container: FindingContainer) {
  closeRail();
  jumpTo(findingMoment(finding, container));
}
</script>
