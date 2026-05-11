<template>
  <div class="space-y-4 p-4">
    <div class="mb-6">
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

    <!-- Type Selection (only when creating) -->
    <fieldset v-if="!isEditing" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.type") }}</legend>
      <div class="space-y-3">
        <label
          class="card card-border cursor-pointer transition-colors"
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
          class="card card-border cursor-pointer transition-colors"
          :class="[
            type === 'cloud' ? 'border-primary bg-primary/10' : '',
            isCloudLinked ? 'cursor-not-allowed opacity-50' : '',
          ]"
        >
          <div class="card-body flex-row items-center gap-3 p-4">
            <input type="radio" v-model="type" value="cloud" class="radio radio-primary" :disabled="isCloudLinked" />
            <div>
              <div class="font-semibold">{{ $t("notifications.destination-form.cloud-title") }}</div>
              <div class="text-base-content/60 text-sm">
                {{ $t("notifications.destination-form.cloud-description") }}
              </div>
              <div v-if="isCloudLinked" class="text-success mt-1 text-xs">
                <mdi:check class="inline" />
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

const { close, onCreated, destination } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
}>();

const isEditing = !!destination;
const type = ref<"webhook" | "cloud">((destination?.type as "webhook" | "cloud") ?? "webhook");

const { cloudConfig, fetchCloudConfig } = useCloudConfig();
const isCloudLinked = computed(() => !!cloudConfig.value?.linked);

onMounted(() => fetchCloudConfig());
</script>
