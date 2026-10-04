<template>
  <!-- The host list's Add host panel, agents first. Swarm and Kubernetes find their
       hosts themselves, so there it only lists them. -->
  <SettingsSection :title="$t('settings.hosts')" :desc="$t('setup.hosts.subtitle')">
    <AddHostPanel v-if="status?.canAddAgents" :status="status" list-first />
    <SetupStatusMissing v-else-if="server && !status" :loading="loading" />
    <template v-else>
      <InlineNotice type="info">
        {{ server ? $t("setup.hosts.error-unsupported-mode") : $t("settings.hosts-found") }}
      </InlineNotice>
      <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <li v-for="host in list" :key="host.id" class="flex items-center gap-3 p-4">
          <div class="relative shrink-0">
            <HostIcon :type="host.type" class="size-4 opacity-60" />
            <span
              class="ring-base-200 absolute -right-0.5 -bottom-0.5 size-1.5 rounded-full ring-2"
              :class="host.available ? 'bg-success' : 'bg-error'"
            ></span>
          </div>
          <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ host.name }}</span>
          <span class="text-base-content/60 shrink-0 text-xs">
            {{ host.available ? $t("setup.hosts.connected") : $t("setup.hosts.offline") }}
          </span>
        </li>
      </ul>
    </template>
  </SettingsSection>
</template>

<script lang="ts" setup>
// The layout fetches the status. canAddAgents is false wherever the host list is not
// this hub's own (and the API is absent outside server mode).
const { status, loading } = useSetup();
const { hosts } = useHosts();
const server = config.mode === "server";
const list = computed(() => Object.values(hosts.value).sort((a, b) => a.name.localeCompare(b.name)));
</script>
