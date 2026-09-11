<template>
  <div class="bg-base-100 flex h-full min-h-0 flex-col">
    <div ref="scroller" class="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-3" @scroll="onScroll">
      <!-- Everything hugs the composer, empty or not. Centred, the openers
           floated in the middle of a tall empty column, half a screen from the
           box they type into. -->
      <div v-if="!messages.length" class="mt-auto">
        <p class="text-base-content/60 mb-2 text-sm">{{ $t("cloud-chat.empty") }}</p>
        <button
          v-for="suggestion in suggestions"
          :key="suggestion"
          type="button"
          class="hover:bg-info/10 hover:text-info flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors"
          @click="ask(suggestion, view)"
        >
          <mdi:arrow-top-right class="text-info/60 size-4 shrink-0" />
          <span class="min-w-0 flex-1">{{ suggestion }}</span>
        </button>
      </div>

      <div v-else class="mt-auto space-y-4">
        <div v-for="(message, i) in messages" :key="i">
          <!-- The question is a panel and the answer is the pane itself. Two
               facing bubbles in a 550px column spend half the width on borders
               that say nothing the alignment does not. What the two turns do get
               is a colour each, because in a wall of grey text the only thing a
               reader ever has to find again is where their own question ended:
               a primary wash on the question, a cloud glyph on the answer. -->
          <template v-if="message.role === 'user'">
            <div class="border-primary/25 bg-primary/5 rounded-lg border p-3">
              <ChatFocusedLine v-if="message.view?.focused" :line="message.view.focused" class="mb-2" />
              <p class="text-sm whitespace-pre-wrap">{{ message.text }}</p>
              <ChatViewContext v-if="message.view" :view="message.view" compact class="mt-2" />
            </div>
          </template>
          <template v-else>
            <div class="flex gap-2">
              <span
                class="bg-info/10 text-info mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full"
                aria-hidden="true"
              >
                <mdi:cloud class="size-3.5" />
              </span>
              <div class="min-w-0 flex-1">
                <InlineNotice v-if="message.error" type="error">{{ message.text }}</InlineNotice>
                <ChatMarkdown v-else-if="message.text" :text="message.text" class="text-sm leading-relaxed" />
                <div v-else class="text-base-content/60 flex items-center gap-2 text-sm">
                  <span class="loading loading-dots loading-xs"></span>
                  {{ working }}
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- Sticky, opaque and full bleed, like every other footer in the app. -->
    <div class="bg-base-100 border-base-content/10 sticky bottom-0 z-10 shrink-0 border-t px-4 py-3">
      <ChatViewContext :view="view" class="mb-2">
        <ChatFocusedLine v-if="focused" :line="focused" removable @remove="clearFocus()" />
      </ChatViewContext>
      <!-- Full height rather than input-sm: this is the one field on the
           surface and the thing people came here to type in. -->
      <form class="input flex h-12 w-full items-center gap-1 pr-1" @submit.prevent="send">
        <input
          ref="composer"
          v-model="draft"
          type="text"
          class="input input-ghost grow px-0 text-sm"
          :placeholder="$t('cloud-chat.placeholder')"
          :disabled="streaming"
        />
        <button
          type="submit"
          class="btn btn-primary btn-circle btn-sm"
          :disabled="streaming || !draft.trim()"
          :aria-label="$t('cloud-chat.send')"
        >
          <mdi:send class="size-4" />
        </button>
      </form>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { messages, status, activity, streaming, focused, ask, clearFocus } = useCloudChat();
const view = useViewContext();
const draft = ref("");
const { t } = useI18n();

// Three openers, one per thing the assistant is actually good at from here, so
// the first question cannot miss. Each is sent with the same view context a
// typed question would carry, which is why none of them name a container.
const suggestions = computed(() => [
  t("cloud-chat.suggest-summarize"),
  t("cloud-chat.suggest-errors"),
  t("cloud-chat.suggest-next"),
]);

// Dozzle names what it is doing with a token, so the phrase is chosen here in
// the reader's language; cloud sends prose, which can only be shown as it came.
const working = computed(() => {
  if (activity.value) return t(`cloud-chat.doing.${activity.value}`);
  return status.value || t("cloud-chat.thinking");
});

const scroller = useTemplateRef<HTMLElement>("scroller");
const composer = useTemplateRef<HTMLInputElement>("composer");
// Follow the tail while the reader is at the bottom, and stop the moment they
// scroll up to re-read something: a streaming answer that keeps yanking the
// view back down is worse than one that waits.
const following = ref(true);

function onScroll() {
  const el = scroller.value;
  if (el) following.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
}

watch(
  [messages, status, activity],
  async () => {
    if (!following.value) return;
    await nextTick();
    scroller.value?.scrollTo({ top: scroller.value.scrollHeight });
  },
  { deep: true },
);

// Arriving from a log row's menu means the question is already half asked.
onMounted(() => composer.value?.focus());
watch(focused, (line) => line && composer.value?.focus());

function send() {
  const text = draft.value;
  draft.value = "";
  following.value = true;
  ask(text, view.value);
}
</script>
