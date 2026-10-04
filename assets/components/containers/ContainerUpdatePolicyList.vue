<template>
  <div class="flex flex-col gap-2">
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
          <div class="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
            <span class="status-pill" :class="pill(statusOf(row))" :title="resultFor(row)?.reason">
              {{ $t(`auto-update.status-${statusOf(row)}`) }}
            </span>
            <span v-if="row.last" class="text-base-content/60" :title="row.last.error">
              <i18n-t :keypath="row.last.status === 'done' ? 'auto-update.last-done' : 'auto-update.last-failed'">
                <template #time><RelativeTime :date="new Date(row.last.at)" /></template>
              </i18n-t>
            </span>
            <span v-if="heldBack(row)" class="text-base-content/40">{{ $t("auto-update.dozzle-only-waits") }}</span>
          </div>
        </div>

        <span v-if="lockOf(row) === 'self'" class="text-base-content/60 shrink-0 pt-1 text-xs">
          {{ $t("auto-update.self") }}
        </span>
        <span
          v-else-if="lockOf(row) === 'label'"
          class="text-base-content/60 flex shrink-0 items-center gap-1 pt-1 text-xs"
          :title="`${UPDATE_LABEL}=${row.policy}`"
        >
          <mdi:lock-outline class="size-3.5" />
          {{ $t(`auto-update.policy-${row.policy}`) }} · {{ $t("auto-update.set-by-label") }}
        </span>
        <select
          v-else
          class="select select-sm w-auto shrink-0"
          :aria-label="$t('auto-update.menu')"
          :value="row.choice ?? row.policy"
          :disabled="!!lockOf(row) || saving === row.id"
          @change="choose(row, ($event.target as HTMLSelectElement).value as UpdatePolicy)"
        >
          <option v-for="policy in UPDATE_POLICIES" :key="policy" :value="policy">
            {{ $t(`auto-update.policy-${policy}`) }}
          </option>
        </select>
      </div>
    </div>
    <p v-else-if="policies" class="text-base-content/60 text-sm">{{ $t("auto-update.empty") }}</p>

    <p v-if="policies && !policies.persisted" class="text-base-content/40 text-xs">
      {{ $t("auto-update.needs-volume") }}
    </p>
    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import {
  type ContainerUpdatePolicy,
  type ImageStatusKind,
  type UpdatePolicy,
  UPDATE_LABEL,
  UPDATE_POLICIES,
  UpdatePolicyError,
  heldBack,
  imageStatusKind,
  policyLock,
} from "@/composable/containers/updatePolicy";

const { t } = useI18n();
const { hosts } = useHosts();
const { policies, fetchPolicies, setPolicy } = useUpdatePolicies();
const { checkAll, resultFor } = useImageUpdates();

fetchPolicies(true);
checkAll();

const rows = computed(() => (policies.value?.containers ?? []) as ContainerUpdatePolicy[]);
const multipleHosts = computed(() => Object.keys(hosts.value).length > 1);
const hostName = (id: string) => hosts.value[id]?.name ?? id;

const lockOf = (row: ContainerUpdatePolicy) => policyLock(row, policies.value);
const statusOf = (row: ContainerUpdatePolicy) => imageStatusKind(row.policy, resultFor(row));

function pill(kind: ImageStatusKind) {
  if (kind === "current") return "status-pill-success";
  if (kind === "available") return "status-pill-warning";
  return "status-pill-neutral";
}

const saving = ref<string>();
const error = ref("");

async function choose(row: ContainerUpdatePolicy, policy: UpdatePolicy) {
  saving.value = row.id;
  error.value = "";
  try {
    await setPolicy([row], policy);
  } catch (e) {
    error.value =
      e instanceof UpdatePolicyError && e.code === "set-by-label"
        ? t("auto-update.set-by-label")
        : e instanceof UpdatePolicyError && e.code === "not-persisted"
          ? t("auto-update.needs-volume")
          : t("auto-update.save-failed");
    // Put the select back to what is saved.
    await fetchPolicies(true);
  } finally {
    saving.value = undefined;
  }
}
</script>
