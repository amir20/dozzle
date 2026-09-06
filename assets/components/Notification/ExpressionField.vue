<template>
  <div>
    <p v-if="hint" class="text-base-content/60 mb-1.5 text-sm">{{ hint }}</p>
    <div
      class="input focus-within:input-primary h-auto w-full py-1 focus-within:z-50"
      :class="modelValue.trim() && !error ? 'input-primary' : { 'input-error!': error }"
    >
      <div ref="editorRef" class="w-full"></div>
    </div>

    <p v-if="error" class="text-error mt-1 font-mono text-xs">{{ error }}</p>

    <!-- Examples double as documentation: clicking one fills the editor. -->
    <div v-if="examples.length" class="mt-2 flex flex-wrap items-center gap-1.5">
      <span class="text-base-content/50 text-xs">{{ $t("notifications.alert-form.examples") }}</span>
      <button
        v-for="example in examples"
        :key="example"
        type="button"
        class="border-base-content/15 bg-base-content/5 text-base-content/70 hover:border-primary hover:text-primary cursor-pointer rounded-md border px-2 py-0.5 font-mono text-xs transition-colors"
        :title="$t('notifications.alert-form.use-example')"
        @click="setValue(example)"
      >
        {{ example }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Completion } from "@codemirror/autocomplete";

const props = defineProps<{
  hint?: string;
  placeholder: string;
  error?: string | null;
  examples?: string[];
  getHints: () => Completion[];
}>();

const examples = computed(() => props.examples ?? []);
const modelValue = defineModel<string>({ required: true });

const editorRef = ref<HTMLElement>();
const { setValue } = useExprEditorField(editorRef, {
  placeholder: props.placeholder,
  // Read once: the editor owns its content after mount, and re-seeding it on every model
  // change would fight the user's cursor.
  initialValue: modelValue.value,
  getHints: () => props.getHints(),
  onChange: (v) => (modelValue.value = v),
});
</script>
