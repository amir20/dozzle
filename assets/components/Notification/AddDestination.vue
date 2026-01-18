<template>
  <div class="space-y-6 p-4">
    <div class="mb-6">
      <h2 class="text-2xl font-bold">Add Destination</h2>
      <p class="text-base-content/60">Where should notifications be sent?</p>
    </div>

    <!-- Name -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Name</legend>
      <input v-model="name" type="text" class="input focus:input-primary w-full" placeholder="e.g., Production Slack" />
    </fieldset>

    <!-- Type Selection -->
    <fieldset class="fieldset">
      <legend class="fieldset-legend text-lg">Type</legend>
      <div class="space-y-3">
        <label
          class="card card-border 20 cursor-pointer transition-colors"
          :class="type === 'webhook' ? 'border-primary bg-primary/10' : ''"
        >
          <div class="card-body flex-row items-center gap-3 p-4">
            <input type="radio" v-model="type" value="webhook" class="radio radio-primary" />
            <div>
              <div class="font-semibold">HTTP Webhook</div>
              <div class="text-base-content/60 text-sm">Slack, Discord, custom endpoint</div>
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
              <div class="font-semibold">Dozzle Cloud</div>
              <div class="text-base-content/60 text-sm">Push, email, and dashboard</div>
            </div>
          </div>
        </label>
      </div>
    </fieldset>

    <!-- Webhook URL (only for webhook type) -->
    <fieldset v-if="type === 'webhook'" class="fieldset">
      <legend class="fieldset-legend text-lg">Webhook URL</legend>
      <input
        v-model="webhookUrl"
        type="url"
        class="input focus:input-primary w-full"
        placeholder="https://hooks.foo.com/services/..."
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
        Test
      </button>
      <div class="flex-1"></div>
      <button class="btn" @click="close?.()">Cancel</button>
      <button class="btn btn-primary" :disabled="!canAdd" @click="addDestination">
        <span v-if="isAdding" class="loading loading-spinner loading-sm"></span>
        Add Destination
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { close, onCreated } = defineProps<{
  close?: () => void;
  onCreated?: () => void;
}>();

const name = ref("");
const type = ref<"webhook" | "cloud">("webhook");
const webhookUrl = ref("");
const isTesting = ref(false);
const isAdding = ref(false);
const error = ref<string | null>(null);

const canTest = computed(() => {
  if (type.value === "webhook") {
    return webhookUrl.value.trim().length > 0;
  }
  return false;
});

const canAdd = computed(() => {
  if (isAdding.value) return false;
  if (!name.value.trim()) return false;
  if (type.value === "webhook" && !webhookUrl.value.trim()) return false;
  return true;
});

async function testDestination() {
  isTesting.value = true;
  // TODO: Implement actual test when backend is ready
  await new Promise((resolve) => setTimeout(resolve, 1000));
  isTesting.value = false;
}

async function addDestination() {
  if (!canAdd.value) return;

  isAdding.value = true;
  error.value = null;

  try {
    const response = await fetch(withBase("/api/notifications/dispatchers"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: name.value.trim(),
        type: type.value,
        url: type.value === "webhook" ? webhookUrl.value.trim() : undefined,
      }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `HTTP ${response.status}`);
    }

    onCreated?.();
    close?.();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to add destination";
  } finally {
    isAdding.value = false;
  }
}
</script>
