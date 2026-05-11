<template>
  <div class="card bg-base-100 hover:border-primary cursor-pointer border border-transparent" @click="editDestination">
    <div class="card-body gap-2 p-4">
      <div class="flex items-start gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-lg">
          <mdi:webhook v-if="destination.type === 'webhook'" class="text-lg" />
          <mdi:cloud v-else class="text-primary-content text-lg" />
        </div>
        <div class="flex-1">
          <h4 class="font-semibold">{{ destination.name }}</h4>
          <p class="text-base-content/60 text-sm">
            {{
              destination.type === "webhook"
                ? $t("notifications.destination.http-webhook")
                : $t("notifications.destination.dozzle-cloud")
            }}
          </p>
        </div>
        <div class="dropdown dropdown-end" @click.stop>
          <label tabindex="0" class="btn btn-ghost btn-sm btn-square">
            <ion:ellipsis-vertical />
          </label>
          <ul
            tabindex="0"
            class="menu dropdown-content rounded-box bg-base-100 border-base-content/20 z-50 w-40 border p-1 shadow-sm"
          >
            <li>
              <a @click="editDestination">{{ $t("notifications.destination.edit") }}</a>
            </li>
            <li v-if="destination.type !== 'cloud'">
              <a @click="duplicateDestination">{{ $t("notifications.destination.duplicate") }}</a>
            </li>
            <li v-if="destination.type !== 'cloud'">
              <a class="text-error" @click="deleteDestination">{{ $t("notifications.destination.delete") }}</a>
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

const { destination, onUpdated, existingDispatchers } = defineProps<{
  destination: Dispatcher;
  onUpdated?: () => void;
  existingDispatchers: Dispatcher[];
}>();

const showDrawer = useDrawer();

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

async function duplicateDestination() {
  await fetch(withBase("/api/notifications/dispatchers"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      name: `Copy of ${destination.name}`,
      type: destination.type,
      url: destination.url,
      template: destination.template,
      headers: destination.headers,
    }),
  });
  onUpdated?.();
}

async function deleteDestination() {
  await fetch(withBase(`/api/notifications/dispatchers/${destination.id}`), { method: "DELETE" });
  onUpdated?.();
}
</script>
