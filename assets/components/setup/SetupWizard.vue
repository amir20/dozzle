<template>
  <dialog ref="modal" class="modal" @close="onClose" @cancel="onCancel">
    <div class="modal-box flex max-h-[90vh] w-full max-w-215 overflow-hidden p-0 max-md:flex-col md:h-160">
      <!-- Rail: where you are and how much is left. On a phone it collapses to a bar. -->
      <aside class="bg-base-200/60 border-base-content/10 w-55 shrink-0 border-r p-5 max-md:hidden">
        <div class="text-base font-semibold">{{ $t("setup.title") }}</div>
        <ol class="mt-5 flex flex-col gap-1">
          <li
            v-for="(id, i) in steps"
            :key="id"
            class="flex items-start gap-2.5 rounded-md px-2 py-1.5"
            :class="{ 'bg-base-300': i === index }"
            :aria-current="i === index ? 'step' : undefined"
          >
            <span
              class="mt-px flex size-5 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
              :class="chipClass[stateOf(id, i)]"
            >
              <mdi:check v-if="stateOf(id, i) === 'done'" class="size-3.5" />
              <template v-else>{{ i + 1 }}</template>
            </span>
            <span class="min-w-0">
              <span class="block text-sm" :class="i === index ? 'font-semibold' : 'text-base-content/70'">
                {{ $t(`setup.steps.${id}`) }}
              </span>
              <span v-if="id === 'login'" class="text-base-content/40 block text-xs">
                {{ $t("setup.steps.login-note") }}
              </span>
              <span v-else-if="id === 'cloud'" class="text-base-content/40 block text-xs">
                {{ $t("setup.steps.cloud-note") }}
              </span>
            </span>
          </li>
        </ol>
      </aside>

      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="flex gap-1 px-4 pt-4 md:hidden" aria-hidden="true">
          <span
            v-for="(id, i) in steps"
            :key="id"
            class="h-0.75 flex-1 rounded-full transition-colors"
            :class="i === index ? 'bg-primary' : i < index ? 'bg-primary/45' : 'bg-base-content/15'"
          ></span>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-8 max-md:p-5">
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
        </div>

        <div class="border-base-content/10 bg-base-100 flex items-center gap-2 border-t px-8 py-4 max-md:px-5">
          <button v-if="!isLast" type="button" class="btn btn-ghost btn-sm" :disabled="busy" @click="close">
            {{ $t("setup.finish-later") }}
          </button>
          <div class="ml-auto flex items-center gap-2">
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
          </div>
        </div>
      </div>
    </div>
  </dialog>
</template>

<script lang="ts" setup>
import type { SetupStepHandle, SetupStepId, SetupStepState } from "@/composable/setup/setup";

const { status, loading, wizardOpen, fetchStatus } = useSetup();
const { linked, canLink } = useCloudSurface();
const { initialLoad } = useCloudConfig();
const setupSeen = useProfileStorage("setupSeen", false);

const modal = ref<HTMLDialogElement>();
const handle = useTemplateRef<SetupStepHandle>("step");

const steps = ref<SetupStepId[]>([]);
const index = ref(0);
const skipped = ref(new Set<SetupStepId>());

const currentId = computed<SetupStepId | undefined>(() => steps.value[index.value]);
const isLast = computed(() => index.value >= steps.value.length - 1 && steps.value.length > 0);
const busy = computed(() => !!handle.value?.busy);

const chipClass: Record<SetupStepState, string> = {
  done: "bg-success text-success-content",
  current: "bg-primary text-primary-content",
  skipped: "border-base-content/30 text-base-content/40 border border-dashed",
  todo: "bg-base-content/10 text-base-content/60",
};

function stateOf(id: SetupStepId, i: number): SetupStepState {
  if (i === index.value) return "current";
  if (skipped.value.has(id)) return "skipped";
  if (id === "login" && status.value) {
    if (setupLoginConfigured(status.value)) return "done";
    return i < index.value ? "skipped" : "todo";
  }
  return i < index.value ? "done" : "todo";
}

// A restart or the cloud round trip left a marker naming the step to come back to.
// When the page came back from Cloud, the wizard owns that return: the hash is
// dropped here, synchronously during setup, before CloudPopover's mounted hook
// (which waits on the cloud config fetch) can see it and open its own modal.
const resumeAtLoad = readSetupResume();
if (resumeAtLoad && window.location.hash === "#cloudLinked") {
  history.replaceState(history.state, "", window.location.pathname + window.location.search);
}

// Opened by hand it shows a spinner right away. Opened by itself it waits for the
// status first, so a server without the setup API never flashes an error at anyone.
async function open(startAt: SetupStepId | undefined, auto: boolean) {
  skipped.value = new Set();
  steps.value = [];
  index.value = 0;
  if (!auto && !modal.value?.open) modal.value?.showModal();

  await Promise.all([fetchStatus(), initialLoad]);
  const s = status.value;
  if (!s) {
    if (auto) wizardOpen.value = false;
    return;
  }
  if (!modal.value?.open) modal.value?.showModal();

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
  if (value && !modal.value?.open) {
    open(readSetupResume(), autoOpening);
    autoOpening = false;
  } else if (!value && modal.value?.open) close();
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
