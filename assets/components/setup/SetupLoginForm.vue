<template>
  <!-- Login: the same choice in the setup wizard and on Settings → Security. The wizard's
       footer saves it; Settings brings its own button. -->
  <div class="flex flex-col gap-5">
    <!-- Already running with a provider. -->
    <template v-if="state === 'on'">
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <ReadOnlyRow :label="$t('setup.login.on')" :value="`auth: ${status.authProvider}`">
          <template #icon>
            <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
              <mdi:check class="size-3.5" />
            </div>
          </template>
        </ReadOnlyRow>
        <div v-if="standalone" class="p-2">
          <a
            href="https://dozzle.dev/guide/authentication"
            target="_blank"
            rel="noreferrer noopener"
            class="hover:bg-base-300 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
          >
            <mdi:account-key-outline class="size-4 opacity-60" />
            <span class="flex-1">{{ $t("settings.login-docs") }}</span>
            <mdi:open-in-new class="size-3.5 opacity-40" />
          </a>
        </div>
      </div>
      <SetupLocked v-if="status.locked.authProvider" env="DOZZLE_AUTH_PROVIDER" class="-mt-3" />
    </template>

    <!-- Saved, waiting for a restart. -->
    <template v-else-if="status.pending.authProvider">
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <ReadOnlyRow
          v-if="status.pending.authProvider === 'simple'"
          :label="$t('setup.login.saved-volume')"
          value="/data/users.yml"
        >
          <template #icon>
            <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
              <mdi:check class="size-3.5" />
            </div>
          </template>
        </ReadOnlyRow>
        <ReadOnlyRow :label="$t('setup.login.after-restart')" :value="`auth: ${status.pending.authProvider}`">
          <template #icon>
            <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
              <mdi:check class="size-3.5" />
            </div>
          </template>
        </ReadOnlyRow>
      </div>
      <!-- In Settings the restart banner above says the same, with the same snippet. -->
      <template v-if="!status.canRestart && !standalone">
        <p class="text-sm">{{ $t("setup.login.manual-restart") }}</p>
        <SetupSnippet :code="setupEnvSnippet(status)" />
      </template>
    </template>

    <!-- No login, and nothing here can change that. -->
    <template v-else-if="state === 'off'">
      <SetupAccessNotice v-if="notices && !status.locked.authProvider" :status="status" />
      <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border">
        <ReadOnlyRow :label="$t('settings.login-off')" value="auth: none">
          <template #icon>
            <div class="bg-warning/10 text-warning shrink-0 rounded-full p-1">
              <mdi:lock-open-variant-outline class="size-3.5" />
            </div>
          </template>
        </ReadOnlyRow>
      </div>
      <SetupLocked v-if="status.locked.authProvider" env="DOZZLE_AUTH_PROVIDER" class="-mt-3" />
    </template>

    <!-- Pick one. -->
    <template v-else>
      <!-- Said up front, so nobody fills in the form only to have it refused. -->
      <InlineNotice v-if="notices && !status.canWrite" type="warning">{{
        $t("setup.error.window-closed")
      }}</InlineNotice>
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
        <form v-else class="grid gap-3 sm:grid-cols-2" @submit.prevent="onSubmit">
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
              :class="{ 'input-error': mismatch }"
            />
          </label>
          <p v-if="mismatch" class="text-error text-xs sm:col-span-2" role="alert">
            {{ $t("setup.login.mismatch") }}
          </p>
          <p v-else class="text-base-content/40 text-xs sm:col-span-2">{{ $t("setup.login.password-hint") }}</p>
          <!-- Enter submits. In the wizard the visible button is the footer's Next. -->
          <button type="submit" class="hidden" aria-hidden="true" tabindex="-1"></button>
        </form>

        <div
          v-if="!status.usersFileExists"
          class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border"
        >
          <ReadOnlyRow :label="$t('setup.login.saved-volume')" value="/data/users.yml">
            <template #icon>
              <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
                <mdi:check class="size-3.5" />
              </div>
            </template>
          </ReadOnlyRow>
          <ReadOnlyRow :label="$t('setup.login.after-restart')" value="auth: simple">
            <template #icon>
              <div class="bg-success/10 text-success shrink-0 rounded-full p-1">
                <mdi:check class="size-3.5" />
              </div>
            </template>
          </ReadOnlyRow>
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

      <div v-if="standalone && saveLabel">
        <!-- Plain while the restart banner above offers its own primary Restart. -->
        <button
          type="button"
          class="btn btn-sm"
          :class="{ 'btn-primary': !restartOffered }"
          :disabled="!canSave || saving"
          @click="save"
        >
          <span v-if="saving" class="loading loading-spinner loading-xs"></span>
          {{ saveLabel }}
        </button>
      </div>
    </template>

    <InlineNotice v-if="error" type="error">{{ error }}</InlineNotice>
  </div>
</template>

<script lang="ts" setup>
import type { SetupStatus } from "@/composable/setup/setup";

const {
  status,
  standalone = false,
  notices = true,
} = defineProps<{
  status: SetupStatus;
  // False when the page already shows the access notice for every form on it.
  notices?: boolean;
  // Settings: a button of its own, a docs link once login is on, and a plain status
  // where nothing can be saved. The wizard drives saving from its footer.
  standalone?: boolean;
}>();
const emit = defineEmits<{ submit: [] }>();

const { t } = useI18n();
const { createAccount, useProxy: saveProxy } = useSetup();

type LoginMethod = "account" | "proxy" | "oidc";
const method = ref<LoginMethod>("account");
const tabs = computed<{ id: LoginMethod; label: string }[]>(() => [
  { id: "account", label: t("setup.login.tab-account") },
  { id: "proxy", label: t("setup.login.tab-proxy") },
  { id: "oidc", label: t("setup.login.tab-oidc") },
]);

// The wizard skips this step when a flag fixes the provider, and warns inside the
// chooser when the window is closed. Settings shows every install, so it says so instead.
const state = computed<"on" | "pending" | "off" | "choose">(() => {
  if (status.authProvider !== "none") return "on";
  if (status.pending.authProvider) return "pending";
  if (standalone && (status.locked.authProvider || !setupCanEdit(status))) return "off";
  return "choose";
});

const username = ref("");
const email = ref("");
const password = ref("");
const confirm = ref("");
// Flagged once the confirmation can no longer become the password, not while it is
// still being typed out.
const mismatch = computed(() => confirm.value !== "" && !password.value.startsWith(confirm.value));
const usernameValid = computed(() => /^[A-Za-z0-9_.-]{1,64}$/.test(username.value));
const accountValid = computed(
  () => usernameValid.value && password.value.length >= 8 && password.value === confirm.value,
);

// Whether this tab has anything to save: OIDC and an existing users file are set up
// in the compose file, not here.
const savable = computed(() => method.value === "proxy" || (method.value === "account" && !status.usersFileExists));
const canSave = computed(() => savable.value && (method.value !== "account" || accountValid.value));
const saveLabel = computed(() => {
  if (!savable.value) return "";
  return method.value === "account" ? t("setup.login.create-account") : t("setup.login.use-proxy");
});

// What Settings' restart banner checks before it shows its Restart button.
const restartOffered = computed(() => setupPendingChanges(status).length > 0 && setupCanRestartNow(status));

const saving = ref(false);
const error = ref("");

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

// Saves the account or the proxy choice. It applies after a restart, which the caller
// starts: the wizard right away, Settings from its banner.
async function save(): Promise<boolean> {
  if (!canSave.value) return false;
  error.value = "";
  saving.value = true;
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
    return true;
  } catch (e) {
    error.value = t(setupLoginErrorKey(e));
    return false;
  } finally {
    saving.value = false;
  }
}

function onSubmit() {
  if (standalone) save();
  else emit("submit");
}

defineExpose({ method, savable, canSave, saveLabel, save, saving, error });
</script>
