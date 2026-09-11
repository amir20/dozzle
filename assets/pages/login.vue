<template>
  <!-- Capped rather than fixed: `w-96` plus the hero's own padding is 416px, so a
       384px card horizontally scrolled the login page on every phone. `max-w-full`
       does not save it, because the hero shrinks to fit its content and resolves a
       percentage max-width against a width it is still computing. -->
  <div class="card bg-base-100 w-full max-w-96 shadow-2xl">
    <div class="card-body gap-6">
      <div class="flex flex-col items-center gap-3 text-center">
        <Logo class="size-12" />
        <h1 class="text-xl font-semibold">{{ $t("title.login") }}</h1>
      </div>

      <InlineNotice v-if="oauthError" type="error" role="alert">{{ $t("error.oauth-failed") }}</InlineNotice>

      <form
        v-if="passwordLogin"
        action=""
        method="post"
        @submit.prevent="onLogin"
        ref="form"
        class="flex flex-col gap-3"
      >
        <!-- The placeholder carries the label, so aria-label keeps it announced. -->
        <label
          class="input bg-base-200 border-base-content/15 focus-within:border-primary h-12 w-full"
          :class="fieldClass"
        >
          <mdi:account class="size-[1.15rem] opacity-45" />
          <input
            type="text"
            name="username"
            :placeholder="$t('label.username')"
            :aria-label="$t('label.username')"
            autocomplete="username"
            autofocus
            required
            :disabled="loading"
          />
        </label>

        <label
          class="input bg-base-200 border-base-content/15 focus-within:border-primary h-12 w-full"
          :class="fieldClass"
        >
          <mdi:key class="size-[1.15rem] opacity-45" />
          <input
            type="password"
            name="password"
            :placeholder="$t('label.password')"
            :aria-label="$t('label.password')"
            autocomplete="current-password"
            required
            :disabled="loading"
          />
        </label>

        <p class="text-error -mt-1 text-sm" v-if="error">{{ $t("error.invalid-auth") }}</p>

        <button class="btn btn-primary mt-1 h-12 font-medium shadow-none" type="submit" :disabled="loading">
          <span class="loading loading-spinner loading-sm" v-if="loading"></span>
          {{ $t("button.login") }}
        </button>
      </form>

      <div class="divider my-0 text-xs opacity-50" v-if="passwordLogin && oauthProviders.length">
        {{ $t("label.or") }}
      </div>

      <div class="flex flex-col gap-2" v-if="oauthProviders.length">
        <a
          v-for="provider in oauthProviders"
          :key="provider.name"
          :href="loginUrlFor(provider)"
          class="btn border-base-content/15 bg-base-200 hover:border-base-content/25 hover:bg-base-300 h-12 gap-2.5 font-medium shadow-none"
        >
          <component :is="iconFor(provider.icon)" class="size-[1.15rem] opacity-80" />
          {{ $t("button.login-with", { provider: provider.name }) }}
        </a>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Component } from "vue";
import Logo from "@/logo.svg";
import MdiGithub from "~icons/mdi/github";
import MdiShieldAccount from "~icons/mdi/shield-account";
import MdiLoginVariant from "~icons/mdi/login-variant";

const { t } = useI18n();

setTitle(t("title.login"));

const error = ref(false);
const loading = ref(false);
const form = ref<HTMLFormElement>();
const params = new URLSearchParams(window.location.search);
const oauthError = params.has("error");
const oauthProviders = config.oauthProviders ?? [];
// Absent when no OAuth provider is configured, in which case the password form is
// the only way in and always shows.
const passwordLogin = config.passwordLogin ?? true;

// Bad credentials are a property of the pair, so both fields turn red together.
const fieldClass = computed(() => (error.value ? "border-error focus-within:border-error" : ""));

// unplugin-icons resolves icons at compile time, so map the backend's icon id explicitly.
const icons: Record<string, Component> = {
  "mdi:github": MdiGithub,
  "mdi:shield-account": MdiShieldAccount,
};

function iconFor(icon: string): Component {
  return icons[icon] ?? MdiLoginVariant;
}

// loginUrl is already prefixed with base by the backend, so it is used as is.
function loginUrlFor({ loginUrl }: { loginUrl: string }) {
  const redirectUrl = params.get("redirectUrl");
  return redirectUrl ? `${loginUrl}&redirectUrl=${encodeURIComponent(redirectUrl)}` : loginUrl;
}

async function onLogin() {
  loading.value = true;
  const response = await fetch(withBase("/api/token"), {
    body: new FormData(form.value),
    method: "POST",
  });

  if (response.status == 200) {
    error.value = false;
    window.location.href = safeRedirect(params.get("redirectUrl"), config.base, window.location.origin);
  } else {
    error.value = true;
  }
  loading.value = false;
}
</script>
<route lang="yaml">
meta:
  layout: splash
</route>
