// An agent started with --disable-actions or --disable-shell refuses those
// requests whatever the hub allows, and reports it when it connects. These
// combine the hub's settings with that report, per host.

const { hosts } = useHosts();

export const actionsAllowedOn = (hostId: string) => !!config.enableActions && !hosts.value[hostId]?.actionsDisabled;

export const updatesAllowedOn = (hostId: string) => !!config.enableUpdates && !hosts.value[hostId]?.actionsDisabled;

export const shellAllowedOn = (hostId: string) => !!config.enableShell && !hosts.value[hostId]?.shellDisabled;
