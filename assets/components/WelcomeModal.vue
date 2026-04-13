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

const modal = ref<HTMLDialogElement>();
const step = ref<"step1" | "step2">("step1");
const intent = ref("");
const selectedOptions = ref(new Set<string>());
const submitting = ref(false);

const chipOptions = [
  { value: "alerts", label: t("cloud.welcome.chip-alerts") },
  { value: "daily_summary", label: t("cloud.welcome.chip-summary") },
  { value: "multiple_hosts", label: t("cloud.welcome.chip-hosts") },
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
  if (selectedOptions.value.has(value)) {
    selectedOptions.value.delete(value);
  } else {
    selectedOptions.value.add(value);
  }
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
  await postFeedback(true);
  submitting.value = false;
  if (onNotificationsPage.value) {
    createFirstAlert();
  } else {
    step.value = "step2";
  }
}

function createFirstAlert() {
  close();
  router.push({ path: "/notifications", query: { action: "create-alert" } });
}

function open() {
  modal.value?.showModal();
}

function close() {
  modal.value?.close();
}

function onClose() {
  if (step.value === "step1") {
    postFeedback(true);
  }
}

defineExpose({ open });
</script>
