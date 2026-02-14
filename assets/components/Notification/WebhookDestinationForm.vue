<template>
  <div class="space-y-4">
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

    <!-- Webhook URL -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">{{ $t("notifications.destination-form.webhook-url") }}</legend>
      <input
        v-model="webhookUrl"
        type="url"
        class="input focus:input-primary w-full text-base"
        :class="{ 'input-primary': isValidUrl, 'input-error': webhookUrl.trim() && !isValidUrl }"
        :placeholder="$t('notifications.destination-form.webhook-url-placeholder')"
      />
    </fieldset>

    <!-- Payload Format (create mode only) -->
    <fieldset v-if="!isEditing" class="fieldset">
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

    <!-- Template -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">
        {{ $t("notifications.destination-form.template") }}
        <span class="text-base-content/60 ml-2 text-sm font-normal">{{
          $t("notifications.destination-form.template-hint")
        }}</span>
      </legend>
      <div
        ref="templateEditorRef"
        class="border-base-content/20 focus-within:border-primary min-h-48 w-full overflow-auto rounded-lg border"
      ></div>
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
      <button class="btn" @click="close?.()">
        {{ $t("notifications.destination-form.cancel") }}
      </button>
      <button class="btn btn-primary" :disabled="!canSave" @click="saveDestination">
        <span v-if="isSaving" class="loading loading-spinner loading-sm"></span>
        {{ isEditing ? $t("notifications.destination-form.save") : $t("notifications.destination-form.add") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, TestWebhookResult } from "@/types/notifications";
import { createTemplateEditor } from "@/composable/templateEditor";
import { PAYLOAD_TEMPLATES, type PayloadFormat } from "./payloadTemplates";

const { close, onCreated, destination, isEditing } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
  isEditing: boolean;
}>();

const nameInput = ref<HTMLInputElement>();
const templateEditorRef = ref<HTMLElement>();
const name = ref(destination?.name ?? "");
useFocus(nameInput, { initialValue: true });
const webhookUrl = ref(destination?.url ?? "");
const payloadFormat = ref<PayloadFormat>(isEditing ? "custom" : "slack");
const template = ref(isEditing ? (destination?.template ?? "") : PAYLOAD_TEMPLATES[payloadFormat.value]);
const isTesting = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const testResult = ref<TestWebhookResult | null>(null);

let templateEditorView: Awaited<ReturnType<typeof createTemplateEditor>> | undefined;

function selectPayloadFormat(format: PayloadFormat) {
  payloadFormat.value = format;
  template.value = PAYLOAD_TEMPLATES[format];
  setEditorContent(template.value);
}

function setEditorContent(value: string) {
  if (!templateEditorView) return;
  templateEditorView.dispatch({
    changes: { from: 0, to: templateEditorView.state.doc.length, insert: value },
  });
}

onMounted(async () => {
  if (!templateEditorRef.value) return;
  templateEditorView = await createTemplateEditor({
    parent: templateEditorRef.value,
    initialValue: template.value,
    onChange: (v) => (template.value = v),
  });
});

onScopeDispose(() => {
  templateEditorView?.destroy();
});

const canTest = computed(() => webhookUrl.value.trim().length > 0);

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
  if (!isValidUrl.value) return false;
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
      type: "webhook",
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
