<template>
  <!--
    What the assistant can see, said quietly.

    If it silently knows what is on your screen you cannot tell what it knows,
    and one wrong assumption makes you distrust all of it. This is a footnote
    rather than a row of pills: it has to be legible at a glance and invisible
    the rest of the time, and a line of uppercase chips above the composer reads
    as status the reader is meant to act on.
  -->
  <div class="text-base-content/40 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs">
    <mdi:eye-outline class="size-3.5 shrink-0" />
    <template v-for="(part, i) in parts" :key="part.key">
      <span v-if="i" aria-hidden="true">·</span>
      <span :class="part.name ? 'text-base-content/70 truncate font-mono' : ''">{{ part.label }}</span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { ViewContext } from "@/composable/viewContext";

const { view } = defineProps<{ view: ViewContext }>();
const { t } = useI18n();

const parts = computed(() => {
  const out: { key: string; label: string; name?: boolean }[] = [];
  const names = view.containers.map((c) => c.name);

  if (names.length === 1) {
    out.push({ key: "container", label: names[0], name: true });
  } else if (names.length > 1) {
    out.push({ key: "container", label: t("cloud-chat.n-containers", { n: names.length }) });
  } else if (view.target) {
    out.push({ key: "target", label: view.target, name: true });
  }

  if (view.lines?.length) out.push({ key: "lines", label: t("cloud-chat.lines-in-view", { n: view.lines.length }) });
  if (view.search) out.push({ key: "search", label: `"${view.search}"` });
  if (view.levels?.length) out.push({ key: "levels", label: view.levels.join(" / ") });
  if (view.historical) out.push({ key: "historical", label: t("cloud-chat.historical") });

  return out;
});
</script>
