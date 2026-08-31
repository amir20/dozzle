<template>
  <dialog ref="modal" class="modal" @close="onClose">
    <div class="modal-box max-w-lg p-8">
      <!-- Progress. Three steps, so the user can see the flow is short. -->
      <div class="mb-6 flex gap-1">
        <span
          v-for="i in 3"
          :key="i"
          class="h-[3px] flex-1 rounded-full transition-colors"
          :class="i === step ? 'bg-primary' : i < step ? 'bg-primary/45' : 'bg-base-content/15'"
        ></span>
      </div>

      <!-- ------------------------------------------------------------------
        Step 1 — what Cloud already does on its own.

        This screen exists to answer "I linked it, now what?" before the user
        asks. It never asks for anything; the only interactive element is the
        streaming toggle, and only when streaming is off.
      ------------------------------------------------------------------- -->
      <template v-if="step === 1">
        <template v-if="streamLogs">
          <span class="status-pill status-pill-success mb-3">
            <span class="size-1.5 rounded-full bg-current"></span>
            {{ $t("cloud.connected") }}
          </span>
          <h3 class="text-xl font-bold">{{ $t("cloud.welcome.watching-title") }}</h3>
          <p class="text-base-content/60 mt-2 text-sm">{{ $t("cloud.welcome.watching-body") }}</p>

          <ol class="mt-5 space-y-0">
            <li v-for="(beat, i) in timeline" :key="beat.when" class="flex gap-3">
              <div class="flex flex-col items-center">
                <span
                  class="mt-1.5 size-2.5 shrink-0 rounded-full"
                  :class="i === 0 ? 'bg-primary' : 'border-base-content/30 border-[1.5px]'"
                ></span>
                <span v-if="i < timeline.length - 1" class="bg-base-content/15 my-1 w-px flex-1"></span>
              </div>
              <div :class="i < timeline.length - 1 ? 'pb-3.5' : ''">
                <div class="text-base-content/55 font-mono text-[0.65rem] font-semibold tracking-wider uppercase">
                  {{ beat.when }}
                </div>
                <p class="mt-0.5 text-sm">{{ beat.what }}</p>
              </div>
            </li>
          </ol>
        </template>

        <template v-else>
          <span class="status-pill status-pill-warning mb-3">
            <span class="size-1.5 rounded-full bg-current"></span>
            {{ $t("cloud.welcome.scan-paused") }}
          </span>
          <h3 class="text-xl font-bold">{{ $t("cloud.welcome.paused-title") }}</h3>
          <p class="text-base-content/60 mt-2 text-sm">{{ $t("cloud.welcome.paused-body") }}</p>

          <label
            class="border-warning/45 bg-warning/10 mt-5 flex cursor-pointer items-center justify-between gap-4 rounded-lg border p-3"
          >
            <span class="text-sm">{{ $t("cloud.welcome.streaming-off") }}</span>
            <input
              type="checkbox"
              class="toggle toggle-primary toggle-sm shrink-0"
              :checked="streamLogs"
              :disabled="isSavingStreamLogs"
              @change="onStreamLogsChange(($event.target as HTMLInputElement).checked)"
            />
          </label>
        </template>

        <!--
          The privacy answer sits directly under the "Cloud watches your logs"
          headline on purpose. For a self-hosting audience that sentence raises
          the question, so this is where it gets answered.
        -->
        <p class="bg-success/10 text-base-content/70 mt-5 flex gap-2 rounded-lg p-3 text-xs leading-relaxed">
          <mdi:shield-check-outline class="text-success mt-0.5 shrink-0 text-sm" />
          <span>
            {{ $t("cloud.welcome.privacy") }}
            <a :href="`${cloudUrl}/privacy`" target="_blank" rel="noreferrer noopener" class="link link-primary">
              {{ $t("cloud.welcome.privacy-link") }}
              <mdi:open-in-new class="inline align-[-0.1em] text-[0.9em]" />
            </a>
          </span>
        </p>

        <button class="btn btn-primary btn-block mt-6" @click="step = 2">
          {{ $t("cloud.welcome.next-alerts") }}
        </button>
      </template>

      <!-- ------------------------------------------------------------------
        Step 2 — the three push categories.

        Named exactly as AlertForm names its alert types (log / metric / event)
        so the vocabulary carries over to the Notifications page.
      ------------------------------------------------------------------- -->
      <template v-else-if="step === 2">
        <h3 class="text-xl font-bold">{{ $t("cloud.welcome.push-title") }}</h3>
        <p class="text-base-content/60 mt-2 text-sm">{{ $t("cloud.welcome.push-body") }}</p>

        <div class="mt-5 space-y-2">
          <div
            v-for="category in categories"
            :key="category.id"
            class="rounded-lg border transition-colors"
            :class="category.enabled ? 'border-primary/45 bg-primary/[0.06]' : 'border-base-content/15'"
          >
            <div class="flex items-start gap-3 p-3">
              <component :is="category.icon" class="text-base-content/70 mt-0.5 shrink-0 text-lg" />
              <div class="flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="text-sm font-semibold">{{ category.label }}</span>
                  <span v-if="category.recommended" class="status-pill status-pill-primary">
                    {{ $t("cloud.welcome.recommended") }}
                  </span>
                </div>
                <p class="text-base-content/60 mt-0.5 text-xs">{{ category.description }}</p>
                <button
                  class="text-base-content/50 hover:text-primary mt-1.5 flex items-center gap-1 font-mono text-[0.68rem]"
                  @click="toggleExpanded(category.id)"
                >
                  <mdi:chevron-right
                    class="transition-transform"
                    :class="expanded.has(category.id) ? 'rotate-90' : ''"
                  />
                  {{ $t("cloud.welcome.rule-count", { on: enabledIn(category), total: category.rules.length }) }}
                </button>
              </div>
              <input
                type="checkbox"
                class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
                :aria-label="category.label"
                v-model="category.enabled"
              />
            </div>

            <div v-if="expanded.has(category.id)" class="space-y-1.5 pt-0 pr-3 pb-3 pl-10">
              <label
                v-for="rule in category.rules"
                :key="rule.key"
                class="flex cursor-pointer items-center gap-2 font-mono text-xs"
                :class="rule.enabled ? '' : 'text-base-content/45'"
              >
                <input v-model="rule.enabled" type="checkbox" class="checkbox checkbox-primary checkbox-xs" />
                <span>{{ rule.label }}</span>
              </label>
              <p
                v-if="category.caution"
                class="border-warning/60 text-base-content/65 mt-1 border-l-2 py-1 pl-2 text-xs leading-relaxed"
              >
                {{ category.caution }}
              </p>
            </div>
          </div>
        </div>

        <!-- Name the destination. Otherwise "nothing ever happens" just moves down a layer. -->
        <div
          class="border-base-content/20 text-base-content/65 mt-4 flex flex-wrap items-center gap-2 rounded-lg border border-dashed px-3 py-2 text-xs"
        >
          <span>{{ $t("cloud.welcome.alerts-go-to") }}</span>
          <span class="text-base-content font-mono">{{ destination }}</span>
          <a
            :href="`${cloudUrl}/settings`"
            target="_blank"
            rel="noreferrer noopener"
            class="link link-primary ml-auto whitespace-nowrap"
          >
            {{ $t("cloud.welcome.change") }}
            <mdi:open-in-new class="inline align-[-0.1em] text-[0.9em]" />
          </a>
        </div>

        <button class="btn btn-primary btn-block mt-5" :disabled="creating" @click="createAlerts">
          <span v-if="creating" class="loading loading-spinner loading-xs"></span>
          {{
            activeRules.length === 0
              ? $t("cloud.welcome.continue-without")
              : $t("cloud.welcome.turn-on", activeRules.length)
          }}
        </button>
        <button class="btn btn-ghost btn-block btn-sm mt-1" :disabled="creating" @click="skipAlerts">
          {{ $t("cloud.welcome.skip-alerts") }}
        </button>
      </template>

      <!-- ------------------------------------------------------------------
        Step 3 — where each kind of notification shows up.

        Every destination opens in a new tab so the user keeps the modal and
        their place in Dozzle.
      ------------------------------------------------------------------- -->
      <template v-else>
        <span class="status-pill status-pill-success mb-3">
          <span class="size-1.5 rounded-full bg-current"></span>
          {{ $t("cloud.welcome.ready") }}
        </span>
        <h3 class="text-xl font-bold">{{ $t("cloud.welcome.done-title") }}</h3>
        <p class="text-base-content/60 mt-2 text-sm">{{ $t("cloud.welcome.done-body") }}</p>

        <div class="mt-5 space-y-2">
          <div class="border-base-content/15 flex items-center gap-3 rounded-lg border p-3">
            <mdi:clipboard-text-search-outline class="text-base-content/70 shrink-0 text-lg" />
            <div class="flex-1">
              <div class="text-sm font-semibold">{{ $t("cloud.welcome.where-findings") }}</div>
              <p class="text-base-content/60 mt-0.5 text-xs">
                {{ $t("cloud.welcome.where-findings-detail", { day: firstScanDay }) }}
              </p>
            </div>
            <a :href="`${cloudUrl}/findings`" target="_blank" rel="noreferrer noopener" class="btn btn-xs">
              {{ $t("cloud.welcome.open-findings") }}
              <mdi:open-in-new class="text-[0.9em]" />
            </a>
          </div>

          <div class="border-base-content/15 flex items-center gap-3 rounded-lg border p-3">
            <mdi:bell-ring-outline class="text-base-content/70 shrink-0 text-lg" />
            <div class="flex-1">
              <div class="text-sm font-semibold">{{ $t("cloud.welcome.where-alerts") }}</div>
              <p v-if="createdCount > 0" class="text-base-content/60 mt-0.5 text-xs">
                {{
                  $t("cloud.welcome.where-alerts-detail", {
                    rules: $t("cloud.welcome.rule-plural", createdCount),
                    destination,
                  })
                }}
              </p>
              <p v-else class="text-warning/90 mt-0.5 text-xs">{{ $t("cloud.welcome.where-alerts-off") }}</p>
            </div>
            <button v-if="createdCount === 0" class="btn btn-xs" @click="step = 2">
              {{ $t("cloud.welcome.turn-on-short") }}
            </button>
          </div>

          <div class="border-base-content/15 flex items-center gap-3 rounded-lg border p-3">
            <mdi:tune-variant class="text-base-content/70 shrink-0 text-lg" />
            <div class="flex-1">
              <div class="text-sm font-semibold">{{ $t("cloud.welcome.where-tuning") }}</div>
              <p class="text-base-content/60 mt-0.5 text-xs">{{ $t("cloud.welcome.where-tuning-detail") }}</p>
            </div>
            <a :href="notificationsHref" target="_blank" rel="noreferrer noopener" class="btn btn-xs">
              {{ $t("notifications.title") }}
              <mdi:open-in-new class="text-[0.9em]" />
            </a>
          </div>
        </div>

        <button class="btn btn-primary btn-block mt-6" @click="close">{{ $t("cloud.welcome.done") }}</button>
      </template>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button></button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import MdiBellRingOutline from "~icons/mdi/bell-ring-outline";
import MdiChartLine from "~icons/mdi/chart-line";
import MdiTextBoxOutline from "~icons/mdi/text-box-outline";

const { t } = useI18n();
const router = useRouter();
const { showToast } = useToast();
const { cloudConfig, cloudStatus } = useCloudConfig();

const cloudUrl = __CLOUD_URL__;
const notificationsHref = withBase("/notifications");

const modal = ref<HTMLDialogElement>();
const step = ref(1);
const creating = ref(false);
const createdCount = ref(0);
const expanded = ref(new Set<string>());
let usageReported = false;

type AlertKind = "event" | "metric" | "log";

interface StarterRule {
  key: string;
  kind: AlertKind;
  label: string;
  // Kept in English so the rule stays recognizable if the user switches locale.
  ruleName: string;
  expression: string;
  enabled: boolean;
  cooldown: number;
  sampleWindow: number;
}

interface Category {
  id: string;
  label: string;
  description: string;
  caution?: string;
  icon: Component;
  recommended: boolean;
  enabled: boolean;
  rules: StarterRule[];
}

// Metric rules re-fire at most hourly per container (Cooldown is clamped to
// [0, 3600] server-side, so this is the longest quiet period available).
const METRIC_COOLDOWN = 3600;

// SampleWindow is a count of stat samples, and Docker's stats stream emits
// roughly once a second, so this reads as seconds. The server clamps it to
// [1, 300] and fires only when >=80% of a full buffer matched — 300 is the
// longest "sustained" window the engine can express.
const SUSTAINED_WINDOW = 300;

// Disk usage does not oscillate, so a long window would only delay the first
// notification. 15 is the server-side default.
const DISK_WINDOW = 15;

function buildCategories(): Category[] {
  return [
    {
      id: "lifecycle",
      label: t("notifications.alert-form.event-alert"),
      description: t("cloud.welcome.lifecycle-desc"),
      icon: MdiBellRingOutline,
      recommended: true,
      enabled: true,
      rules: [
        {
          key: "exited",
          kind: "event",
          label: t("cloud.welcome.signals.exited"),
          ruleName: "Container exited with an error",
          // Ignore clean/graceful shutdowns: 0 (success), 130 (SIGINT), 143 (SIGTERM), 137 (SIGKILL).
          // These commonly fire on `docker stop`, Ctrl+C, and Watchtower update cycles, which are
          // not errors. Still alerts on genuine error exits (1, 2, 125, ...) and crash-loops.
          expression: 'name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])',
          enabled: true,
          cooldown: 0,
          sampleWindow: 0,
        },
        {
          key: "unhealthy",
          kind: "event",
          label: t("cloud.welcome.signals.unhealthy"),
          ruleName: "Container became unhealthy",
          expression: 'name == "health_status" && attributes["healthStatus"] == "unhealthy"',
          enabled: true,
          cooldown: 0,
          sampleWindow: 0,
        },
        {
          key: "oom",
          kind: "event",
          label: t("cloud.welcome.signals.oom"),
          ruleName: "Container killed (OOM)",
          expression: 'name == "oom"',
          enabled: true,
          cooldown: 0,
          sampleWindow: 0,
        },
        {
          key: "restart",
          kind: "event",
          label: t("cloud.welcome.signals.restart"),
          ruleName: "Container restarted",
          expression: 'name == "restart"',
          enabled: false,
          cooldown: 0,
          sampleWindow: 0,
        },
      ],
    },
    {
      id: "metrics",
      label: t("notifications.alert-form.metric-alert"),
      description: t("cloud.welcome.metrics-desc"),
      caution: t("cloud.welcome.metrics-caution"),
      icon: MdiChartLine,
      recommended: false,
      enabled: true,
      rules: [
        {
          key: "cpu",
          kind: "metric",
          label: t("cloud.welcome.signals.cpu"),
          ruleName: "Sustained high CPU",
          // `cpu` is already normalized by core count server-side, so this is
          // overall load, not per-core.
          expression: "cpu >= 90",
          enabled: true,
          cooldown: METRIC_COOLDOWN,
          sampleWindow: SUSTAINED_WINDOW,
        },
        {
          key: "memory",
          kind: "metric",
          label: t("cloud.welcome.signals.memory"),
          ruleName: "Sustained high memory",
          // Percentage of the container's memory limit, or of host memory when
          // no limit is set.
          expression: "memory >= 90",
          enabled: true,
          cooldown: METRIC_COOLDOWN,
          sampleWindow: SUSTAINED_WINDOW,
        },
        {
          key: "disk",
          kind: "metric",
          label: t("cloud.welcome.signals.disk"),
          ruleName: "Volume running out of space",
          expression: "any(mounts, .usedPercent >= 85)",
          enabled: true,
          cooldown: METRIC_COOLDOWN,
          sampleWindow: DISK_WINDOW,
        },
        {
          // Off by default so it never double-fires with the percentage rule.
          // It is the better rule on large volumes, where 85% still leaves
          // hundreds of gigabytes free.
          key: "disk-free",
          kind: "metric",
          label: t("cloud.welcome.signals.disk-free"),
          ruleName: "Volume under 1 GB free",
          expression: "any(mounts, .availableBytes < 1073741824)",
          enabled: false,
          cooldown: METRIC_COOLDOWN,
          sampleWindow: DISK_WINDOW,
        },
      ],
    },
    {
      // Off by default. This is the category people assume they want and the
      // one most likely to make them mute everything, so opting in is deliberate.
      id: "logs",
      label: t("notifications.alert-form.log-alert"),
      description: t("cloud.welcome.logs-desc"),
      caution: t("cloud.welcome.logs-caution"),
      icon: MdiTextBoxOutline,
      recommended: false,
      enabled: false,
      rules: [
        {
          key: "fatal",
          kind: "log",
          label: t("cloud.welcome.signals.fatal"),
          ruleName: "Fatal log line",
          expression: 'level == "fatal"',
          enabled: true,
          cooldown: 0,
          sampleWindow: 0,
        },
        {
          key: "error",
          kind: "log",
          label: t("cloud.welcome.signals.error"),
          ruleName: "Error log line",
          expression: 'level == "error"',
          enabled: false,
          cooldown: 0,
          sampleWindow: 0,
        },
      ],
    },
  ];
}

const categories = ref<Category[]>(buildCategories());

const activeRules = computed(() =>
  categories.value.filter((c) => c.enabled).flatMap((c) => c.rules.filter((r) => r.enabled)),
);

const streamLogs = computed(() => cloudConfig.value?.streamLogs ?? true);
const destination = computed(() => cloudStatus.value?.user.email ?? t("cloud.welcome.your-cloud-account"));

const firstScanDay = computed(() => {
  const d = new Date();
  d.setDate(d.getDate() + 1);
  return d.toLocaleDateString(undefined, { weekday: "long" });
});

const timeline = computed(() => [
  { when: t("cloud.welcome.beat-now"), what: t("cloud.welcome.beat-now-detail") },
  {
    when: t("cloud.welcome.beat-tomorrow", { day: firstScanDay.value }),
    what: t("cloud.welcome.beat-tomorrow-detail"),
  },
  { when: t("cloud.welcome.beat-after"), what: t("cloud.welcome.beat-after-detail") },
]);

function enabledIn(category: Category) {
  return category.rules.filter((r) => r.enabled).length;
}

function toggleExpanded(id: string) {
  const next = new Set(expanded.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  expanded.value = next;
}

const isSavingStreamLogs = ref(false);

async function onStreamLogsChange(value: boolean) {
  if (!cloudConfig.value) return;
  isSavingStreamLogs.value = true;
  try {
    const res = await fetch(withBase("/api/cloud/config"), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ streamLogs: value }),
    });
    if (res.ok) cloudConfig.value.streamLogs = value;
  } catch {
    // Leave the toggle as-is; the user can retry from Settings.
  } finally {
    isSavingStreamLogs.value = false;
  }
}

// The survey this modal used to open with has been retired. The same endpoint
// now records which categories were turned on, which is the drop-off signal
// worth having.
async function reportUsage(skipped: boolean) {
  if (usageReported) return;
  usageReported = true;
  try {
    await fetch(withBase("/api/cloud/feedback"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source: "welcome_modal",
        selectedOptions: categories.value.filter((c) => c.enabled).map((c) => c.id),
        skipped,
      }),
    });
  } catch {
    // Reporting failure should never block the user.
  }
}

let abortController: AbortController | null = null;

async function createAlerts() {
  if (creating.value) return;
  const chosen = activeRules.value;
  if (chosen.length === 0) {
    createdCount.value = 0;
    reportUsage(true);
    step.value = 3;
    return;
  }

  creating.value = true;
  abortController?.abort();
  abortController = new AbortController();
  const signal = abortController.signal;

  try {
    const dispatchersRes = await fetch(withBase("/api/notifications/dispatchers"), { signal });
    if (!dispatchersRes.ok) throw new Error("dispatchers fetch failed");
    const dispatchers: Array<{ id: number; type: string }> = await dispatchersRes.json();
    const cloud = dispatchers.find((d) => d.type === "cloud");
    if (!cloud) throw new Error("cloud dispatcher missing");

    // Fire rule POSTs in parallel. Partial failure is not cleaned up — if one
    // rejects, the earlier ones are already saved and the user lands on the
    // fallback toast path. Acceptable for a welcome modal; the user can edit
    // or delete rules from /notifications.
    await Promise.all(
      chosen.map((rule) =>
        fetch(withBase("/api/notifications/rules"), {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          signal,
          body: JSON.stringify({
            name: rule.ruleName,
            enabled: true,
            dispatcherId: cloud.id,
            containerExpression: "true",
            logExpression: rule.kind === "log" ? rule.expression : "",
            eventExpression: rule.kind === "event" ? rule.expression : "",
            metricExpression: rule.kind === "metric" ? rule.expression : "",
            cooldown: rule.cooldown,
            sampleWindow: rule.sampleWindow,
          }),
        }).then((res) => {
          if (!res.ok) throw new Error("rule POST failed");
        }),
      ),
    );

    createdCount.value = chosen.length;
    reportUsage(false);
    step.value = 3;
  } catch (err) {
    if ((err as Error)?.name === "AbortError") return;
    close();
    showToast({ type: "warning", message: t("notifications.default-alert-failed") }, { expire: 6000 });
    router.push({ path: "/notifications", query: { action: "create-alert" } });
  } finally {
    creating.value = false;
  }
}

function skipAlerts() {
  createdCount.value = 0;
  reportUsage(true);
  step.value = 3;
}

function open() {
  step.value = 1;
  creating.value = false;
  createdCount.value = 0;
  expanded.value = new Set();
  categories.value = buildCategories();
  usageReported = false;
  modal.value?.showModal();
}

function close() {
  modal.value?.close();
}

function onClose() {
  reportUsage(true);
}

onBeforeUnmount(() => {
  abortController?.abort();
});

defineExpose({ open });
</script>
