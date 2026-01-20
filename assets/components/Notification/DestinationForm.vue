<template>
  <div class="space-y-6 p-4">
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

    <!-- Error -->
    <div v-if="error" class="alert alert-error">
      <span>{{ error }}</span>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2 pt-4">
      <button class="btn" @click="testDestination" :disabled="!canTest || isTesting">
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
import { CreateDispatcherDocument, UpdateDispatcherDocument, type Dispatcher } from "@/types/graphql";

const { close, onCreated, destination } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
  destination?: Dispatcher;
}>();

const createMutation = useMutation(CreateDispatcherDocument);
const updateMutation = useMutation(UpdateDispatcherDocument);

const isEditing = computed(() => !!destination);

const nameInput = ref<HTMLInputElement>();
const name = ref(destination?.name ?? "");
useFocus(nameInput, { initialValue: true });
const type = ref<"webhook" | "cloud">((destination?.type as "webhook" | "cloud") ?? "webhook");
const webhookUrl = ref(destination?.url ?? "");
const isTesting = ref(false);
const isSaving = ref(false);
const error = ref<string | null>(null);

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
  isTesting.value = true;
  // TODO: Implement actual test when backend is ready
  await new Promise((resolve) => setTimeout(resolve, 1000));
  isTesting.value = false;
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
    };

    const result = isEditing.value
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
