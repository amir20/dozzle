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

    <!-- Name (only for webhook type) -->
    <fieldset v-if="type === 'webhook'" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.name") }}</legend>
      <input
        ref="nameInput"
        v-model="name"
        type="text"
        class="input focus:input-primary w-full text-base"
        required
        :class="{ 'input-primary': name.trim().length > 0 }"
        :placeholder="$t('notifications.destination-form.name-placeholder')"
      />
    </fieldset>

    <!-- Cloud linked success (when editing cloud with prefix) -->
    <fieldset v-if="type === 'cloud' && destination?.prefix" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.api-key") }}</legend>
      <div class="join w-full">
        <input type="text" :value="destination.prefix + '...'" readonly class="input join-item input-success w-full" />
        <span class="join-item btn btn-success pointer-events-none">
          <mdi:check class="text-lg" />
        </span>
      </div>
      <p class="text-base-content/60 mt-2 text-sm">
        {{ $t("notifications.destination-form.cloud-settings-hint") }}
        <a :href="cloudSettingsUrl" target="_blank" class="link link-primary">
          {{ $t("notifications.destination-form.cloud-settings-link") }}
        </a>
      </p>
    </fieldset>

    <!-- Link Dozzle Cloud (only for cloud type, when creating or not linked) -->
    <div v-else-if="type === 'cloud'" class="card card-border border-primary/30 bg-primary/5">
      <div class="card-body items-center text-center">
        <mdi:cloud-outline class="text-primary text-4xl" />
        <h3 class="card-title">{{ $t("notifications.destination-form.link-cloud") }}</h3>
        <p class="text-base-content/60 text-sm">{{ $t("notifications.destination-form.cloud-description") }}</p>
        <a :href="cloudLinkUrl" class="btn btn-primary btn-lg mt-2">
          <mdi:link-variant class="text-lg" />
          {{ $t("notifications.destination-form.link-cloud-button") }}
        </a>
      </div>
    </div>

    <!-- Webhook URL (only for webhook type) -->
    <fieldset v-if="type === 'webhook'" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.webhook-url") }}</legend>
      <input
        v-model="webhookUrl"
        type="url"
        class="input focus:input-primary w-full text-base"
        :class="{ 'input-primary': isValidUrl, 'input-error': webhookUrl.trim() && !isValidUrl }"
        :placeholder="$t('notifications.destination-form.webhook-url-placeholder')"
      />
    </fieldset>

    <!-- Payload Format (only for webhook type) -->
    <fieldset v-if="type === 'webhook'" class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.payload-format") }}</legend>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="format in ['slack', 'discord', 'ntfy', 'custom'] as const"
          :key="format"
          type="button"
          class="btn btn-sm"
          :class="payloadFormat === format ? 'btn-primary' : 'btn-ghost'"
          @click="selectPayloadFormat(format)"
        >
          {{ $t(`notifications.destination-form.format-${format}`) }}
        </button>
      </div>
    </fieldset>

    <!-- Template (only for webhook type) -->
    <fieldset v-if="type === 'webhook'" class="fieldset">
      <legend class="fieldset-legend text-lg">
        {{ $t("notifications.destination-form.template") }}
        <span class="text-base-content/60 ml-2 text-sm font-normal">{{
          $t("notifications.destination-form.template-hint")
        }}</span>
      </legend>
      <textarea
        v-model="template"
        class="textarea focus:textarea-primary min-h-48 w-full font-mono text-sm"
        :class="{ 'textarea-primary': template.trim().length > 0 }"
      ></textarea>
    </fieldset>

    <!-- Error -->
    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- Test Result -->
    <div v-if="testResult" class="alert" :class="testResult.success ? 'alert-success' : 'alert-error'">
      <span v-if="testResult.success">
        {{ $t("notifications.destination-form.test-success") }}
        <span v-if="testResult.statusCode" class="opacity-70">({{ testResult.statusCode }})</span>
      </span>
      <span v-else>
        {{ testResult.error }}
      </span>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 pt-4">
      <button
        v-if="type === 'webhook'"
        class="btn"
        @click="testDestination"
        :disabled="!canTest || !isValidUrl || isTesting"
      >
        <span v-if="isTesting" class="loading loading-spinner loading-sm"></span>
        {{ $t("notifications.destination-form.test") }}
      </button>
      <div class="flex-1"></div>
      <button class="btn" @click="close?.()">{{ $t("notifications.destination-form.cancel") }}</button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveDestination">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.destination-form.save") : $t("notifications.destination-form.add") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, TestWebhookResult } from "@/types/notifications";

type PayloadFormat = "slack" | "discord" | "ntfy" | "custom";

const PAYLOAD_TEMPLATES: Record<PayloadFormat, string> = {
  slack: `{
  "text": "{{ .Container.Name }}",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*{{ .Container.Name }}*\\n{{ .Log.Message }}"
      }
    },
    {
      "type": "context",
      "elements": [
        {
          "type": "mrkdwn",
          "text": "Host: {{ .Container.Host }} | Image: {{ .Container.Image }}"
        }
      ]
    }
  ]
}`,
  discord: `{
  "content": "{{ .Container.Name }}",
  "embeds": [
    {
      "title": "{{ .Container.Name }}",
      "description": "{{ .Log.Message }}",
      "fields": [
        { "name": "Host", "value": "{{ .Container.Host }}", "inline": true },
        { "name": "Image", "value": "{{ .Container.Image }}", "inline": true }
      ]
    }
  ]
}`,
  ntfy: `{
  "topic": "dozzle-{{ .Container.Host }}",
  "title": "{{ .Container.Name }}",
  "message": "{{ .Log.Message }}"
}`,
  custom: `{
  "container": "{{ .Container.Name }}",
  "level": "{{ .Log.Level }}",
  "message": "{{ .Log.Message }}"
}`,
};

const {
  close,
  onCreated,
  destination,
  existingDispatchers = [],
} = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
  existingDispatchers?: Dispatcher[];
}>();

const hasExistingCloudDestination = computed(() => {
  // When editing, exclude the current destination from the check
  const others = isEditing ? existingDispatchers.filter((d) => d.id !== destination!.id) : existingDispatchers;
  return others.some((d) => d.type === "cloud");
});

const isEditing = !!destination;

const nameInput = ref<HTMLInputElement>();
const name = ref(destination?.name ?? "");
useFocus(nameInput, { initialValue: true });
const type = ref<"webhook" | "cloud">((destination?.type as "webhook" | "cloud") ?? "webhook");
const webhookUrl = ref(destination?.url ?? "");
const payloadFormat = ref<PayloadFormat>(isEditing ? "custom" : "slack");
const template = ref(isEditing ? (destination?.template ?? "") : PAYLOAD_TEMPLATES[payloadFormat.value]);
const isTesting = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const testResult = ref<TestWebhookResult | null>(null);

const callbackUrl = `${window.location.origin}${withBase("/")}`;
const cloudLinkUrl = `${__CLOUD_URL__}/link?appUrl=${encodeURIComponent(callbackUrl)}`;
const cloudSettingsUrl = `${__CLOUD_URL__}/settings`;

function selectPayloadFormat(format: PayloadFormat) {
  payloadFormat.value = format;
  template.value = PAYLOAD_TEMPLATES[format];
}

const canTest = computed(() => {
  if (type.value === "webhook") {
    return webhookUrl.value.trim().length > 0;
  }
  return false;
});

const isValidUrl = computed(() => {
  try {
    new URL(webhookUrl.value.trim());
    return true;
  } catch {
    return false;
  }
});

const canSave = computed(() => {
  if (isSaving.value) return false;
  if (type.value === "cloud") return false;
  if (type.value === "webhook") {
    if (!name.value.trim()) return false;
    if (!isValidUrl.value) return false;
  }
  return true;
});

async function testDestination() {
  if (!canTest.value) return;

  isTesting.value = true;
  testResult.value = null;

  try {
    const res = await fetch(withBase("/api/notifications/test-webhook"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url: webhookUrl.value.trim(),
        template: template.value.trim() || undefined,
      }),
    });

    const data: TestWebhookResult = await res.json();
    testResult.value = data;
  } catch (e) {
    testResult.value = { success: false, error: e instanceof Error ? e.message : "Test failed" };
  } finally {
    isTesting.value = false;
  }
}

async function saveDestination() {
  if (!canSave.value) return;

  isSaving.value = true;
  error.value = null;

  try {
    const input = {
      name: name.value.trim(),
      type: type.value,
      url: webhookUrl.value.trim(),
      template: template.value.trim() || undefined,
    };

    const url = isEditing
      ? withBase(`/api/notifications/dispatchers/${destination!.id}`)
      : withBase("/api/notifications/dispatchers");

    const res = await fetch(url, {
      method: isEditing ? "PUT" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });

    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || "Failed to save destination");
    }

    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to save destination";
  } finally {
    isSaving.value = false;
  }
}
</script>
