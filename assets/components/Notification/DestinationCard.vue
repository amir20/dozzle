<template>
  <div class="card bg-base-100 w-72">
    <div class="card-body gap-2 p-4">
      <div class="flex items-start gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-lg">
          <mdi:webhook v-if="destination.type === 'webhook'" class="text-lg" />
          <mdi:cloud v-else class="text-primary-content text-lg" />
        </div>
        <div class="flex-1">
          <h4 class="font-semibold">{{ destination.name }}</h4>
          <p class="text-base-content/60 text-sm">
            {{ destination.type === "webhook" ? "HTTP Webhook" : "Dozzle Cloud" }}
          </p>
        </div>
        <div class="dropdown dropdown-end">
          <label tabindex="0" class="btn btn-ghost btn-sm btn-square">
            <ion:ellipsis-vertical />
          </label>
          <ul
            tabindex="0"
            class="menu dropdown-content rounded-box bg-base-100 border-base-content/20 z-50 w-40 border p-1 shadow-sm"
          >
            <li><a @click="$emit('edit', destination)">Edit</a></li>
            <li><a class="text-error" @click="$emit('delete', destination)">Delete</a></li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
export interface Destination {
  id: number;
  name: string;
  type: "webhook" | "cloud";
  url?: string;
}

defineProps<{
  destination: Destination;
}>();

defineEmits<{
  edit: [destination: Destination];
  delete: [destination: Destination];
}>();
</script>
