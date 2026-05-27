<template>
  <dialog ref="modal" class="modal" @close="onClose">
    <div class="modal-box max-w-md p-8">
      <!-- Step 1: Feedback -->
      <template v-if="step === 'step1'">
        <div class="flex flex-col items-center gap-2 text-center">
          <mdi:check-circle class="text-success text-4xl" />
          <h3 class="text-xl font-bold">{{ $t("cloud.welcome.title") }}</h3>
          <p class="text-base-content/60 text-sm">{{ $t("cloud.welcome.subtitle") }}</p>
        </div>

        <div class="divider"></div>

        <p class="mb-3 text-sm font-medium">{{ $t("cloud.welcome.question") }}</p>

        <textarea
          v-model="intent"
          class="textarea textarea-bordered w-full text-sm"
          rows="3"
          :placeholder="$t('cloud.welcome.placeholder')"
        ></textarea>

        <p class="text-base-content/60 mt-3 mb-2 text-xs">{{ $t("cloud.welcome.or-pick") }}</p>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="option in chipOptions"
            :key="option.value"
            class="btn btn-sm"
            :class="selectedOptions.has(option.value) ? 'btn-primary' : 'btn-outline'"
            @click="toggleOption(option.value)"
          >
            {{ option.label }}
          </button>
        </div>

        <button class="btn btn-primary btn-block mt-6" :disabled="submitting" @click="submitFeedback">
          <span v-if="submitting" class="loading loading-spinner loading-xs"></span>
          {{ $t("cloud.welcome.get-started") }}
        </button>
        <button class="btn btn-ghost btn-block btn-sm mt-1" :disabled="submitting" @click="skipFeedback">
          {{ $t("cloud.welcome.skip") }}
        </button>
      </template>

      <!-- Step 2: Triage signal checklist -->
      <template v-else-if="step === 'step2'">
        <h3 class="text-xl font-bold">{{ $t("cloud.welcome.step2-title") }}</h3>
        <p class="text-base-content/60 mt-2 text-sm">{{ $t("cloud.welcome.step2-body") }}</p>

        <div class="mt-5 space-y-3">
          <label
            v-for="signal in signals"
            :key="signal.key"
            class="border-base-300 hover:border-primary/40 flex cursor-pointer gap-3 rounded-lg border p-3"
          >
            <input
              v-model="selectedSignals"
              type="checkbox"
              :value="signal.key"
              class="checkbox checkbox-primary checkbox-sm mt-0.5"
            />
            <div class="flex-1">
              <p class="text-sm font-semibold">{{ signal.label }}</p>
              <p class="text-base-content/60 text-xs">{{ signal.description }}</p>
            </div>
          </label>
        </div>

        <p class="text-base-content/60 mt-4 text-xs">{{ $t("cloud.welcome.footer") }}</p>

        <button
          class="btn btn-primary btn-block mt-5"
          :disabled="creating || selectedSignals.length === 0"
          @click="createDefaultAlerts"
        >
          <span v-if="creating" class="loading loading-spinner loading-xs"></span>
          {{ $t("cloud.welcome.create-alerts") }}
        </button>
        <button class="btn btn-ghost btn-block btn-sm mt-1" :disabled="creating" @click="close">
          {{ $t("cloud.welcome.later") }}
        </button>
      </template>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button></button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const { showToast } = useToast();

const modal = ref<HTMLDialogElement>();
const step = ref<"step1" | "step2">("step1");
const intent = ref("");
const selectedOptions = ref(new Set<string>());
const submitting = ref(false);
const creating = ref(false);
let feedbackSent = false;

const chipOptions = [
  { value: "error_alerts", label: t("cloud.welcome.chip-alerts") },
  { value: "ai_assistant", label: t("cloud.welcome.chip-assistant") },
  { value: "search_logs", label: t("cloud.welcome.chip-search-logs") },
  { value: "remote_access", label: t("cloud.welcome.chip-remote-access") },
  { value: "log_digests", label: t("cloud.welcome.chip-digests") },
  { value: "something_else", label: t("cloud.welcome.chip-other") },
];

type SignalKey = "exited" | "unhealthy" | "oom" | "restart" | "disk";
type SignalKind = "event" | "metric";

interface SignalDef {
  key: SignalKey;
  kind: SignalKind;
  label: string;
  description: string;
  // ruleName is intentionally English/stable so the rule stays recognizable
  // if the user later switches locale.
  ruleName: string;
  expression: string;
  defaultOn: boolean;
}

const signals = computed<SignalDef[]>(() => [
  {
    key: "exited",
    kind: "event",
    label: t("cloud.welcome.signals.exited"),
    description: t("cloud.welcome.signals.exited-desc"),
    ruleName: "Container exited with an error",
    // Ignore clean/graceful shutdowns: 0 (success), 143 (SIGTERM), 137 (SIGKILL).
    // These commonly fire on `docker stop` and Watchtower update cycles, which are
    // not errors. Still alerts on genuine error exits (1, 2, 125, ...) and crash-loops.
    expression: 'name == "die" && !(attributes["exitCode"] in ["0", "143", "137"])',
    defaultOn: true,
  },
  {
    key: "unhealthy",
    kind: "event",
    label: t("cloud.welcome.signals.unhealthy"),
    description: t("cloud.welcome.signals.unhealthy-desc"),
    ruleName: "Container became unhealthy",
    expression: 'name == "health_status" && attributes["healthStatus"] == "unhealthy"',
    defaultOn: true,
  },
  {
    key: "oom",
    kind: "event",
    label: t("cloud.welcome.signals.oom"),
    description: t("cloud.welcome.signals.oom-desc"),
    ruleName: "Container killed (OOM)",
    expression: 'name == "oom"',
    defaultOn: true,
  },
  {
    key: "restart",
    kind: "event",
    label: t("cloud.welcome.signals.restart"),
    description: t("cloud.welcome.signals.restart-desc"),
    ruleName: "Container restarted",
    expression: 'name == "restart"',
    defaultOn: false,
  },
  {
    key: "disk",
    kind: "metric",
    label: t("cloud.welcome.signals.disk"),
    description: t("cloud.welcome.signals.disk-desc"),
    ruleName: "Volume running out of space",
    expression: "any(mounts, .usedPercent >= 85)",
    defaultOn: true,
  },
]);

const selectedSignals = ref<SignalKey[]>([]);

function toggleOption(value: string) {
  const next = new Set(selectedOptions.value);
  if (next.has(value)) {
    next.delete(value);
  } else {
    next.add(value);
  }
  selectedOptions.value = next;
}

async function postFeedback(skipped: boolean) {
  try {
    await fetch(withBase("/api/cloud/feedback"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source: "welcome_modal",
        intent: skipped ? undefined : intent.value || undefined,
        selectedOptions: skipped ? undefined : Array.from(selectedOptions.value),
        skipped,
      }),
    });
  } catch {
    // Feedback failure should not block the user
  }
}

const onNotificationsPage = computed(() => route.path === "/notifications");

async function submitFeedback() {
  submitting.value = true;
  feedbackSent = true;
  await postFeedback(false);
  submitting.value = false;
  if (onNotificationsPage.value) {
    await createDefaultAlerts();
  } else {
    step.value = "step2";
  }
}

async function skipFeedback() {
  submitting.value = true;
  feedbackSent = true;
  await postFeedback(true);
  submitting.value = false;
  if (onNotificationsPage.value) {
    // User explicitly skipped — don't silently create defaults on their behalf.
    // They're already on the notifications page; just dismiss.
    close();
  } else {
    step.value = "step2";
  }
}

let abortController: AbortController | null = null;

async function createDefaultAlerts() {
  if (creating.value) return;
  creating.value = true;
  abortController?.abort();
  abortController = new AbortController();
  const signal = abortController.signal;
  const chosen = signals.value.filter((s) => selectedSignals.value.includes(s.key));
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
      chosen.map((s) =>
        fetch(withBase("/api/notifications/rules"), {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          signal,
          body: JSON.stringify({
            name: s.ruleName,
            enabled: true,
            dispatcherId: cloud.id,
            logExpression: "",
            containerExpression: "true",
            eventExpression: s.kind === "event" ? s.expression : "",
            metricExpression: s.kind === "metric" ? s.expression : "",
            // Metric alerts: don't re-fire more than once an hour per container,
            // and require the threshold to hold for the default sample window.
            cooldown: s.kind === "metric" ? 3600 : 0,
            sampleWindow: s.kind === "metric" ? 60 : 0,
          }),
        }).then((res) => {
          if (!res.ok) throw new Error("rule POST failed");
        }),
      ),
    );

    close();
    router.push({ path: "/notifications" });
  } catch (err) {
    if ((err as Error)?.name === "AbortError") return;
    close();
    showToast(
      {
        type: "warning",
        message: t("notifications.default-alert-failed"),
      },
      { expire: 6000 },
    );
    router.push({ path: "/notifications", query: { action: "create-alert" } });
  } finally {
    creating.value = false;
  }
}

function open() {
  step.value = "step1";
  intent.value = "";
  selectedOptions.value = new Set();
  selectedSignals.value = signals.value.filter((s) => s.defaultOn).map((s) => s.key);
  feedbackSent = false;
  modal.value?.showModal();
}

function close() {
  modal.value?.close();
}

onBeforeUnmount(() => {
  abortController?.abort();
});

function onClose() {
  if (step.value === "step1" && !feedbackSent) {
    feedbackSent = true;
    postFeedback(true);
  }
}

defineExpose({ open });
</script>
