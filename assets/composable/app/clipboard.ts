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
  // readonly keeps the on-screen keyboard down on mobile, fixed positioning keeps
  // the page from scrolling to a field nobody can see, and 16px keeps iOS from
  // zooming towards it.
  textarea.setAttribute("readonly", "");
  textarea.setAttribute("aria-hidden", "true");
  textarea.style.cssText = "position:fixed;top:0;left:0;width:1px;height:1px;opacity:0;font-size:16px;";
  copyHost().appendChild(textarea);

  // Whatever the user had selected is about to be replaced, so put it back after.
  const selection = window.getSelection();
  const previous = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : undefined;

  // select() alone is a no-op on iOS Safari; a range plus an explicit selection
  // range is what actually takes there, and is harmless everywhere else.
  textarea.select();
  const range = document.createRange();
  range.selectNodeContents(textarea);
  selection?.removeAllRanges();
  selection?.addRange(range);
  textarea.setSelectionRange(0, value.length);

  // execCommand reports failure by returning false rather than throwing, and unlike
  // VueUse's legacy mode we care about the answer.
  let copied = false;
  try {
    copied = document.execCommand?.("copy") ?? false;
  } catch {
    copied = false;
  }

  textarea.remove();
  if (previous) {
    selection?.removeAllRanges();
    selection?.addRange(previous);
  }
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
    // A link or an image tag is worth offering for manual selection. A stack trace
    // is not: past a line or two the toast becomes the thing in the way.
    const selectable = manual && manual.length <= 200 && !manual.includes("\n") ? escapeHtml(manual) : "";
    showToast(
      {
        title: insecure ? t("error.copy-insecure") : t("error.copy-not-supported"),
        message: [insecure ? t("error.copy-insecure-hint") : "", selectable].filter(Boolean).join("<br>"),
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
