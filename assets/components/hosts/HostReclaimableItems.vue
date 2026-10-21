<template>
  <div class="border-base-content/10 border-t px-2 py-1.5 text-xs">
    <div v-if="preview.status === 'loading'" class="flex justify-center py-1">
      <span class="loading loading-spinner size-3.5 opacity-60"></span>
    </div>
    <p v-else-if="preview.status === 'error'" class="text-base-content/60 py-1">{{ preview.error }}</p>
    <template v-else-if="preview.status === 'ready'">
      <!-- Rows are 20px, so 160px rests on a row boundary instead of halving one. -->
      <ul class="max-h-40 overflow-y-auto">
        <li v-for="item in list" :key="item.key" class="flex items-center gap-2 py-0.5">
          <span class="min-w-0 flex-1 truncate font-mono" :title="item.title">{{ item.label }}</span>
          <span v-if="item.aside" class="text-base-content/40 shrink-0">{{ item.aside }}</span>
          <span class="w-16 shrink-0 text-right font-mono">{{ item.size }}</span>
        </li>
      </ul>
      <p class="text-base-content/40 border-base-content/10 mt-1.5 border-t pt-1.5">{{ $t("reclaim.preview-note") }}</p>
    </template>
  </div>
</template>

<script lang="ts" setup>
import type { ReclaimablePreview } from "@/stores/hosts";

const { kind, preview } = defineProps<{ kind: "images" | "volumes"; preview: ReclaimablePreview }>();

const { locale } = useI18n();

// Docker names an anonymous volume and identifies an image by a 64-character hash.
const short = (id: string) => id.replace(/^sha256:/, "").slice(0, 12);
const isHash = (name: string) => /^[0-9a-f]{64}$/.test(name);
const size = (bytes: number) => (bytes < 0 ? "—" : formatBytes(bytes, { decimals: 1 }));

const list = computed(() => {
  if (preview.status !== "ready") return [];
  if (kind === "images") {
    return preview.items.images.map((img) => ({
      key: img.id,
      label: img.tags[0] ?? `<none> ${short(img.id)}`,
      title: [...img.tags, short(img.id)].join("\n"),
      aside: img.tags.length > 1 ? `+${img.tags.length - 1}` : "",
      size: size(img.size),
    }));
  }
  return preview.items.volumes.map((v) => {
    const created = new Date(v.created);
    return {
      key: v.name,
      label: isHash(v.name) ? short(v.name) : v.name,
      title: v.name,
      aside:
        created.getFullYear() > 1
          ? created.toLocaleDateString(locale.value, { month: "short", day: "numeric", year: "numeric" })
          : "",
      size: size(v.size),
    };
  });
});
</script>
