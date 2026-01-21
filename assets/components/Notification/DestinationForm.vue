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

    <!-- Name -->
    <fieldset class="fieldset">
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
          class="card card-border border-base-content/20 cursor-pointer transition-colors"
          :class="type === 'cloud' ? 'border-primary bg-primary/10' : ''"
        >
          <div class="card-body flex-row items-center gap-3 p-4">
            <input type="radio" v-model="type" value="cloud" class="radio radio-primary" />
            <div>
              <div class="font-semibold">{{ $t("notifications.destination-form.cloud-title") }}</div>
              <div class="text-base-content/60 text-sm">
                {{ $t("notifications.destination-form.cloud-description") }}
              </div>
            </div>
          </div>
        </label>
      </div>
    </fieldset>

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
      <button class="btn" @click="testDestination" :disabled="!canTest || !isValidUrl || isTesting">
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
import { useMutation } from "@urql/vue";
import {
  CreateDispatcherDocument,
  UpdateDispatcherDocument,
  TestWebhookDocument,
  type Dispatcher,
  type TestWebhookResult,
} from "@/types/graphql";

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

const { close, onCreated, destination } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
}>();

const createMutation = useMutation(CreateDispatcherDocument);
const updateMutation = useMutation(UpdateDispatcherDocument);
const testMutation = useMutation(TestWebhookDocument);

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
  if (!name.value.trim()) return false;
  if (type.value === "webhook" && !isValidUrl.value) return false;
  return true;
});

async function testDestination() {
  if (!canTest.value) return;

  isTesting.value = true;
  testResult.value = null;

  try {
    const input = {
      url: webhookUrl.value.trim(),
      template: template.value.trim() || undefined,
    };

    const result = await testMutation.executeMutation({ input });

    if (result.error) {
      testResult.value = { success: false, error: result.error.message };
    } else if (result.data?.testWebhook) {
      testResult.value = result.data.testWebhook;
    }
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
      url: type.value === "webhook" ? webhookUrl.value.trim() : undefined,
      template: type.value === "webhook" && template.value.trim() ? template.value.trim() : undefined,
    };

    const result = isEditing
      ? await updateMutation.executeMutation({ id: destination!.id, input })
      : await createMutation.executeMutation({ input });

    if (result.error) {
      throw new Error(result.error.message);
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
