// The cloud welcome normally opens when the link round trip lands back on
// #cloudLinked. When the setup wizard started that link, the wizard owns the return
// and hands the welcome over once it closes, so the two modals never open at once
// and people who link from the wizard still get the starter alerts.
//
// The marker lives in localStorage because the wizard can restart Dozzle between
// linking Cloud and closing, which reloads the page.

const PENDING_KEY = "DOZZLE_CLOUD_WELCOME_PENDING";

// Set by whoever hands the welcome over. CloudPopover owns the modal and consumes it.
const requested = ref(false);

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
    requested,
    requestCloudWelcome: () => {
      requested.value = true;
    },
  };
}
