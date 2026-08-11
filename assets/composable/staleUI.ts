import i18n from "@/modules/i18n";

// @ts-ignore
const { t } = i18n.global;

// True once we know the running UI was built from an older release than the server it talks to.
// Happens when Dozzle is upgraded underneath a long-lived tab: assets are from the old build,
// but the API is already the new one.
const stale = ref(false);

const reload = () => window.location.reload();

export const useStaleUI = () => {
  const { showToast } = useToast();

  const markStale = () => {
    if (stale.value) return;
    stale.value = true;

    showToast(
      {
        id: "stale-ui",
        title: t("alert.new-version.title"),
        message: t("alert.new-version.message"),
        type: "info",
        action: { label: t("button.reload"), handler: reload },
      },
      { once: true },
    );

    // Reloading while someone is tailing logs or sitting in a shell is rude, so wait until
    // the tab is backgrounded and pick up the new build then.
    useEventListener(document, "visibilitychange", () => {
      if (document.visibilityState === "hidden") reload();
    });
  };

  return { stale, markStale };
};
