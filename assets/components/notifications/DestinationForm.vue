<template>
  <div class="flex min-h-full flex-col space-y-6 p-4">
    <div>
      <h2 class="text-2xl font-bold">
        <template v-if="type === 'cloud'">
          {{ $t("notifications.destination-form.cloud-title") }}
        </template>
        <template v-else>
          {{
            isEditing
              ? $t("notifications.destination-form.edit-title")
              : $t("notifications.destination-form.create-title")
          }}
        </template>
      </h2>
      <p class="text-base-content/60">
        <template v-if="type === 'cloud'">
          {{ $t("notifications.destination-form.cloud-description") }}
        </template>
        <template v-else>
          {{ $t("notifications.destination-form.description") }}
        </template>
      </p>
    </div>

    <!-- Type Selection (only when creating). Same selectable cards as the alert form's
         type picker rather than a radio list, so the two drawers pick things the same way. -->
    <section v-if="!isEditing">
      <FormStepHeading :step="1" :title="$t('notifications.destination-form.type')" />
      <div class="grid gap-2 sm:grid-cols-2">
        <button
          v-for="option in types"
          :key="option.type"
          type="button"
          class="card border text-left transition-colors"
          :class="[
            type === option.type
              ? 'border-primary bg-primary/10 ring-primary/40 ring-1'
              : 'border-base-content/15 hover:border-base-content/35 hover:bg-base-content/5',
            option.disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
          ]"
          :aria-pressed="type === option.type"
          :disabled="option.disabled"
          @click="type = option.type"
        >
          <div class="card-body gap-1 p-3">
            <div class="flex items-center gap-2 font-semibold">
              <component :is="option.icon" :class="type === option.type ? 'text-primary' : 'text-base-content/50'" />
              {{ $t(`notifications.destination-form.${option.type}-title`) }}
            </div>
            <div class="text-base-content/60 text-xs">
              {{ $t(`notifications.destination-form.${option.type}-description`) }}
            </div>
            <div v-if="option.disabled" class="text-success flex items-center gap-1 text-xs">
              <mdi:check class="size-3.5 shrink-0" />
              {{ $t("notifications.destination-form.cloud-exists") }}
            </div>
          </div>
        </button>
      </div>
    </section>

    <!-- Type-specific form -->
    <WebhookDestinationForm
      v-if="type === 'webhook'"
      :destination="destination"
      :existing-dispatchers="existingDispatchers"
      :close="close"
      :on-created="onCreated"
      :is-editing="isEditing"
    />
    <CloudDestinationForm v-else :destination="destination" :close="close" />
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher } from "@/types/notifications";
import WebhookDestinationForm from "./WebhookDestinationForm.vue";
import CloudDestinationForm from "./CloudDestinationForm.vue";
import FormStepHeading from "@/components/ui/FormStepHeading.vue";
import WebhookIcon from "~icons/mdi/webhook";
import CloudIcon from "~icons/mdi/cloud-outline";

const { close, onCreated, destination, existingDispatchers } = defineProps<{
  close?: () => void;
  onCreated?: (created?: Dispatcher) => void;
  destination?: Dispatcher;
  /** Used to warn when a name is already taken; the picker in the alert form goes by name. */
  existingDispatchers?: Dispatcher[];
}>();

const isEditing = !!destination;
const type = ref<"webhook" | "cloud">((destination?.type as "webhook" | "cloud") ?? "webhook");

const { cloudConfig, fetchCloudConfig } = useCloudConfig();
const isCloudLinked = computed(() => !!cloudConfig.value?.linked);

// Only one cloud account can be linked, so the card stays visible but unpickable once it is.
const types = computed(() => [
  { type: "webhook" as const, icon: WebhookIcon, disabled: false },
  { type: "cloud" as const, icon: CloudIcon, disabled: isCloudLinked.value },
]);

onMounted(() => fetchCloudConfig());
</script>
