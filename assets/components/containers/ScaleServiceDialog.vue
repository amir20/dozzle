<template>
  <dialog ref="dialog" class="modal max-md:modal-bottom">
    <form class="modal-box flex w-full max-w-sm flex-col p-0" @submit.prevent="submit">
      <div class="flex flex-col gap-1 px-6 pt-6">
        <h2 class="text-2xl font-bold">{{ $t("toolbar.scale-service") }}</h2>
        <p class="text-base-content/60 truncate font-mono text-sm">{{ service.name }}</p>
      </div>
      <!-- Same shape as the replicated case, so the dialog reads as the same
           action, just one with nothing to set. -->
      <InlineNotice v-if="global" type="info" class="mx-6 my-4" data-testid="scale-global">
        {{ $t("toolbar.scale-global") }}
      </InlineNotice>
      <label
        v-else
        class="border-base-content/15 bg-base-200/40 mx-6 my-4 flex items-center gap-4 rounded-lg border p-4"
      >
        <span class="flex flex-col text-sm">
          {{ $t("toolbar.scale-replicas") }}
          <span class="text-base-content/40 text-xs">{{ $t("toolbar.scale-running", { count: running }) }}</span>
        </span>
        <input
          ref="input"
          v-model.number="replicas"
          type="number"
          min="1"
          step="1"
          required
          class="input input-sm ml-auto w-24 text-right font-mono"
          data-testid="scale-replicas"
        />
      </label>
      <div class="border-base-content/10 flex justify-end gap-2 border-t px-6 py-4">
        <button type="button" class="btn btn-sm" @click="dialog?.close()">
          {{ global ? $t("button.close") : $t("button.cancel") }}
        </button>
        <button v-if="!global" type="submit" class="btn btn-primary btn-sm" :disabled="scaling || !valid">
          <span v-if="scaling" class="loading loading-spinner loading-xs"></span>
          {{ $t("toolbar.scale-service") }}
        </button>
      </div>
    </form>
    <form method="dialog" class="modal-backdrop">
      <button>{{ $t("button.cancel") }}</button>
    </form>
  </dialog>
</template>

<script lang="ts" setup>
import type { Service } from "@/models/Stack";

const { service, global = false } = defineProps<{ service: Service; global?: boolean }>();

const dialog = useTemplateRef<HTMLDialogElement>("dialog");
const input = useTemplateRef<HTMLInputElement>("input");
const { scaling, scaleService } = useScaleService();

// What the user sees running is the best guess at the replica count without
// asking a manager, and in a settled service it is the same number.
const running = computed(() => service.containers.filter((c) => c.state === "running").length);
const replicas = ref(1);
const valid = computed(() => Number.isInteger(replicas.value) && replicas.value >= 1);

function open() {
  replicas.value = Math.max(running.value, 1);
  dialog.value?.showModal();
  if (!global) nextTick(() => input.value?.select());
}

async function submit() {
  if (global || !valid.value) return;
  if (await scaleService(service.name, replicas.value)) dialog.value?.close();
}

defineExpose({ open });
</script>
