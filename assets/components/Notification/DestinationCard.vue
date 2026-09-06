<template>
  <div
    class="card bg-base-100 border"
    :class="confirmingDelete ? 'border-error/50' : 'hover:border-primary border-transparent'"
  >
    <!-- Confirming replaces the card rather than stacking a banner under it: the card is only
         288px wide, so a filled alert with two buttons wrapped onto three ragged lines. -->
    <div v-if="confirmingDelete" class="card-body gap-3 p-4">
      <div class="flex items-start gap-3">
        <mdi:alert-outline class="text-error mt-0.5 shrink-0 text-lg" />
        <div class="min-w-0 flex-1">
          <h4 class="font-semibold">{{ $t("notifications.destination.delete-warning") }}</h4>
          <p class="text-base-content/60 mt-1 text-sm">
            <!-- Deleting orphans every alert pointing here, so name the cost before doing it. -->
            {{
              usedByCount
                ? $t("notifications.destination.delete-warning-used", { count: usedByCount })
                : $t("notifications.destination.unused")
            }}
          </p>
        </div>
      </div>
      <div class="flex justify-end gap-2">
        <button class="btn btn-sm" :disabled="isDeleting" @click="confirmingDelete = false">
          {{ $t("notifications.destination.delete-cancel") }}
        </button>
        <button class="btn btn-sm btn-error" :disabled="isDeleting" @click="deleteDestination">
          <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
          {{ $t("notifications.destination.delete-confirm") }}
        </button>
      </div>
    </div>

    <div v-else class="card-body gap-2 p-4">
      <div class="flex items-start gap-3">
        <button
          type="button"
          class="flex flex-1 cursor-pointer items-start gap-3 text-left"
          :aria-label="$t('notifications.destination.edit')"
          @click="editDestination"
        >
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
            <mdi:webhook v-if="destination.type === 'webhook'" class="text-lg" />
            <mdi:cloud v-else class="text-primary text-lg" />
          </div>
          <div class="min-w-0 flex-1">
            <h4 class="truncate font-semibold">{{ destination.name }}</h4>
            <p class="text-base-content/60 text-sm">
              {{
                destination.type === "webhook"
                  ? $t("notifications.destination.http-webhook")
                  : $t("notifications.destination.dozzle-cloud")
              }}
            </p>
            <p class="text-base-content/50 mt-1 text-xs">
              {{
                usedByCount
                  ? $t("notifications.destination.used-by", { count: usedByCount })
                  : $t("notifications.destination.unused")
              }}
            </p>
          </div>
        </button>
        <div class="dropdown dropdown-end" @click.stop>
          <div tabindex="0" role="button" class="btn btn-ghost btn-sm btn-square">
            <ion:ellipsis-vertical />
          </div>
          <ul
            tabindex="0"
            class="menu dropdown-content rounded-box bg-base-100 border-base-content/20 z-50 w-40 border p-1 shadow-sm"
          >
            <li>
              <a @click="runFromMenu(editDestination)">{{ $t("notifications.destination.edit") }}</a>
            </li>
            <li v-if="destination.type !== 'cloud'">
              <a @click="runFromMenu(duplicateDestination)">{{ $t("notifications.destination.duplicate") }}</a>
            </li>
            <li v-if="destination.type !== 'cloud'">
              <a class="text-error" @click="runFromMenu(() => (confirmingDelete = true))">
                {{ $t("notifications.destination.delete") }}
              </a>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher } from "@/types/notifications";
import DestinationForm from "./DestinationForm.vue";

const { destination, onUpdated, existingDispatchers, usedByCount } = defineProps<{
  destination: Dispatcher;
  onUpdated?: () => void;
  existingDispatchers: Dispatcher[];
  /** How many alerts send here. Drives the delete warning. */
  usedByCount: number;
}>();

const { t } = useI18n();
const { showToast } = useToast();
const showDrawer = useDrawer();

const confirmingDelete = ref(false);
const isDeleting = ref(false);

/**
 * A CSS-only daisyUI dropdown stays open until it loses focus, so picking an item left the menu
 * sitting on top of whatever the item revealed.
 */
function runFromMenu(action: () => void) {
  if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
  action();
}

function editDestination() {
  showDrawer(
    DestinationForm,
    {
      destination,
      onCreated: onUpdated,
      existingDispatchers,
    },
    "md",
  );
}

/** Appends a counter so duplicating twice doesn't produce two identically named copies. */
function copyName() {
  const taken = new Set(existingDispatchers.map((d) => d.name));
  const base = t("notifications.destination.copy-of", { name: destination.name });
  if (!taken.has(base)) return base;
  for (let i = 2; ; i++) {
    const candidate = `${base} ${i}`;
    if (!taken.has(candidate)) return candidate;
  }
}

async function duplicateDestination() {
  try {
    const res = await fetch(withBase("/api/notifications/dispatchers"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: copyName(),
        type: destination.type,
        url: destination.url,
        template: destination.template,
        headers: destination.headers,
      }),
    });
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error ?? res.statusText);
    onUpdated?.();
  } catch (e) {
    showToast({ type: "error", message: e instanceof Error ? e.message : t("notifications.destination.copy-failed") });
  }
}

async function deleteDestination() {
  isDeleting.value = true;
  try {
    const res = await fetch(withBase(`/api/notifications/dispatchers/${destination.id}`), { method: "DELETE" });
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error ?? res.statusText);
    confirmingDelete.value = false;
    onUpdated?.();
  } catch (e) {
    showToast({
      type: "error",
      message: e instanceof Error ? e.message : t("notifications.destination.delete-failed"),
    });
  } finally {
    isDeleting.value = false;
  }
}
</script>
