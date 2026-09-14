<template>
  <div>
    <h2 class="text-2xl font-bold">{{ $t("setup.login.title") }}</h2>
    <p class="text-base-content/60 mt-1 text-sm">{{ $t("setup.login.subtitle") }}</p>

    <!-- Restarting: the page is about to go away, so nothing else competes with it. -->
    <div v-if="phase === 'restarting' || phase === 'timeout'" class="mt-6">
      <SetupRestarting :timed-out="phase === 'timeout'" :hint="$t('setup.login.restart-hint')" />
    </div>

    <!-- /data is not a volume: anything written now is gone on the next recreate. -->
    <div v-else-if="!status.dataPersisted" class="mt-6 flex flex-col gap-4">
      <div class="flex items-start gap-3">
        <div class="bg-warning/10 text-warning shrink-0 rounded-full p-2">
          <mdi:harddisk-remove class="size-5" />
        </div>
        <div class="min-w-0">
          <div class="text-sm font-semibold">{{ $t("setup.login.no-data-title") }}</div>
          <p class="text-base-content/60 mt-0.5 text-sm">{{ $t("setup.login.no-data-body") }}</p>
        </div>
      </div>
      <SetupSnippet :code="volumeSnippet" />
      <div>
        <button type="button" class="btn btn-sm" :disabled="loading" @click="fetchStatus">
          <span v-if="loading" class="loading loading-spinner loading-xs"></span>
          <mdi:refresh v-else class="size-4" />
          {{ $t("setup.login.check-again") }}
        </button>
        <button type="button" class="btn btn-ghost btn-sm text-base-content/60 ml-2" @click="$emit('skip')">
          {{ $t("setup.login.skip") }}
        </button>
      </div>
    </div>

    <!-- Already running with a provider. -->
    <div v-else-if="status.authProvider !== 'none'" class="mt-6">
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <SetupCheckRow :label="$t('setup.login.on')" :value="`auth: ${status.authProvider}`" />
      </div>
    </div>

    <!-- Saved, waiting for a restart. -->
    <div v-else-if="status.pending.authProvider" class="mt-6 flex flex-col gap-4">
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <SetupCheckRow
          v-if="status.pending.authProvider === 'simple'"
          :label="$t('setup.login.saved-volume')"
          value="/data/users.yml"
        />
        <SetupCheckRow :label="$t('setup.login.after-restart')" :value="`auth: ${status.pending.authProvider}`" />
      </div>
      <template v-if="!status.canRestart">
        <p class="text-sm">{{ $t("setup.login.manual-restart") }}</p>
        <SetupSnippet :code="setupEnvSnippet(status)" />
      </template>
    </div>

    <!-- Pick one. -->
    <div v-else class="mt-6 flex flex-col gap-5">
      <div role="tablist" class="tabs tabs-box tabs-sm w-fit">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          role="tab"
          class="tab"
          :class="{ 'tab-active': method === tab.id }"
          :aria-selected="method === tab.id"
          @click="method = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>

      <template v-if="method === 'account'">
        <template v-if="status.usersFileExists">
          <InlineNotice type="info">{{ $t("setup.login.users-exist") }}</InlineNotice>
          <SetupSnippet :code="simpleSnippet" />
        </template>
        <form v-else class="grid gap-3 sm:grid-cols-2" @submit.prevent="submit">
          <label class="flex flex-col gap-1">
            <span class="text-sm font-medium">{{ $t("setup.login.username") }}</span>
            <input
              v-model.trim="username"
              type="text"
              autocomplete="username"
              class="input focus:input-primary w-full text-base"
              :class="{ 'input-error': username && !usernameValid }"
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-sm font-medium">
              {{ $t("setup.login.email") }}
              <span class="text-base-content/40 font-normal">{{ $t("setup.login.optional") }}</span>
            </span>
            <input
              v-model.trim="email"
              type="email"
              autocomplete="email"
              class="input focus:input-primary w-full text-base"
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-sm font-medium">{{ $t("setup.login.password") }}</span>
            <input
              v-model="password"
              type="password"
              autocomplete="new-password"
              class="input focus:input-primary w-full text-base"
              :class="{ 'input-error': password && password.length < 8 }"
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-sm font-medium">{{ $t("setup.login.confirm") }}</span>
            <input
              v-model="confirm"
              type="password"
              autocomplete="new-password"
              class="input focus:input-primary w-full text-base"
              :class="{ 'input-error': confirm && confirm !== password }"
            />
          </label>
          <p class="text-base-content/40 text-xs sm:col-span-2">{{ $t("setup.login.password-hint") }}</p>
          <!-- Enter submits; the visible button is the footer's Next. -->
          <button type="submit" class="hidden" aria-hidden="true" tabindex="-1"></button>
        </form>

        <div
          v-if="!status.usersFileExists"
          class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
        >
          <SetupCheckRow :label="$t('setup.login.saved-volume')" value="/data/users.yml" />
          <SetupCheckRow :label="$t('setup.login.after-restart')" value="auth: simple" />
        </div>
      </template>

      <template v-else-if="method === 'proxy'">
        <p class="text-sm">{{ $t("setup.login.proxy-body") }}</p>
        <div class="flex items-start gap-3">
          <div class="bg-warning/10 text-warning shrink-0 rounded-full p-2">
            <mdi:shield-alert-outline class="size-5" />
          </div>
          <div class="min-w-0">
            <div class="text-sm font-semibold">{{ $t("setup.login.proxy-warning-title") }}</div>
            <p class="text-base-content/60 mt-0.5 text-sm">{{ $t("setup.login.proxy-warning-body") }}</p>
          </div>
        </div>
        <a
          href="https://dozzle.dev/guide/authentication/forward-proxy"
          target="_blank"
          rel="noreferrer noopener"
          class="link link-hover text-base-content/60 w-fit text-sm"
        >
          {{ $t("setup.login.proxy-docs") }}
          <mdi:open-in-new class="inline size-3.5 align-[-0.1em] opacity-40" />
        </a>
      </template>

      <template v-else>
        <p class="text-sm">{{ $t("setup.login.oidc-body") }}</p>
        <SetupSnippet :code="oidcSnippet" />
        <a
          href="https://dozzle.dev/guide/authentication/oidc"
          target="_blank"
          rel="noreferrer noopener"
          class="link link-hover text-base-content/60 w-fit text-sm"
        >
          {{ $t("setup.login.oidc-docs") }}
          <mdi:open-in-new class="inline size-3.5 align-[-0.1em] opacity-40" />
        </a>
      </template>

      <div>
        <button type="button" class="btn btn-ghost btn-sm text-base-content/60" @click="$emit('skip')">
          {{ $t("setup.login.skip") }}
        </button>
      </div>
    </div>

    <InlineNotice v-if="error" type="error" class="mt-4">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { SetupNextResult, SetupStatus, SetupStepId } from "@/composable/setup/setup";

const { status, nextStep } = defineProps<{ status: SetupStatus; nextStep?: SetupStepId }>();
defineEmits<{ skip: [] }>();

const { t } = useI18n();
const { loading, fetchStatus, createAccount, useProxy: saveProxy, restart, waitForRestart } = useSetup();

type Method = "account" | "proxy" | "oidc";
const method = ref<Method>("account");
const tabs = computed<{ id: Method; label: string }[]>(() => [
  { id: "account", label: t("setup.login.tab-account") },
  { id: "proxy", label: t("setup.login.tab-proxy") },
  { id: "oidc", label: t("setup.login.tab-oidc") },
]);

const username = ref("");
const email = ref("");
const password = ref("");
const confirm = ref("");
const usernameValid = computed(() => /^[A-Za-z0-9_.-]{1,64}$/.test(username.value));
const accountValid = computed(
  () => usernameValid.value && password.value.length >= 8 && password.value === confirm.value,
);

const phase = ref<"idle" | "saving" | "restarting" | "timeout">("idle");
const error = ref("");

const volumeSnippet = `services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
volumes:
  dozzle_data:`;

const simpleSnippet = `services:
  dozzle:
    environment:
      DOZZLE_AUTH_PROVIDER: simple`;

const oidcSnippet = `services:
  dozzle:
    environment:
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://auth.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret`;

function messageFor(e: unknown) {
  if (e instanceof SetupError) {
    if (e.status === 409) return t("setup.error.conflict");
    if (e.status === 412) return t("setup.error.no-data");
    if (e.status === 403) return t("setup.error.forbidden");
  }
  return t("setup.error.generic");
}

// Login is turned on before anything else, so a save restarts straight away and
// the wizard picks up at the next step once the user has signed in.
async function restartNow(): Promise<SetupNextResult> {
  if (!status.canRestart) return "advance";
  writeSetupResume(nextStep ?? "restart");
  phase.value = "restarting";
  try {
    await restart();
  } catch (e) {
    clearSetupResume();
    phase.value = "idle";
    error.value = messageFor(e);
    return "stay";
  }
  if (!(await waitForRestart())) phase.value = "timeout";
  return "stay";
}

async function submit() {
  if (nextDisabled.value) return;
  await next();
}

async function next(): Promise<SetupNextResult> {
  error.value = "";
  if (!status.dataPersisted) return "stay";
  if (status.authProvider !== "none") return "advance";
  if (status.pending.authProvider) {
    return status.canRestart ? restartNow() : "advance";
  }
  if (method.value === "oidc" || (method.value === "account" && status.usersFileExists)) return "skip";

  phase.value = "saving";
  try {
    if (method.value === "account") {
      await createAccount({
        username: username.value,
        password: password.value,
        ...(email.value ? { email: email.value } : {}),
      });
    } else {
      await saveProxy();
    }
  } catch (e) {
    phase.value = "idle";
    error.value = messageFor(e);
    return "stay";
  }
  phase.value = "idle";
  // Without a restart endpoint the step shows the manual instructions and waits.
  return status.canRestart ? restartNow() : "stay";
}

const nextLabel = computed(() => {
  if (!status.dataPersisted || status.authProvider !== "none") return t("setup.next");
  if (status.pending.authProvider) return status.canRestart ? t("setup.restart.button") : t("setup.next");
  if (method.value === "account" && !status.usersFileExists) return t("setup.login.create-account");
  if (method.value === "proxy") return t("setup.login.use-proxy");
  return t("setup.next");
});

const nextDisabled = computed(() => {
  if (phase.value !== "idle") return true;
  if (!status.dataPersisted) return true;
  if (status.authProvider !== "none" || status.pending.authProvider) return false;
  if (method.value === "account" && !status.usersFileExists) return !accountValid.value;
  return false;
});

const busy = computed(() => phase.value === "saving" || phase.value === "restarting");

defineExpose({ nextLabel, nextDisabled, nextPlain: false, busy, next });
</script>
