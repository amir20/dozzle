// The async clipboard API only exists in a secure context, so on a plain http://
// install navigator.clipboard is simply not there. Every copy in Dozzle falls back
// to the old execCommand("copy") trick, and when that fails too it says so instead
// of showing a "Copied" toast that is not true.

type CopyOptions = {
  // Replaces the body of the success toast. Toast messages render as HTML, so
  // anything coming from a container has to be escaped by the caller.
  message?: string;
  // Shown in the failure toast so the value can still be selected by hand.
  manual?: string;
  // Skips the success toast for buttons that already confirm inline. A failure
  // is always announced: that is the whole point.
  quiet?: boolean;
  // Offered in the failure toast when the surface has a way to deliver the same
  // content without the clipboard, e.g. downloading the logs instead.
  action?: { label: string; handler: () => void };
};

// While a <dialog> is open as a modal, everything outside it is inert: a textarea
// parked on document.body cannot be selected, so execCommand copies nothing. Drawers
// are modal dialogs (SideDrawer.vue), so a copy button inside one has to put its
// scratch textarea in the same subtree.
function copyHost() {
  const dialogs = [...document.querySelectorAll("dialog[open]")];
  const focused = dialogs.filter((dialog) => dialog.contains(document.activeElement));
  return focused.at(-1) ?? dialogs.at(-1) ?? document.body;
}

function legacyCopy(value: string) {
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "absolute";
  textarea.style.opacity = "0";
  copyHost().appendChild(textarea);
  textarea.select();
  // execCommand reports failure by returning false rather than throwing, and unlike
  // VueUse's legacy mode we care about the answer.
  let copied = false;
  try {
    copied = document.execCommand("copy");
  } catch {
    copied = false;
  }
  textarea.remove();
  return copied;
}

export function useCopy() {
  const { showToast } = useToast();
  const { t } = useI18n();
  const copied = ref(false);
  let reset: ReturnType<typeof setTimeout> | undefined;

  function succeeded({ message, quiet }: CopyOptions) {
    copied.value = true;
    clearTimeout(reset);
    reset = setTimeout(() => (copied.value = false), 1500);
    if (!quiet) {
      showToast(
        { title: t("toasts.copied.title"), message: message ?? t("toasts.copied.message"), type: "info" },
        { expire: 2000 },
      );
    }
    return true;
  }

  function failed({ manual, action }: CopyOptions) {
    const insecure = !window.isSecureContext;
    showToast(
      {
        title: insecure ? t("error.copy-insecure") : t("error.copy-not-supported"),
        message: [insecure ? t("error.copy-insecure-hint") : "", manual ? escapeHtml(manual) : ""]
          .filter(Boolean)
          .join("<br>"),
        type: "warning",
        action,
      },
      { expire: 10000 },
    );
    return false;
  }

  async function copy(value: string, options: CopyOptions = {}) {
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(value);
        return succeeded(options);
      } catch {
        // A denied permission or a lost user gesture still leaves the legacy path.
      }
    }
    return legacyCopy(value) ? succeeded(options) : failed({ manual: value, ...options });
  }

  // Content that has to be fetched first. Safari drops the user gesture across an
  // await, so the async API is handed the pending blob instead of the resolved text.
  // Without that API there is nothing to do but wait for the text and try execCommand.
  async function copyLazy(fetchText: () => Promise<string>, options: CopyOptions = {}) {
    let pending: Promise<string> | undefined;
    const once = () => (pending ??= fetchText());

    if (navigator.clipboard?.write && typeof ClipboardItem !== "undefined") {
      try {
        const blob = once().then((text) => new Blob([text], { type: "text/plain" }));
        // The clipboard write rejects with its own error when the blob does, so mark
        // this branch handled and let the rethrow below carry the real one.
        blob.catch(() => {});
        await navigator.clipboard.write([new ClipboardItem({ "text/plain": blob })]);
        return succeeded(options);
      } catch {
        // Rethrows when it was the fetch that failed: that is the caller's error to
        // report, not a clipboard problem.
        await once();
      }
    }

    return legacyCopy(await once()) ? succeeded(options) : failed(options);
  }

  return { copy, copyLazy, copied };
}
