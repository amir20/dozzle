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

      <!-- Step 2: Onboarding Checklist -->
      <template v-else-if="step === 'step2'">
        <h3 class="text-xl font-bold">{{ $t("cloud.welcome.step2-title") }}</h3>
        <p class="text-base-content/60 mt-1 text-sm">{{ $t("cloud.welcome.step2-subtitle") }}</p>

        <div class="mt-6 space-y-4">
          <div v-for="(item, index) in checklistItems" :key="index" class="flex gap-3">
            <div
              class="flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold"
              :class="index === 0 ? 'bg-primary text-primary-content' : 'bg-base-200 text-base-content/60'"
            >
              {{ index + 1 }}
            </div>
            <div>
              <component
                :is="item.href ? 'a' : 'p'"
                class="text-sm font-semibold"
                :class="item.href ? 'link link-hover' : ''"
                :href="item.href"
                :target="item.href ? '_blank' : undefined"
                :rel="item.href ? 'noreferrer noopener' : undefined"
              >
                {{ item.title }}
              </component>
              <p class="text-base-content/60 text-xs">{{ item.description }}</p>
            </div>
          </div>
        </div>

        <button class="btn btn-primary btn-block mt-6" @click="createFirstAlert">
          {{ $t("cloud.welcome.create-alert") }}
        </button>
        <button class="btn btn-ghost btn-block btn-sm mt-1" @click="close">
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
const cloudUrl = __CLOUD_URL__;
const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const { showToast } = useToast();

const modal = ref<HTMLDialogElement>();
const step = ref<"step1" | "step2">("step1");
const intent = ref("");
const selectedOptions = ref(new Set<string>());
const submitting = ref(false);
let feedbackSent = false;

const chipOptions = [
  { value: "error_alerts", label: t("cloud.welcome.chip-alerts") },
  { value: "ai_assistant", label: t("cloud.welcome.chip-assistant") },
  { value: "multiple_hosts", label: t("cloud.welcome.chip-hosts") },
  { value: "remote_access", label: t("cloud.welcome.chip-remote-access") },
  { value: "log_digests", label: t("cloud.welcome.chip-digests") },
  { value: "something_else", label: t("cloud.welcome.chip-other") },
];

const checklistItems = computed(() => [
  {
    title: t("cloud.welcome.checklist-alert-title"),
    description: t("cloud.welcome.checklist-alert-desc"),
  },
  {
    title: t("cloud.welcome.checklist-notify-title"),
    description: t("cloud.welcome.checklist-notify-desc"),
    href: `${cloudUrl}/channels`,
  },
  {
    title: t("cloud.welcome.checklist-agent-title"),
    description: t("cloud.welcome.checklist-agent-desc"),
    href: `${cloudUrl}/assistant`,
  },
]);

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
    createFirstAlert();
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
    createFirstAlert();
  } else {
    step.value = "step2";
  }
}

async function createFirstAlert() {
  close();
  try {
    const dispatchersRes = await fetch(withBase("/api/notifications/dispatchers"));
    if (!dispatchersRes.ok) throw new Error("dispatchers fetch failed");
    const dispatchers: Array<{ id: number; type: string }> = await dispatchersRes.json();
    const cloud = dispatchers.find((d) => d.type === "cloud");
    if (!cloud) throw new Error("cloud dispatcher missing");

    const ruleRes = await fetch(withBase("/api/notifications/rules"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: t("cloud.welcome.default-alert-name"),
        enabled: true,
        dispatcherId: cloud.id,
        logExpression: "",
        containerExpression: "true",
        eventExpression: 'name == "die" && attributes["exitCode"] != "0"',
        metricExpression: "",
        cooldown: 0,
        sampleWindow: 0,
      }),
    });
    if (!ruleRes.ok) throw new Error("rule POST failed");
    const rule: { id: number } = await ruleRes.json();

    router.push({ path: "/notifications", query: { highlight: String(rule.id) } });
  } catch {
    showToast(
      {
        type: "warning",
        message: t("notifications.default-alert-failed"),
      },
      { expire: 6000 },
    );
    router.push({ path: "/notifications", query: { action: "create-alert" } });
  }
}

function open() {
  step.value = "step1";
  intent.value = "";
  selectedOptions.value = new Set();
  feedbackSent = false;
  modal.value?.showModal();
}

function close() {
  modal.value?.close();
}

function onClose() {
  if (step.value === "step1" && !feedbackSent) {
    feedbackSent = true;
    postFeedback(true);
  }
}

defineExpose({ open });
</script>
