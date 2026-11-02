<template>
  <!-- A row in the page's destinations panel rather than a card of its own. At 288px a card held a
       name and a count and left the rest of the page empty; full width has room for what a
       destination actually is: where it posts, in what shape, and which rules feed it. -->
  <div>
    <div class="flex items-start gap-3 p-4">
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-full"
        :class="destination.type === 'cloud' ? 'bg-primary/10 text-primary' : 'bg-base-content/10'"
      >
        <mdi:cloud v-if="destination.type === 'cloud'" class="size-4" />
        <mdi:webhook v-else class="size-4" />
      </div>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
          <button type="button" class="truncate text-left font-semibold hover:underline" @click="editDestination">
            {{ destination.name }}
          </button>
          <!-- The type, unless the name already says it (the cloud destination is named after it). -->
          <span v-if="typeLabel !== destination.name" class="text-base-content/60 text-xs">{{ typeLabel }}</span>
          <!-- Two webhooks are otherwise indistinguishable once named vaguely. -->
          <span v-if="webhookHost" class="text-base-content/40 truncate font-mono text-xs">{{ webhookHost }}</span>
        </div>

        <!-- How it sends: label left, value right, the same column the alert rows use. -->
        <div
          v-if="details.length"
          class="text-base-content/60 mt-2 grid grid-cols-[minmax(0,7rem)_1fr] gap-x-3 gap-y-1 text-xs"
        >
          <template v-for="row in details" :key="row.label">
            <span>{{ row.label }}</span>
            <span class="text-base-content/90 truncate font-mono">
              {{ row.value }}<span v-if="row.masked" class="text-base-content/30 tracking-widest">••••••••</span>
            </span>
          </template>
        </div>

        <!-- Which rules feed it, by name. A count said how much a delete would break but not what. -->
        <div class="mt-3 flex flex-wrap items-center gap-1.5 text-xs">
          <span v-if="usedBy.length" class="text-base-content/60 mr-1">
            {{ $t("notifications.destination.used-by", usedBy.length) }}
          </span>
          <span
            v-for="alert in usedBy"
            :key="alert.id"
            class="border-base-content/15 bg-base-100 rounded border px-1.5 py-0.5"
            :class="{ 'text-base-content/40 line-through': !alert.enabled }"
          >
            {{ alert.name }}
          </span>
          <span v-if="!usedBy.length" class="text-base-content/40">{{ $t("notifications.destination.unused") }}</span>
        </div>
      </div>

      <Popover
        placement="bottom-end"
        panel-class="rounded-box bg-base-200 border-base-content/10 w-40 border p-1 shadow-sm"
        @click.stop
      >
        <template #trigger>
          <button type="button" class="btn btn-ghost btn-sm btn-square">
            <ion:ellipsis-vertical />
          </button>
        </template>
        <ul class="menu w-full p-0">
          <li>
            <a @click="editDestination">{{ $t("notifications.destination.edit") }}</a>
          </li>
          <li v-if="destination.type !== 'cloud'">
            <a @click="duplicateDestination">{{ $t("notifications.destination.duplicate") }}</a>
          </li>
          <li v-if="destination.type !== 'cloud'">
            <a class="text-error" @click="confirmingDelete = true">
              {{ $t("notifications.destination.delete") }}
            </a>
          </li>
        </ul>
      </Popover>
    </div>

    <!-- Deleting orphans every alert pointing here, so name the cost before doing it. -->
    <div
      v-if="confirmingDelete"
      class="border-base-content/10 flex flex-wrap items-center justify-between gap-2 border-t px-4 py-2 text-xs"
    >
      <span class="flex items-center gap-2">
        <mdi:alert-outline class="text-error size-4 shrink-0" />
        <span class="font-semibold">{{ $t("notifications.destination.delete-warning") }}</span>
        <span class="text-base-content/60">
          {{
            usedBy.length
              ? $t("notifications.destination.delete-warning-used", usedBy.length)
              : $t("notifications.destination.unused")
          }}
        </span>
      </span>
      <span class="flex shrink-0 gap-2">
        <button class="btn btn-xs" :disabled="isDeleting" @click="confirmingDelete = false">
          {{ $t("notifications.destination.delete-cancel") }}
        </button>
        <button class="btn btn-xs btn-error" :disabled="isDeleting" @click="deleteDestination">
          <span v-if="isDeleting" class="loading loading-spinner loading-xs"></span>
          {{ $t("notifications.destination.delete-confirm") }}
        </button>
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Dispatcher, NotificationRule } from "@/types/notifications";
import DestinationForm from "./DestinationForm.vue";
import { PAYLOAD_TEMPLATES, type PayloadFormat } from "./payloadTemplates";

const { destination, onUpdated, existingDispatchers, usedBy } = defineProps<{
  destination: Dispatcher;
  onUpdated?: () => void;
  existingDispatchers: Dispatcher[];
  /** The alerts that send here. Listed on the row and counted in the delete warning. */
  usedBy: NotificationRule[];
}>();

const { t } = useI18n();
const { showToast } = useToast();
const showDrawer = useDrawer();

const confirmingDelete = ref(false);
const isDeleting = ref(false);

const typeLabel = computed(() =>
  destination.type === "webhook"
    ? t("notifications.destination.http-webhook")
    : t("notifications.destination.dozzle-cloud"),
);

/** Host of the webhook URL, so cards for two webhooks aren't identical. */
const webhookHost = computed(() => {
  if (!destination.url) return "";
  try {
    return new URL(destination.url).host;
  } catch {
    return "";
  }
});

/** A template that is byte-for-byte a preset is named after it; anything else is the user's own. */
const payloadFormat = computed(() => {
  const template = destination.template?.trim();
  if (!template) return undefined;
  const preset = (Object.keys(PAYLOAD_TEMPLATES) as PayloadFormat[]).find(
    (format) => format !== "custom" && PAYLOAD_TEMPLATES[format] === template,
  );
  return t(`notifications.destination-form.format-${preset ?? "custom"}`);
});

const details = computed(() => {
  const rows: { label: string; value: string; masked?: boolean }[] = [];
  if (destination.type === "cloud") {
    if (destination.prefix) {
      rows.push({ label: t("notifications.destination-form.api-key"), value: destination.prefix, masked: true });
    }
    return rows;
  }
  if (payloadFormat.value) {
    rows.push({ label: t("notifications.destination-form.payload-format"), value: payloadFormat.value });
  }
  const headers = Object.keys(destination.headers ?? {});
  if (headers.length) {
    rows.push({ label: t("notifications.destination-form.headers"), value: headers.join(", ") });
  }
  return rows;
});

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
    showToast({
      type: "error",
      message: e instanceof Error ? escapeHtml(e.message) : t("notifications.destination.copy-failed"),
    });
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
      message: e instanceof Error ? escapeHtml(e.message) : t("notifications.destination.delete-failed"),
    });
  } finally {
    isDeleting.value = false;
  }
}
</script>
