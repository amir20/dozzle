// The cloud welcome normally opens when the link round trip lands back on
// #cloudLinked. When the setup wizard started that link, the wizard owns the return
// and hands the welcome over once it closes, so the two modals never open at once
// and people who link from the wizard still get the starter alerts.
//
// The marker lives in localStorage because the wizard can restart Dozzle between
// linking Cloud and closing, which reloads the page.

const PENDING_KEY = "DOZZLE_CLOUD_WELCOME_PENDING";

// Step the welcome should open at, set by whoever hands it over. CloudPopover owns
// the modal and consumes it.
const requestedStep = ref<number | null>(null);

export function markCloudWelcomePending() {
  try {
    localStorage.setItem(PENDING_KEY, "1");
  } catch {
    // Blocked storage: the welcome simply is not handed over.
  }
}

export function cloudWelcomePending(): boolean {
  try {
    return localStorage.getItem(PENDING_KEY) === "1";
  } catch {
    return false;
  }
}

export function clearCloudWelcomePending() {
  try {
    localStorage.removeItem(PENDING_KEY);
  } catch {
    // Nothing to clear.
  }
}

export function useCloudWelcome() {
  return {
    requestedStep,
    requestCloudWelcome: (step: number) => {
      requestedStep.value = step;
    },
  };
}
