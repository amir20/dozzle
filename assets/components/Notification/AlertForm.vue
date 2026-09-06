<template>
  <div class="flex min-h-full flex-col">
    <div class="space-y-6 p-4 pb-8">
      <div>
        <h2 class="text-2xl font-bold">
          {{ isEditing ? $t("notifications.alert-form.edit-title") : $t("notifications.alert-form.create-title") }}
        </h2>
        <p class="text-base-content/60">{{ $t(`notifications.alert-form.description-${alertType}`) }}</p>
      </div>

      <!-- 1. Alert type -->
      <section>
        <FormStepHeading :step="1" :title="$t('notifications.alert-form.alert-type')" />
        <div class="grid gap-2 sm:grid-cols-3">
          <button
            v-for="option in alertTypes"
            :key="option.type"
            type="button"
            class="card cursor-pointer border text-left transition-colors"
            :class="
              alertType === option.type
                ? 'border-primary bg-primary/10 ring-primary/40 ring-1'
                : 'border-base-content/15 hover:border-base-content/35 hover:bg-base-content/5'
            "
            :aria-pressed="alertType === option.type"
            @click="alertType = option.type"
          >
            <div class="card-body gap-1 p-3">
              <div class="flex items-center gap-2 font-semibold">
                <component
                  :is="option.icon"
                  :class="alertType === option.type ? 'text-primary' : 'text-base-content/50'"
                />
                {{ $t(`notifications.alert-form.${option.type}-alert`) }}
              </div>
              <div class="text-base-content/60 text-xs">
                {{ $t(`notifications.alert-form.${option.type}-alert-description`) }}
              </div>
            </div>
          </button>
        </div>
        <div v-if="typeChanged" class="alert alert-warning mt-2 py-2 text-sm">
          <mdi:alert-outline />
          <span>{{ $t("notifications.alert-form.type-switch-warning", { type: originalTypeLabel }) }}</span>
        </div>
      </section>

      <!-- 2. Containers -->
      <section>
        <FormStepHeading :step="2" :title="$t('notifications.alert-form.container-filter')" required />
        <ExpressionField
          v-model="containerExpression"
          :hint="$t('notifications.alert-form.container-filter-hint')"
          placeholder='name contains "api"'
          :error="containerResult?.error"
          :examples="containerExamples"
          :get-hints="containerHints"
        />
        <div class="mt-1 text-sm">
          <span v-if="isLoading && !containerResult" class="text-base-content/60">
            <span class="loading loading-spinner loading-xs align-middle"></span>
            {{ $t("notifications.alert-form.checking") }}
          </span>
          <template v-else-if="containerResult && !containerResult.error">
            <span v-if="containerResult.containers?.length" class="text-success">
              <mdi:check class="inline" />
              {{
                $t("notifications.alert-form.containers-match", {
                  count: containerResult.containers.length,
                  names: matchedNames,
                })
              }}
            </span>
            <span v-else class="text-warning">
              <mdi:alert class="inline" />
              {{ $t("notifications.alert-form.no-containers-match") }}
            </span>
          </template>
        </div>
      </section>

      <!-- 3. Condition -->
      <section>
        <FormStepHeading :step="3" :title="$t(`notifications.alert-form.${alertType}-filter`)" required />
        <KeepAlive>
          <LogAlertFields
            v-if="alertType === 'log'"
            ref="fieldsRef"
            :alert="alert"
            :prefill="prefill"
            :container-expression="containerExpression"
            :has-containers="hasContainers"
            :is-loading="isLoading"
            :validate-preview="validatePreview"
          />
          <MetricAlertFields
            v-else-if="alertType === 'metric'"
            ref="fieldsRef"
            :alert="alert"
            :prefill="prefill"
            :container-expression="containerExpression"
            :has-containers="hasContainers"
            :is-loading="isLoading"
            :validate-preview="validatePreview"
          />
          <EventAlertFields
            v-else
            ref="fieldsRef"
            :alert="alert"
            :prefill="prefill"
            :container-expression="containerExpression"
            :has-containers="hasContainers"
            :is-loading="isLoading"
            :validate-preview="validatePreview"
          />
        </KeepAlive>
      </section>

      <!-- 4. Destination -->
      <section>
        <FormStepHeading :step="4" :title="$t('notifications.alert-form.destination')" required />
        <div v-if="destinations.length" class="grid gap-2 sm:grid-cols-2">
          <button
            v-for="dest in destinations"
            :key="dest.id"
            type="button"
            class="card cursor-pointer border text-left transition-colors"
            :class="
              dispatcherId === dest.id
                ? 'border-primary bg-primary/10 ring-primary/40 ring-1'
                : 'border-base-content/15 hover:border-base-content/35 hover:bg-base-content/5'
            "
            :aria-pressed="dispatcherId === dest.id"
            @click="dispatcherId = dest.id"
          >
            <div class="card-body flex-row items-center gap-3 p-3">
              <mdi:webhook
                v-if="dest.type === 'webhook'"
                class="shrink-0 text-lg"
                :class="dispatcherId === dest.id ? 'text-primary' : 'text-base-content/50'"
              />
              <mdi:cloud
                v-else
                class="shrink-0 text-lg"
                :class="dispatcherId === dest.id ? 'text-primary' : 'text-base-content/50'"
              />
              <div class="min-w-0">
                <div class="truncate font-semibold">{{ dest.name }}</div>
                <div class="text-base-content/60 text-xs">
                  {{
                    dest.type === "webhook"
                      ? $t("notifications.destination.http-webhook")
                      : $t("notifications.destination.dozzle-cloud")
                  }}
                </div>
              </div>
              <mdi:check v-if="dispatcherId === dest.id" class="text-primary ml-auto shrink-0" />
            </div>
          </button>
        </div>
        <div v-else class="text-base-content/60 text-sm">{{ $t("notifications.alert-form.no-destinations") }}</div>
        <button type="button" class="btn btn-ghost btn-sm mt-2 gap-1" @click="addDestination">
          <mdi:plus />
          {{ $t("notifications.add-destination") }}
        </button>
      </section>

      <!-- 5. Name -->
      <section>
        <FormStepHeading :step="5" :title="$t('notifications.alert-form.alert-name')" required />
        <input
          v-model="alertName"
          type="text"
          class="input focus:input-primary w-full text-base"
          :class="alertName.trim() ? 'input-primary' : ''"
          required
          :placeholder="suggestedName || $t('notifications.alert-form.alert-name-placeholder')"
          @input="nameTouched = true"
        />
        <p class="text-base-content/60 mt-1 text-xs">{{ $t("notifications.alert-form.alert-name-hint") }}</p>
      </section>
    </div>

    <!-- Actions stay reachable in a form this tall -->
    <div class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 mt-auto border-t p-4">
      <div v-if="saveError" class="alert alert-error mb-3 py-2 text-sm">
        <span>{{ saveError }}</span>
      </div>

      <div v-if="confirmingDiscard" class="flex flex-wrap items-center justify-end gap-2">
        <span class="mr-auto text-sm">{{ $t("notifications.alert-form.discard-title") }}</span>
        <button class="btn btn-sm" @click="confirmingDiscard = false">
          {{ $t("notifications.alert-form.discard-keep") }}
        </button>
        <button class="btn btn-sm btn-error" @click="discard">
          {{ $t("notifications.alert-form.discard-confirm") }}
        </button>
      </div>

      <div v-else class="flex flex-wrap items-center justify-end gap-2">
        <span v-if="blockers.length" class="text-base-content/70 mr-auto text-sm">
          {{ $t("notifications.alert-form.still-needed", { fields: blockerLabels }) }}
        </span>
        <button class="btn" @click="close?.()">{{ $t("notifications.alert-form.cancel") }}</button>
        <button class="btn btn-primary" :disabled="!canSave" @click="save">
          <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
          {{ isEditing ? $t("notifications.alert-form.save") : $t("notifications.alert-form.create") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useAlertForm, type AlertType, type AlertPrefill, type SaveBlocker } from "@/composable/alertForm";
import LogAlertFields from "./LogAlertFields.vue";
import MetricAlertFields from "./MetricAlertFields.vue";
import EventAlertFields from "./EventAlertFields.vue";
import DestinationForm from "./DestinationForm.vue";
import FormStepHeading from "./FormStepHeading.vue";
import ExpressionField from "./ExpressionField.vue";
import type { Dispatcher, NotificationRule } from "@/types/notifications";
import TextBoxIcon from "~icons/mdi/text-box-outline";
import ChartLineIcon from "~icons/mdi/chart-line";
import BellRingIcon from "~icons/mdi/bell-ring-outline";

const props = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  alert?: NotificationRule;
  prefill?: AlertPrefill;
}>();

const { t } = useI18n();
const showDrawer = useDrawer();
// Captured in setup so addDestination can hand the drawer back to this same component.
const self = getCurrentInstance()!.type;

const {
  isEditing,
  alertName,
  containerExpression,
  dispatcherId,
  destinations,
  containerResult,
  containerHints,
  isLoading,
  isSaving,
  saveError,
  baseCanSave,
  saveAlert,
  validatePreview,
} = useAlertForm(props);

const alertTypes = [
  { type: "log" as const, icon: TextBoxIcon },
  { type: "metric" as const, icon: ChartLineIcon },
  { type: "event" as const, icon: BellRingIcon },
];

const containerExamples = ['name contains "api"', 'image startsWith "postgres"', 'labels["env"] == "prod"'];

function typeOf(alert?: NotificationRule): AlertType {
  if (alert?.metricExpression) return "metric";
  if (alert?.eventExpression) return "event";
  return "log";
}

const originalType = typeOf(props.alert);
const alertType = ref<AlertType>(props.alert ? originalType : (props.prefill?.alertType ?? "log"));

// Saving writes only the active type's expression, so switching type on an existing alert
// throws the other one away. Say so rather than letting it happen silently.
const typeChanged = computed(() => isEditing.value && alertType.value !== originalType);
const originalTypeLabel = computed(() => t(`notifications.alert-form.${originalType}-alert`));

const fieldsRef = ref<
  InstanceType<typeof LogAlertFields> | InstanceType<typeof MetricAlertFields> | InstanceType<typeof EventAlertFields>
>();

const matchedContainers = computed(() => containerResult.value?.containers ?? []);
const hasContainers = computed(() => matchedContainers.value.length > 0);
const matchedNames = computed(() => {
  const names = matchedContainers.value.map((c) => c.name);
  return names.length > 3 ? `${names.slice(0, 3).join(", ")}…` : names.join(", ");
});

// Name comes last, so offer one derived from what the user just built.
const nameTouched = ref(!!(props.alert?.name || props.prefill?.name));
const suggestedName = computed(() => {
  const type = t(`notifications.alert-form.${alertType.value}-alert`);
  if (matchedNames.value) return `${type} · ${matchedNames.value}`;
  if (containerExpression.value.trim()) return `${type} · ${containerExpression.value.trim()}`;
  return "";
});
watch(suggestedName, (name) => {
  if (!nameTouched.value) alertName.value = name;
});

const blockers = computed<SaveBlocker[]>(() => {
  const missing: SaveBlocker[] = [];
  if (!containerExpression.value.trim() || containerResult.value?.error) missing.push("container-expression");
  if (!fieldsRef.value?.canSave) missing.push("condition");
  if (dispatcherId.value < 0) missing.push("destination");
  if (!alertName.value.trim()) missing.push("name");
  return missing;
});
const blockerLabels = computed(() => blockers.value.map((b) => t(`notifications.alert-form.missing.${b}`)).join(", "));

const canSave = computed(() => baseCanSave.value && (fieldsRef.value?.canSave ?? false));

async function save() {
  if (!canSave.value || !fieldsRef.value) return;
  registerGuard(null);
  await saveAlert(fieldsRef.value.typeFields);
}

// Unsaved-changes guard. Esc and the backdrop are one keystroke away from throwing away a
// half-built expression, so confirm in the footer instead of closing.
const registerGuard = useDrawerCloseGuard();
const confirmingDiscard = ref(false);
const initial = ref<string>();
const snapshot = computed(() =>
  JSON.stringify({
    alertType: alertType.value,
    alertName: alertName.value,
    containerExpression: containerExpression.value,
    dispatcherId: dispatcherId.value,
    typeFields: fieldsRef.value?.typeFields,
  }),
);
onMounted(async () => {
  await nextTick();
  initial.value = snapshot.value;
});
const isDirty = computed(() => initial.value !== undefined && initial.value !== snapshot.value);

registerGuard(() => {
  if (!isDirty.value || isSaving.value) return true;
  confirmingDiscard.value = true;
  return false;
});

function discard() {
  registerGuard(null);
  props.close?.();
}

// The drawer holds one component at a time, so adding a destination means handing it over
// and coming back with everything the user has typed so far.
function addDestination() {
  const carried: AlertPrefill = {
    name: nameTouched.value ? alertName.value : undefined,
    alertType: alertType.value,
    containerExpression: containerExpression.value,
    ...(fieldsRef.value?.typeFields as Partial<AlertPrefill>),
  };
  registerGuard(null);
  showDrawer(
    DestinationForm,
    {
      onCreated: (created?: Dispatcher) => {
        showDrawer(
          self,
          {
            alert: props.alert,
            onCreated: props.onCreated,
            prefill: { ...carried, dispatcherId: created?.id ?? dispatcherId.value },
          },
          "lg",
        );
      },
    },
    "md",
  );
}
</script>
