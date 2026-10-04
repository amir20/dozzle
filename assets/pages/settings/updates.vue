<template>
  <!-- Written so nobody needs the docs: when, which, what will update, and what keeps
       it safe. -->
  <section class="flex flex-col gap-4">
    <div>
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">{{ $t("settings.updates") }}</h2>
      <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.update.subtitle") }}</p>
    </div>

    <InlineNotice v-if="status && !status.enableActions" type="info">
      {{ $t("auto-update.needs-actions") }}
      <template #actions>
        <router-link to="/settings/security" class="btn btn-sm">{{ $t("settings.security") }}</router-link>
      </template>
    </InlineNotice>

    <SelfUpdateStatus v-if="status && scheduled" :status="status" :auto-update="scheduled" />

    <AutoUpdateForm v-if="status" :status="status" autosave />
    <SetupStatusMissing v-else :loading="loading" />
  </section>

  <section v-if="status" class="flex flex-col gap-3">
    <div class="flex items-end justify-between gap-3">
      <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
        {{ $t("auto-update.containers") }}
      </h2>
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
    </div>

    <!-- Read only: what the mode and the labels add up to. Labels are set in compose. -->
    <div
      v-if="rows.length"
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
    </div>
    <p v-else-if="policies && mode === 'off'" class="text-base-content/60 text-sm">
      {{ $t("auto-update.mode-off-desc") }}
    </p>
    <i18n-t v-else-if="policies" keypath="auto-update.empty" tag="p" class="text-base-content/60 text-sm">
      <template #label>
        <code class="font-mono">{{ UPDATE_LABEL }}=auto</code>
      </template>
    </i18n-t>
  </section>

  <section class="flex flex-col gap-3">
    <h2 class="text-base-content/60 text-xs font-semibold tracking-wide uppercase">
      {{ $t("auto-update.safety-title") }}
    </h2>
    <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
      <div v-for="line in safety" :key="line.key" class="flex items-start gap-3 p-4">
        <div class="bg-info/10 text-info shrink-0 rounded-full p-1.5">
          <component :is="line.icon" class="size-4" />
        </div>
        <p class="text-sm">{{ $t(`auto-update.safety-${line.key}`) }}</p>
      </div>
      <div class="p-2">
        <a
          href="https://dozzle.dev/guide/actions#auto-updating-containers"
          target="_blank"
          rel="noopener"
          class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
        >
          <mdi:book-open-variant class="size-4 opacity-60" />
          <span class="flex-1">{{ $t("auto-update.learn-more") }}</span>
          <mdi:open-in-new class="size-3.5 opacity-40" />
        </a>
      </div>
    </div>
  </section>
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

const scheduled = computed(() => {
  const update = status.value?.autoUpdate;
  return update && update.mode !== "off" ? update : undefined;
});

// The saved mode: the list says what the next run will do.
const mode = computed(() => status.value?.autoUpdate?.containers ?? DEFAULT_UPDATE_CONTAINERS_MODE);

const rows = computed(() =>
  ((policies.value?.containers ?? []) as ContainerUpdatePolicy[]).filter((c) => willAutoUpdate(c, mode.value)),
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
