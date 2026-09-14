<template>
  <StepModal ref="modal" :title="$t('setup.title')" :steps="railSteps" @close="onClose" @cancel="onCancel">
    <template v-if="status && currentId">
      <SetupLoginStep v-if="currentId === 'login'" ref="step" :status="status" :next-step="steps[index + 1]" />
      <SetupActionsStep v-else-if="currentId === 'actions'" ref="step" :status="status" />
      <SetupCloudStep v-else-if="currentId === 'cloud'" ref="step" :next-step="steps[index + 1]" />
      <SetupRestartStep v-else ref="step" :status="status" @seen="setupSeen = true" />
    </template>
    <div v-else-if="loading" class="flex h-full items-center justify-center">
      <span class="loading loading-spinner loading-sm"></span>
    </div>
    <InlineNotice v-else type="error">{{ $t("setup.error.load") }}</InlineNotice>

    <template #footer-start>
      <button v-if="!isLast" type="button" class="btn btn-ghost btn-sm" :disabled="busy" @click="close">
        {{ $t("setup.finish-later") }}
      </button>
    </template>
    <template #footer-end>
      <button v-if="index > 0" type="button" class="btn btn-sm" :disabled="busy" @click="back">
        {{ $t("setup.back") }}
      </button>
      <button v-if="handle?.skipLabel" type="button" class="btn btn-sm" :disabled="busy" @click="advance(true)">
        {{ handle.skipLabel }}
      </button>
      <button
        v-if="handle"
        type="button"
        class="btn btn-sm"
        :class="{ 'btn-primary': !handle.nextPlain }"
        :disabled="handle.nextDisabled"
        @click="onNext"
      >
        <span v-if="handle.busy" class="loading loading-spinner loading-xs"></span>
        {{ handle.nextLabel }}
      </button>
      <button v-else-if="!status && !loading" type="button" class="btn btn-sm" @click="close">
        {{ $t("setup.close") }}
      </button>
    </template>
  </StepModal>
</template>

<script lang="ts" setup>
import type { SetupStepHandle, SetupStepId, SetupStepState } from "@/composable/setup/setup";
import type StepModal from "@/components/ui/StepModal.vue";

const { t } = useI18n();
const { status, loading, wizardOpen, fetchStatus } = useSetup();
const { linked, canLink } = useCloudSurface();
const { initialLoad } = useCloudConfig();
const { requestCloudWelcome } = useCloudWelcome();
const setupSeen = useProfileStorage("setupSeen", false);

const modal = useTemplateRef<InstanceType<typeof StepModal>>("modal");
const handle = useTemplateRef<SetupStepHandle>("step");

const steps = ref<SetupStepId[]>([]);
const index = ref(0);
const skipped = ref(new Set<SetupStepId>());

const currentId = computed<SetupStepId | undefined>(() => steps.value[index.value]);
const isLast = computed(() => index.value >= steps.value.length - 1 && steps.value.length > 0);
const busy = computed(() => !!handle.value?.busy);

function stateOf(id: SetupStepId, i: number): SetupStepState {
  if (i === index.value) return "current";
  if (skipped.value.has(id)) return "skipped";
  if (id === "login" && status.value) {
    if (setupLoginConfigured(status.value)) return "done";
    return i < index.value ? "skipped" : "todo";
  }
  return i < index.value ? "done" : "todo";
}

const notes: Partial<Record<SetupStepId, string>> = {
  login: "setup.steps.login-note",
  cloud: "setup.steps.cloud-note",
};

const railSteps = computed(() =>
  steps.value.map((id, i) => ({
    id,
    label: t(`setup.steps.${id}`),
    note: notes[id] ? t(notes[id]) : undefined,
    state: stateOf(id, i),
  })),
);

// A restart or the cloud round trip left a marker naming the step to come back to.
// When the page came back from Cloud, the wizard owns that return: the hash is
// dropped here, synchronously during setup, and the cloud welcome is handed over
// once the wizard closes instead of opening on top of it.
const resumeAtLoad = readSetupResume();
if (resumeAtLoad && window.location.hash === "#cloudLinked") {
  markCloudWelcomePending();
  history.replaceState(history.state, "", window.location.pathname + window.location.search);
}

// Opened by hand it shows a spinner right away. Opened by itself it waits for the
// status first, so a server without the setup API never flashes an error at anyone.
async function open(startAt: SetupStepId | undefined, auto: boolean) {
  skipped.value = new Set();
  steps.value = [];
  index.value = 0;
  if (!auto) modal.value?.open();

  await Promise.all([fetchStatus(), initialLoad]);
  const s = status.value;
  if (!s) {
    if (auto) wizardOpen.value = false;
    return;
  }
  modal.value?.open();

  // Frozen for the session, so linking Cloud or saving a toggle does not shuffle
  // the rail under the user.
  steps.value = setupSteps(s, { linked: linked.value, canLink: canLink.value });
  if (startAt) {
    const at = steps.value.indexOf(startAt);
    index.value = at >= 0 ? at : steps.value.length - 1;
  } else if (steps.value[0] === "login" && s.authProvider !== "none") {
    index.value = 1;
  }
}

function close() {
  modal.value?.close();
}

// Escape mid-restart would clear the resume marker the restart depends on.
function onCancel(e: Event) {
  if (busy.value) e.preventDefault();
}

function onClose() {
  setupSeen.value = true;
  clearSetupResume();
  wizardOpen.value = false;
  // Linked Cloud along the way: the welcome picks up at its starter alerts, since
  // the wizard's Cloud step already said what Cloud does.
  if (cloudWelcomePending()) {
    clearCloudWelcomePending();
    requestCloudWelcome(2);
  }
}

function back() {
  if (index.value > 0) index.value--;
}

function advance(skip = false) {
  const id = currentId.value;
  if (skip && id) skipped.value.add(id);
  if (isLast.value) {
    close();
    return;
  }
  index.value++;
}

async function onNext() {
  if (!handle.value) return;
  const result = await handle.value.next();
  if (result === "advance") advance();
  else if (result === "skip") advance(true);
  else if (result === "finish") close();
}

// The last step lists what is pending, so it reads a fresh status on arrival.
watch(currentId, (id) => {
  if (id === "restart") fetchStatus();
});

let autoOpening = false;

watch(wizardOpen, (value) => {
  if (value && !modal.value?.isOpen()) {
    open(readSetupResume(), autoOpening);
    autoOpening = false;
  } else if (!value && modal.value?.isOpen()) close();
});

onMounted(() => {
  const shouldOpen = setupShouldAutoOpen({
    mode: config.mode,
    authProvider: config.authProvider,
    setupSeen: setupSeen.value,
    profile: config.profile,
    resume: resumeAtLoad,
    hideMenu: new URLSearchParams(window.location.search).has("hideMenu"),
  });
  if (shouldOpen) {
    autoOpening = true;
    wizardOpen.value = true;
  }
});
</script>
