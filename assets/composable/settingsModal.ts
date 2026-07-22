// Shared open state for the fullscreen settings popup. Lives outside the layout
// so any surface (topbar cog, actions menu) can open the same modal without
// prop-drilling. Enabled per-user via the `settingsAsPopup` setting; when off,
// those surfaces navigate to the /settings route instead.
const open = ref(false);

export function useSettingsModal() {
  return {
    open,
    openSettings: () => {
      open.value = true;
    },
    closeSettings: () => {
      open.value = false;
    },
  };
}
