<template>
  <div class="bg-base-100 flex h-screen min-h-0 flex-col">
    <div class="border-base-content/10 flex items-center gap-2 border-b px-4 py-3">
      <mdi:message-outline class="text-base-content/60 size-4 shrink-0" />
      <span class="text-sm font-semibold">{{ $t("cloud-chat.title") }}</span>
      <button
        type="button"
        class="btn btn-ghost btn-xs btn-square ml-auto"
        :aria-label="$t('cloud-chat.close')"
        @click="closePane()"
      >
        <mdi:close class="size-4" />
      </button>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
      <p v-if="!messages.length" class="text-base-content/60 text-sm">
        {{ $t("cloud-chat.empty") }}
      </p>

      <div v-for="(message, i) in messages" :key="i" class="mb-4">
        <template v-if="message.role === 'user'">
          <ChatContextChips v-if="message.view" :view="message.view" class="mb-1.5" />
          <div class="border-primary/40 bg-primary/5 rounded-box border px-3 py-2 text-sm">
            {{ message.text }}
          </div>
        </template>
        <div
          v-else
          class="rounded-box border-base-content/10 bg-base-200 border px-3 py-2 text-sm whitespace-pre-wrap"
          :class="{ 'text-error': message.error }"
        >
          {{ message.text || status || $t("cloud-chat.thinking") }}
        </div>
      </div>
    </div>

    <!-- Sticky, opaque and full bleed, like every other footer in the app. -->
    <div class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 border-t px-4 py-3">
      <ChatContextChips :view="view" class="mb-2" />
      <form class="flex items-center gap-2" @submit.prevent="send">
        <input
          v-model="draft"
          type="text"
          class="input input-sm flex-1"
          :placeholder="$t('cloud-chat.placeholder')"
          :disabled="streaming"
        />
        <button type="submit" class="btn btn-primary btn-sm btn-square" :disabled="streaming || !draft.trim()">
          <mdi:send class="size-4" />
        </button>
      </form>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { messages, status, streaming, ask, closePane } = useCloudChat();
const view = useViewContext();
const draft = ref("");

function send() {
  const text = draft.value;
  draft.value = "";
  ask(text, view.value);
}
</script>
