<template>
  <div>
    <InlineNotice v-if="!status.dataPersisted" type="warning">{{ $t("setup.error.no-data") }}</InlineNotice>
    <InlineNotice v-else-if="!status.canWrite" type="info">
      {{ status.authProvider === "none" ? $t("setup.actions.window-closed") : $t("setup.actions.no-access") }}
    </InlineNotice>

    <section :class="{ 'mt-4': !canEdit }">
      <FormStepHeading :step="1" :title="$t('setup.hosts.run-title')" />
      <p class="text-base-content/60 mb-3 text-sm">{{ $t("setup.hosts.run-body") }}</p>

      <label v-if="!status.customCert" class="mb-3 flex items-start justify-between gap-3">
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ $t("setup.hosts.private-label") }}</span>
          <span class="text-base-content/60 mt-0.5 block text-xs">{{ $t("setup.hosts.private-desc") }}</span>
        </span>
        <input
          v-model="usePrivate"
          type="checkbox"
          class="toggle toggle-primary toggle-sm mt-0.5 shrink-0"
          :disabled="!canEdit || adding"
        />
      </label>
      <InlineNotice v-if="certError" type="warning" class="mb-3">{{ certError }}</InlineNotice>

      <div v-if="usePrivate && !cert" class="bg-base-300 flex h-24 items-center justify-center rounded-lg">
        <span class="loading loading-spinner loading-sm"></span>
      </div>
      <SetupSnippet v-else :code="agentSnippet" />
      <p v-if="usePrivate && cert" class="text-base-content/60 mt-2 flex items-start gap-1.5 text-xs">
        <mdi:key-outline class="text-warning mt-px size-3.5 shrink-0" />
        {{ $t("setup.hosts.private-warning") }}
      </p>
      <p v-if="status.customCert" class="text-base-content/40 mt-2 text-xs">
        {{ $t("setup.hosts.custom-cert") }}
        <a href="https://dozzle.dev/guide/agent#custom-certificates" target="_blank" rel="noopener" class="link">
          {{ $t("setup.hosts.cert-docs") }}
        </a>
      </p>
    </section>

    <section class="mt-6">
      <FormStepHeading :step="2" :title="$t('setup.hosts.connect-title')" />
      <form class="grid gap-3 sm:grid-cols-[2fr_1fr_auto] sm:items-end" @submit.prevent="add">
        <label class="flex flex-col gap-1">
          <span class="text-sm font-medium">{{ $t("setup.hosts.address") }}</span>
          <input
            v-model.trim="address"
            type="text"
            autocomplete="off"
            spellcheck="false"
            placeholder="10.0.0.5:7007"
            class="input focus:input-primary w-full font-mono text-base"
            :class="{ 'input-error': address && !addressValid }"
            :disabled="!canEdit || adding"
          />
        </label>
        <label class="flex flex-col gap-1">
          <span class="text-sm font-medium">
            {{ $t("setup.hosts.name") }}
            <span class="text-base-content/40 font-normal">{{ $t("setup.hosts.optional") }}</span>
          </span>
          <input
            v-model.trim="name"
            type="text"
            autocomplete="off"
            class="input focus:input-primary w-full text-base"
            :class="{ 'input-error': name && !nameValid }"
            :disabled="!canEdit || adding"
          />
        </label>
        <button
          type="submit"
          class="btn btn-primary"
          :disabled="!canEdit || adding || !addressValid || !nameValid || waitingForCert"
        >
          <span v-if="adding" class="loading loading-spinner loading-xs"></span>
          <mdi:plus v-else class="size-4" />
          {{ $t("setup.hosts.add") }}
        </button>
      </form>
      <p class="text-base-content/40 mt-2 text-xs">{{ $t("setup.hosts.connect-note") }}</p>

      <InlineNotice v-if="error" type="error" class="mt-3">{{ error }}</InlineNotice>
      <InlineNotice v-else-if="added" type="success" class="mt-3">
        {{ $t("setup.hosts.added", { name: added }) }}
      </InlineNotice>
    </section>

    <section v-if="agents.length" class="mt-6">
      <h3 class="text-base-content/60 mb-2 text-sm font-semibold tracking-wide uppercase">
        {{ $t("setup.hosts.list-title") }}
      </h3>
      <ul class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <li v-for="agent in agents" :key="agent.endpoint" class="flex items-center gap-3 p-4">
          <div class="relative shrink-0">
            <HostIcon type="agent" class="size-4 opacity-60" />
            <span
              class="ring-base-200 absolute -right-0.5 -bottom-0.5 size-1.5 rounded-full ring-2"
              :class="agent.hostId ? 'bg-success' : 'bg-error'"
            ></span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate text-sm font-medium">{{ agent.name || agent.address }}</span>
              <span v-if="agent.private" class="status-pill status-pill-neutral shrink-0">
                {{ $t("setup.hosts.private-pill") }}
              </span>
            </div>
            <div class="text-base-content/60 truncate text-xs">
              <span v-if="agent.name" class="font-mono">{{ agent.address }} · </span>
              {{ agent.hostId ? $t("setup.hosts.connected") : $t("setup.hosts.offline") }}
            </div>
          </div>
          <span v-if="agent.locked" class="text-base-content/40 flex shrink-0 items-center gap-1 text-xs">
            <mdi:lock-outline class="size-3.5" />
            {{ $t("setup.actions.locked", { env: "DOZZLE_REMOTE_AGENT" }) }}
          </span>
          <button
            v-else
            type="button"
            class="btn btn-sm text-error shrink-0"
            :disabled="!canEdit || removing === agent.endpoint"
            @click="remove(agent)"
          >
            <span v-if="removing === agent.endpoint" class="loading loading-spinner loading-xs"></span>
            {{ $t("setup.hosts.remove") }}
          </button>
        </li>
      </ul>
    </section>
  </div>
</template>

<script lang="ts" setup>
import { SetupError, type SetupAgent, type SetupAgentCert, type SetupStatus } from "@/composable/setup/setup";

const { status } = defineProps<{ status: SetupStatus }>();

const { t } = useI18n();
const { addAgent, agentCert, removeAgent } = useSetup();
const { removeHost } = useHosts();

const address = ref("");
const name = ref("");
const adding = ref(false);
const removing = ref<string | null>(null);
const error = ref("");
const added = ref("");

const agents = computed(() => status.agents ?? []);
const canEdit = computed(() => status.dataPersisted && status.canWrite);

// The endpoint grammar is address|name|group, so a pipe in either field would
// quietly turn into a name or a group.
const addressValid = computed(() => /^[^\s|]+$/.test(address.value));
const nameValid = computed(() => /^[^|]*$/.test(name.value));

// Every agent listens on 7007 unless told otherwise, so a bare host is taken to
// mean that port rather than rejected. A bare IPv6 address is all colons, so it
// only has a port once bracketed, and gets brackets when it has none.
function withPort(value: string) {
  if (/^\[.+\]:\d{1,5}$/.test(value) || /^[^:]+:\d{1,5}$/.test(value)) return value;
  if (value.includes(":") && !value.startsWith("[")) return `[${value}]:7007`;
  return `${value}:7007`;
}

// A private pair is only on offer while the hub runs the shared certificate. With
// a custom pair of its own, every agent needs that pair anyway.
const usePrivate = ref(!status.customCert);
// Holds a private key, so it stays in this component and is gone when it unmounts.
const cert = shallowRef<SetupAgentCert | null>(null);
const certError = ref("");
let certLoading = false;

// Nothing to fetch for someone who cannot add a host, so the toggle is off. It
// comes back on if a later status says they can (a refetch after the wizard).
watch(canEdit, (editable) => (usePrivate.value = editable && !status.customCert), { immediate: true });

// Fetched the first time the toggle is on while the panel is showing, not before:
// the first call is what creates the pair in /data.
watch(
  [usePrivate, canEdit],
  async ([on, editable]) => {
    if (!on || !editable || cert.value || certLoading || status.customCert) return;
    certLoading = true;
    certError.value = "";
    try {
      cert.value = await agentCert();
    } catch (e) {
      usePrivate.value = false;
      certError.value = t("setup.hosts.private-error", {
        reason: e instanceof SetupError ? errorMessage(e) : "",
      }).trim();
    } finally {
      certLoading = false;
    }
  },
  { immediate: true },
);

// Adding before the pair arrives would quietly add the host without it.
const waitingForCert = computed(() => usePrivate.value && !cert.value);

const agentSnippet = computed(() =>
  agentComposeSnippet(
    agentImage(status.autoUpdate?.image, config.version),
    usePrivate.value && cert.value ? cert.value : undefined,
  ),
);

function errorMessage(e: unknown) {
  if (!(e instanceof SetupError)) return t("setup.error.generic");
  if (e.status === 403)
    return status.authProvider === "none" ? t("setup.actions.window-closed") : t("setup.actions.no-access");
  switch (e.code) {
    case "exists":
      return t("setup.hosts.error-exists");
    // Two addresses, one Docker engine: most often an agent beside the hub.
    case "duplicate-host":
      return t("setup.hosts.error-duplicate");
    case "no-private-cert":
      return t("setup.hosts.private-missing");
    case "not-persisted":
      return t("setup.error.no-data");
    // Reached the agent, but the two ends hold different pairs. The raw TLS error
    // ("unknown certificate authority") says nothing about what to do. With a custom
    // pair the snippet carries no certificate, so copying it again cannot help; the
    // agent needs the same files mounted.
    case "cert-mismatch":
      return t(status.customCert ? "setup.hosts.error-cert-custom" : "setup.hosts.error-cert");
    case "unreachable":
      return t("setup.hosts.error-connect", { reason: e.message.replace(/^could not connect to agent:\s*/i, "") });
    case "unsupported-mode":
      return t("setup.hosts.error-unsupported-mode");
    case "invalid":
      return t("setup.hosts.error-invalid");
    case "rate-limited":
      return t("setup.hosts.error-rate-limited");
    case "env-agent":
      return t("setup.hosts.error-env-agent");
    case "not-found":
      return t("setup.hosts.error-not-found");
    case "custom-cert":
      return t("setup.hosts.error-custom-cert");
    default:
      return e.message || t("setup.error.generic");
  }
}

async function add() {
  if (!canEdit.value || !addressValid.value || !nameValid.value || waitingForCert.value) return;
  error.value = "";
  added.value = "";
  adding.value = true;
  try {
    const host = await addAgent({
      address: withPort(address.value),
      name: name.value || undefined,
      private: usePrivate.value,
    });
    added.value = host.name;
    address.value = "";
    name.value = "";
  } catch (e) {
    // The pair went missing from /data since it was fetched: drop it so turning
    // the toggle back on asks for a fresh one.
    if (e instanceof SetupError && e.code === "no-private-cert") {
      cert.value = null;
      usePrivate.value = false;
    }
    error.value = errorMessage(e);
  } finally {
    adding.value = false;
  }
}

async function remove(agent: SetupAgent) {
  error.value = "";
  added.value = "";
  removing.value = agent.endpoint;
  try {
    await removeAgent(agent.endpoint);
    removeHost(agent.hostId, agent.endpoint);
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    removing.value = null;
  }
}

const busy = computed(() => adding.value || removing.value !== null);

defineExpose({ busy });
</script>
