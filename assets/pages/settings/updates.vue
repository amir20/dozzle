<template>
  <!-- Written so nobody needs the docs: when, which, what will update, and what keeps
       it safe. -->
  <SettingsSection :title="$t('settings.updates')" :desc="$t('setup.update.subtitle')">
    <InlineNotice v-if="status && !status.enableActions" type="info">
      {{ $t("auto-update.needs-actions") }}
      <template #actions>
        <router-link to="/settings/security" class="btn btn-sm">{{ $t("settings.security") }}</router-link>
      </template>
    </InlineNotice>

    <!-- Only when Dozzle can replace itself: otherwise the card has nothing to offer,
         and the form's notice already says why. A pinned tag's newer release is About's
         to announce, since there is nothing here to pull it with. -->
    <SetupSelfUpdateStatus v-if="status && running?.supported" :status="status" :auto-update="running" />

    <!-- With actions off, the notice above is the one thing to fix, so the form stays quiet. -->
    <SetupAutoUpdateForm v-if="status" :status="status" :notices="status.enableActions" autosave />
    <SetupStatusMissing v-else :loading="loading" />
  </SettingsSection>

  <!-- With the schedule off, or actions off, nothing updates on its own, whatever the
       labels say, so there is no list to show. -->
  <SettingsSection v-if="status && running" :title="$t('auto-update.containers')">
    <template #actions>
      <button
        v-if="config.imageCheckMode !== 'off' && rows.length"
        type="button"
        class="btn btn-ghost btn-xs text-base-content/60"
        :disabled="checking"
        @click="checkAll(true)"
      >
        <span v-if="checking" class="loading loading-spinner size-3"></span>
        {{ $t("toolbar.check-for-updates") }}
      </button>
    </template>

    <!-- Read only: what the mode and the labels add up to. Labels are set in compose.
         The empty state keeps the panel, so nothing reshuffles when a label lands. -->
    <div
      v-if="policies"
      class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
    >
      <div v-for="row in rows" :key="`${row.host}/${row.id}`" class="flex items-start gap-3 p-4">
        <div class="min-w-0 flex-1">
          <div class="truncate text-sm font-medium">{{ row.name }}</div>
          <div class="text-base-content/60 truncate font-mono text-xs">
            {{ row.image }}
            <span v-if="multipleHosts" class="text-base-content/40"> · {{ hostName(row.host) }}</span>
          </div>
        </div>
        <span class="status-pill shrink-0" :class="pill(statusOf(row))" :title="resultFor(row)?.reason">
          {{ $t(`auto-update.status-${statusOf(row)}`) }}
        </span>
      </div>
      <p v-if="!rows.length && mode === 'off'" class="text-base-content/60 p-4 text-sm">
        {{ $t("auto-update.mode-off-desc") }}
      </p>
      <i18n-t v-else-if="!rows.length" keypath="auto-update.empty" tag="p" class="text-base-content/60 p-4 text-sm">
        <template #label>
          <code class="font-mono">{{ UPDATE_LABEL }}=auto</code>
        </template>
      </i18n-t>
    </div>
  </SettingsSection>

  <!-- Static reassurance, so it reads as footnotes rather than a panel of its own. -->
  <SettingsSection :title="$t('auto-update.safety-title')">
    <ul class="text-base-content/60 flex flex-col gap-1.5 text-xs">
      <li v-for="line in safety" :key="line.key" class="flex items-start gap-2">
        <component :is="line.icon" class="mt-px size-3.5 shrink-0 opacity-60" />
        {{ $t(`auto-update.safety-${line.key}`) }}
      </li>
      <li>
        <a
          href="https://dozzle.dev/guide/actions#auto-updating-containers"
          target="_blank"
          rel="noopener"
          class="hover:text-base-content inline-flex items-center gap-2 transition-colors"
        >
          <mdi:book-open-variant class="size-3.5 shrink-0 opacity-60" />
          {{ $t("auto-update.learn-more") }}
          <mdi:open-in-new class="size-3 opacity-40" />
        </a>
      </li>
    </ul>
  </SettingsSection>
</template>

<script lang="ts" setup>
import IconVerify from "~icons/mdi/shield-check-outline";
import IconCleanup from "~icons/mdi/broom";
import IconSkip from "~icons/mdi/pin-outline";
import {
  DEFAULT_UPDATE_CONTAINERS_MODE,
  UPDATE_LABEL,
  type ContainerUpdatePolicy,
  type ImageStatusKind,
  imageStatusKind,
  willAutoUpdate,
} from "@/composable/containers/updatePolicy";

// The layout fetches the status. This page only shows in server mode.
const { status, loading } = useSetup();
const { hosts } = useHosts();
const { policies, fetchPolicies } = useUpdatePolicies();
const { checkAll, checking, resultFor } = useImageUpdates();

fetchPolicies();
checkAll();

// The schedule as it will actually run: on, and with actions on to run it.
const running = computed(() => {
  const update = status.value?.autoUpdate;
  return status.value?.enableActions && update && update.mode !== "off" ? update : undefined;
});

// The saved mode: the list says what the next run will do.
const mode = computed(() => status.value?.autoUpdate?.containers ?? DEFAULT_UPDATE_CONTAINERS_MODE);

const rows = computed(() =>
  running.value
    ? ((policies.value?.containers ?? []) as ContainerUpdatePolicy[]).filter((c) => willAutoUpdate(c, mode.value))
    : [],
);
const multipleHosts = computed(() => Object.keys(hosts.value).length > 1);
const hostName = (id: string) => hosts.value[id]?.name ?? id;

const statusOf = (row: ContainerUpdatePolicy) => imageStatusKind(resultFor(row));

function pill(kind: ImageStatusKind) {
  if (kind === "current") return "status-pill-success";
  if (kind === "available") return "status-pill-warning";
  return "status-pill-neutral";
}

const safety = [
  { key: "verify", icon: IconVerify },
  { key: "cleanup", icon: IconCleanup },
  { key: "skip", icon: IconSkip },
];
</script>
