// An agent started with DOZZLE_ENABLE_ACTIONS or DOZZLE_ENABLE_SHELL set to false refuses those
// requests whatever the hub allows, and reports it when it connects. These
// combine the hub's settings with that report, per host.

const { hosts } = useHosts();

export const actionsAllowedOn = (hostId: string) => !!config.enableActions && !hosts.value[hostId]?.actionsDisabled;

export const updatesAllowedOn = (hostId: string) => !!config.enableUpdates && !hosts.value[hostId]?.actionsDisabled;

// Mirrors shellHostAllowed in internal/web/terminal.go: with no login the server
// only opens a shell on an IP or localhost, since a hostname can be DNS-rebound.
export const isIpOrLocalhost = (hostname: string) => {
  const h = hostname.replace(/^\[|\]$/g, "");
  return h === "localhost" || /^\d{1,3}(\.\d{1,3}){3}$/.test(h) || h.includes(":");
};

export const shellAllowedOn = (hostId: string) =>
  !!config.enableShell &&
  !hosts.value[hostId]?.shellDisabled &&
  (config.authProvider !== "none" || isIpOrLocalhost(location.hostname));
