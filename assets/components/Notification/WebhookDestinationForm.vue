<template>
  <div class="flex min-h-full flex-1 flex-col">
    <div class="space-y-6 pb-8">
      <!-- 1. Name -->
      <section>
        <FormStepHeading :step="1" :title="$t('notifications.destination-form.name')" required />
        <input
          ref="nameInput"
          v-model="name"
          type="text"
          class="input focus:input-primary w-full text-base"
          required
          :class="{ 'input-primary': name.trim().length > 0, 'input-warning': isDuplicateName }"
          :placeholder="$t('notifications.destination-form.name-placeholder')"
        />
        <p v-if="isDuplicateName" class="text-warning mt-1 text-xs">
          <mdi:alert class="inline" />
          {{ $t("notifications.destination-form.name-taken") }}
        </p>
        <p v-else class="text-base-content/60 mt-1 text-xs">{{ $t("notifications.destination-form.name-hint") }}</p>
      </section>

      <!-- 2. Webhook URL -->
      <section>
        <FormStepHeading :step="2" :title="$t('notifications.destination-form.webhook-url')" required />
        <input
          v-model="webhookUrl"
          type="url"
          class="input focus:input-primary w-full text-base"
          :class="{ 'input-primary': isValidUrl, 'input-error': webhookUrl.trim() && !isValidUrl }"
          :placeholder="$t('notifications.destination-form.webhook-url-placeholder')"
        />
        <p v-if="webhookUrl.trim() && !isValidUrl" class="text-error mt-1 text-xs">
          {{ $t("notifications.destination-form.webhook-url-invalid") }}
        </p>
      </section>

      <!-- 3. Payload -->
      <section>
        <FormStepHeading :step="3" :title="$t('notifications.destination-form.payload-format')" />
        <div class="mb-2 flex flex-wrap gap-2">
          <button
            v-for="format in FORMATS"
            :key="format"
            type="button"
            class="btn btn-sm"
            :class="payloadFormat === format ? 'btn-primary' : 'btn-ghost'"
            @click="selectPayloadFormat(format)"
          >
            {{ $t(`notifications.destination-form.format-${format}`) }}
          </button>
        </div>

        <div v-if="confirmingFormat" class="alert alert-warning mb-2 flex-wrap py-2 text-sm">
          <mdi:alert-outline />
          <span class="flex-1">{{ $t("notifications.destination-form.format-replace-warning") }}</span>
          <button class="btn btn-xs" @click="confirmingFormat = null">
            {{ $t("notifications.destination-form.cancel") }}
          </button>
          <button class="btn btn-xs btn-warning" @click="applyPayloadFormat(confirmingFormat)">
            {{ $t("notifications.destination-form.format-replace-confirm") }}
          </button>
        </div>

        <div
          ref="templateEditorRef"
          class="border-base-content/20 focus-within:border-primary min-h-48 w-full overflow-auto rounded-lg border"
        ></div>

        <!-- The backend renders valid JSON field-by-field and anything else as raw text, so say
             which one this template lands in instead of leaving it to a failed send. -->
        <p class="mt-1 text-xs" :class="payloadNoteClass">
          <mdi:alert v-if="mode === 'unbalanced'" class="inline" />
          <mdi:check v-else-if="mode === 'json'" class="inline" />
          <mdi:information-outline v-else class="inline" />
          {{ $t(`notifications.destination-form.payload-mode-${mode}`) }}
        </p>

        <TemplateVariables @insert="insertSnippet" />
      </section>

      <!-- 4. Custom Headers -->
      <section>
        <FormStepHeading :step="4" :title="$t('notifications.destination-form.headers')" />
        <p class="text-base-content/60 mb-2 text-sm">{{ $t("notifications.destination-form.headers-hint") }}</p>
        <div class="space-y-2">
          <div v-for="(header, index) in headers" :key="header.key" class="flex items-center gap-2">
            <input
              v-model="header.name"
              type="text"
              class="input focus:input-primary flex-1 text-base"
              :placeholder="$t('notifications.destination-form.header-name')"
            />
            <input
              v-model="header.value"
              type="text"
              class="input focus:input-primary flex-1 text-base"
              :placeholder="$t('notifications.destination-form.header-value')"
            />
            <button
              type="button"
              class="btn btn-ghost btn-sm btn-square"
              :aria-label="$t('notifications.destination-form.remove-header')"
              @click="headers.splice(index, 1)"
            >
              <carbon:close />
            </button>
          </div>
          <button
            type="button"
            class="btn btn-ghost btn-sm"
            @click="headers.push({ name: '', value: '', key: headerKeyCounter++ })"
          >
            <carbon:add />
            {{ $t("notifications.destination-form.add-header") }}
          </button>
        </div>
      </section>
    </div>

    <!-- Actions -->
    <!-- Opaque and full-bleed: the parent's padding would otherwise leave the scrolling content
         visible down both sides of the bar. -->
    <div class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 -mx-4 mt-auto border-t px-4 py-4">
      <div v-if="error" class="alert alert-error mb-3 py-2 text-sm">
        <span>{{ error }}</span>
      </div>

      <!-- Cleared whenever the request changes, so a green tick always describes what is on screen -->
      <div
        v-if="testResult"
        class="alert mb-3 py-2 text-sm"
        :class="testResult.success ? 'alert-success' : 'alert-error'"
      >
        <span v-if="testResult.success">
          {{ $t("notifications.destination-form.test-success") }}
          <span v-if="testResult.statusCode" class="opacity-70">({{ testResult.statusCode }})</span>
        </span>
        <span v-else>{{ testResult.error }}</span>
      </div>

      <div v-if="confirmingDiscard" class="flex flex-wrap items-center justify-end gap-2">
        <span class="mr-auto text-sm">{{ $t("notifications.destination-form.discard-title") }}</span>
        <button class="btn btn-sm" @click="confirmingDiscard = false">
          {{ $t("notifications.destination-form.discard-keep") }}
        </button>
        <button class="btn btn-sm btn-error" @click="discard">
          {{ $t("notifications.destination-form.discard-confirm") }}
        </button>
      </div>

      <div v-else class="flex flex-wrap items-center gap-2">
        <button class="btn" @click="testDestination" :disabled="!isValidUrl || isTesting">
          <span v-if="isTesting" class="loading loading-spinner loading-sm"></span>
          {{ $t("notifications.destination-form.test") }}
        </button>
        <span v-if="blockers.length" class="text-base-content/60 text-sm">
          {{ $t("notifications.destination-form.still-needed", { fields: blockerLabels }) }}
        </span>
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
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, TestWebhookResult } from "@/types/notifications";
import { createTemplateEditor, payloadMode } from "@/composable/templateEditor";
import { PAYLOAD_TEMPLATES, type PayloadFormat } from "./payloadTemplates";
import FormStepHeading from "./FormStepHeading.vue";
import TemplateVariables from "./TemplateVariables.vue";

const props = defineProps<{
  close?: () => void;
  onCreated?: (created?: Dispatcher) => void;
  destination?: Dispatcher;
  existingDispatchers?: Dispatcher[];
  isEditing: boolean;
}>();

const { t } = useI18n();

const FORMATS = ["slack", "discord", "ntfy", "custom"] as const;

const nameInput = ref<HTMLInputElement>();
const templateEditorRef = ref<HTMLElement>();
const name = ref(props.destination?.name ?? "");
useFocus(nameInput, { initialValue: true });
const webhookUrl = ref(props.destination?.url ?? "");
const payloadFormat = ref<PayloadFormat>(props.isEditing ? "custom" : "slack");
const template = ref(props.isEditing ? (props.destination?.template ?? "") : PAYLOAD_TEMPLATES[payloadFormat.value]);
let headerKeyCounter = 0;
const headers = ref<{ name: string; value: string; key: number }[]>(
  props.destination?.headers
    ? Object.entries(props.destination.headers).map(([name, value]) => ({ name, value, key: headerKeyCounter++ }))
    : [],
);
const isTesting = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);
const testResult = ref<TestWebhookResult | null>(null);
const confirmingFormat = ref<PayloadFormat | null>(null);

let templateEditorView: Awaited<ReturnType<typeof createTemplateEditor>> | undefined;

const mode = computed(() => payloadMode(template.value));
const payloadNoteClass = computed(() =>
  mode.value === "unbalanced" ? "text-error" : mode.value === "json" ? "text-success" : "text-base-content/60",
);

// Two destinations with the same name are indistinguishable in the alert form's picker.
const isDuplicateName = computed(() => {
  const trimmed = name.value.trim().toLowerCase();
  if (!trimmed) return false;
  return (props.existingDispatchers ?? []).some(
    (d) => d.id !== props.destination?.id && d.name.trim().toLowerCase() === trimmed,
  );
});

function selectPayloadFormat(format: PayloadFormat) {
  if (format === payloadFormat.value) return;
  // Picking a preset overwrites whatever is in the editor, which is fine for an untouched
  // preset and destructive for a template the user has actually written.
  const isPreset = Object.values(PAYLOAD_TEMPLATES).includes(template.value.trim());
  if (template.value.trim() && !isPreset) {
    confirmingFormat.value = format;
    return;
  }
  applyPayloadFormat(format);
}

function applyPayloadFormat(format: PayloadFormat) {
  confirmingFormat.value = null;
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

function insertSnippet(snippet: string) {
  if (!templateEditorView) return;
  templateEditorView.dispatch(templateEditorView.state.replaceSelection(snippet));
  templateEditorView.focus();
}

onMounted(async () => {
  if (!templateEditorRef.value) return;
  templateEditorView = await createTemplateEditor({
    parent: templateEditorRef.value,
    initialValue: template.value,
    placeholder: t("notifications.destination-form.template-placeholder"),
    onChange: (v) => (template.value = v),
  });
});

onScopeDispose(() => {
  templateEditorView?.destroy();
});

function headersToRecord(): Record<string, string> | undefined {
  const filtered = headers.value.filter((h) => h.name.trim() && h.value.trim());
  if (filtered.length === 0) return undefined;
  return Object.fromEntries(filtered.map((h) => [h.name.trim(), h.value.trim()]));
}

const isValidUrl = computed(() => {
  const trimmed = webhookUrl.value.trim();
  if (!trimmed) return false;
  try {
    const url = new URL(trimmed);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
});

const blockers = computed(() => {
  const missing: string[] = [];
  if (!name.value.trim()) missing.push("name");
  if (!isValidUrl.value) missing.push("url");
  if (mode.value === "unbalanced") missing.push("template");
  return missing;
});
const blockerLabels = computed(() =>
  blockers.value.map((b) => t(`notifications.destination-form.missing.${b}`)).join(", "),
);

const canSave = computed(() => !isSaving.value && blockers.value.length === 0);

// A result that predates the current URL, template or headers is worse than none: it reads as
// a green light for a request that was never actually sent.
watch([webhookUrl, template, headers], () => (testResult.value = null), { deep: true });

async function testDestination() {
  if (!isValidUrl.value) return;

  isTesting.value = true;
  testResult.value = null;

  try {
    const res = await fetch(withBase("/api/notifications/test-webhook"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url: webhookUrl.value.trim(),
        template: template.value.trim() || undefined,
        headers: headersToRecord(),
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
      headers: headersToRecord(),
    };

    const url = props.isEditing
      ? withBase(`/api/notifications/dispatchers/${props.destination!.id}`)
      : withBase("/api/notifications/dispatchers");

    const res = await fetch(url, {
      method: props.isEditing ? "PUT" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    });

    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || "Failed to save destination");
    }

    registerGuard(null);
    // Hand the saved destination back so callers can select it right away.
    props.onCreated?.(await res.json().catch(() => undefined));
    props.close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to save destination";
  } finally {
    isSaving.value = false;
  }
}

// Unsaved-changes guard, matching the alert form: a stray Esc should not throw away a template.
const registerGuard = useDrawerCloseGuard();
const confirmingDiscard = ref(false);
const initial = ref<string>();
const snapshot = computed(() =>
  JSON.stringify({
    name: name.value,
    webhookUrl: webhookUrl.value,
    template: template.value,
    headers: headers.value.map((h) => [h.name, h.value]),
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
</script>
