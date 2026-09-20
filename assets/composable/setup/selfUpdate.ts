// Pulling a newer image and letting Dozzle replace its own container.
//
// Two surfaces offer this: the wizard's Auto-update step and the About footer in
// settings. They render it differently, but the phases, the progress maths and the
// wording of a failed pull have to match, so they live here.

import type { SetupStatus, SetupStepId } from "@/composable/setup/setup";

export type SelfUpdatePhase = "idle" | "pulling" | "up-to-date" | "restarting" | "timeout";

// Only an install that can see and replace its own container can be updated from
// the UI. A pinned tag can still be pulled by hand, it just reports up to date.
export function canSelfUpdate(status: SetupStatus): boolean {
  const reason = status.autoUpdate?.reason;
  return (
    !!status.autoUpdate &&
    status.enableActions &&
    reason !== "not-server" &&
    reason !== "no-container" &&
    reason !== "swarm-worker"
  );
}

export function useSelfUpdate({ resume }: { resume?: SetupStepId } = {}) {
  const { t } = useI18n();
  const { updateSelf, waitForRestart } = useSetup();

  const phase = ref<SelfUpdatePhase>("idle");
  const progress = ref<number>();
  const error = ref("");
  // The daemon's own words, kept under a plain-language summary for anyone debugging.
  const errorDetail = ref("");
  const busy = computed(() => phase.value === "pulling" || phase.value === "restarting");

  // Pull failures come back as raw daemon text. The common ones get a sentence a person
  // can act on; anything unrecognised is shown as it came.
  function describeUpdateError(image: string, message: string) {
    if (/pull access denied|repository does not exist|manifest unknown|not found|unauthorized|denied/i.test(message)) {
      error.value = t("setup.update.pull-denied", { image });
      errorDetail.value = message;
    } else if (message) {
      error.value = message;
      errorDetail.value = "";
    } else {
      error.value = t("setup.error.generic");
      errorDetail.value = "";
    }
  }

  async function updateNow(image: string) {
    error.value = "";
    errorDetail.value = "";
    progress.value = undefined;
    phase.value = "pulling";
    const pullProgress = createPullProgress();
    let launched = false;
    let finished = false;

    try {
      const response = await updateSelf();
      await readUpdateProgress(response, (event) => {
        if (event.status === "pulling") {
          progress.value = pullProgress(event) ?? progress.value;
        } else if (event.status === "done") {
          launched = finished = true;
        } else if (event.status === "up-to-date") {
          finished = true;
          phase.value = "up-to-date";
        } else if (event.status === "error") {
          finished = true;
          phase.value = "idle";
          describeUpdateError(image, event.error ?? "");
        }
      });
    } catch (e) {
      finished = true;
      phase.value = "idle";
      error.value =
        e instanceof SetupError && e.status === 403 ? t("setup.actions.no-access") : t("setup.error.generic");
    }

    if (!finished) {
      // The stream closed without saying how it went.
      phase.value = "idle";
      error.value = t("setup.error.generic");
      return;
    }
    if (!launched) return;

    // The new container comes back where the update was started from.
    if (resume) writeSetupResume(resume);
    phase.value = "restarting";
    if (!(await waitForRestart({ timeout: 180_000, mustGoDown: true }))) phase.value = "timeout";
  }

  return { phase, progress, error, errorDetail, busy, updateNow };
}
