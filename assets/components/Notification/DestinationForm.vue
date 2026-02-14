<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">
        {{
          isEditing
            ? $t("notifications.destination-form.edit-title")
            : $t("notifications.destination-form.create-title")
        }}
      </h2>
      <p class="text-base-content/60">{{ $t("notifications.destination-form.description") }}</p>
    </div>

    <!-- Link Success Alert -->
    <div v-if="showLinkSuccess" class="alert alert-success">
      <mdi:check-circle class="text-lg" />
      <div>
        <div class="font-semibold">{{ $t("notifications.cloud-link-success.title") }}</div>
        <div class="text-sm">{{ $t("notifications.cloud-link-success.message") }}</div>
      </div>
    </div>

    <!-- Type Selection -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.type") }}</legend>
      <div class="space-y-3">
        <label
          class="card card-border 20 cursor-pointer transition-colors"
          :class="type === 'webhook' ? 'border-primary bg-primary/10' : ''"
        >
          <div class="card-body flex-row items-center gap-3 p-4">
            <input type="radio" v-model="type" value="webhook" class="radio radio-primary" />
            <div>
              <div class="font-semibold">{{ $t("notifications.destination-form.webhook-title") }}</div>
              <div class="text-base-content/60 text-sm">
                {{ $t("notifications.destination-form.webhook-description") }}
              </div>
            </div>
          </div>
        </label>
        <label
          class="card card-border border-base-content/20 transition-colors"
          :class="[
            type === 'cloud' ? 'border-primary bg-primary/10' : '',
            hasExistingCloudDestination && type !== 'cloud' ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
          ]"
        >
          <div class="card-body flex-row items-center gap-3 p-4">
            <input
              type="radio"
              v-model="type"
              value="cloud"
              class="radio radio-primary"
              :disabled="hasExistingCloudDestination && type !== 'cloud'"
            />
            <div>
              <div class="font-semibold">{{ $t("notifications.destination-form.cloud-title") }}</div>
              <div class="text-base-content/60 text-sm">
                {{ $t("notifications.destination-form.cloud-description") }}
              </div>
              <div v-if="hasExistingCloudDestination && type !== 'cloud'" class="text-warning mt-1 text-xs">
                {{ $t("notifications.destination-form.cloud-exists") }}
              </div>
            </div>
          </div>
        </label>
      </div>
    </fieldset>

    <!-- Type-specific form -->
    <WebhookDestinationForm
      v-if="type === 'webhook'"
      :destination="destination"
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

const {
  close,
  onCreated,
  destination,
  existingDispatchers = [],
  showLinkSuccess = false,
} = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
  existingDispatchers?: Dispatcher[];
  showLinkSuccess?: boolean;
}>();

const isEditing = !!destination;
const type = ref<"webhook" | "cloud">((destination?.type as "webhook" | "cloud") ?? "webhook");

const hasExistingCloudDestination = computed(() => {
  const others = isEditing ? existingDispatchers.filter((d) => d.id !== destination!.id) : existingDispatchers;
  return others.some((d) => d.type === "cloud");
});
</script>
