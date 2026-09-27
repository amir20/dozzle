<template>
  <div class="card bg-base-100 w-full max-w-md shadow-2xl">
    <div class="card-body gap-6">
      <div class="flex flex-col items-center gap-3 text-center">
        <Logo class="size-12" />
        <div>
          <h1 class="text-xl font-semibold">{{ $t("mcp-authorize.title") }}</h1>
          <p v-if="request" class="text-base-content/60 mt-1 text-sm">
            <i18n-t keypath="mcp-authorize.subtitle">
              <template #client>
                <span class="text-base-content font-semibold">{{
                  request.clientName || $t("mcp-authorize.unnamed-client")
                }}</span>
              </template>
            </i18n-t>
          </p>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-6">
        <span class="loading loading-spinner opacity-60"></span>
      </div>

      <template v-else-if="failure">
        <InlineNotice type="error" role="alert">{{ failure.description || $t("mcp-authorize.failed") }}</InlineNotice>
        <a v-if="failure.redirectUrl" :href="failure.redirectUrl" class="btn h-12 shadow-none">
          {{ $t("mcp-authorize.return") }}
        </a>
      </template>

      <InlineNotice v-else-if="redirecting" type="success">{{ $t("mcp-authorize.redirecting") }}</InlineNotice>

      <template v-else-if="request">
        <!-- The client name is whatever the client registered with, so the page
             leads with where the code is actually going. -->
        <div class="border-base-content/15 bg-base-200/40 divide-base-content/10 divide-y rounded-lg border text-sm">
          <div class="flex items-center justify-between gap-4 p-4">
            <span class="text-base-content/60">{{ $t("mcp-authorize.redirect-label") }}</span>
            <span class="truncate font-mono font-semibold" :title="request.redirectHost">{{
              request.redirectHost
            }}</span>
          </div>
          <div class="flex items-center justify-between gap-4 p-4">
            <span class="text-base-content/60">{{ $t("mcp-authorize.access-label") }}</span>
            <span class="text-right">{{ $t("mcp-authorize.access-value") }}</span>
          </div>
          <div v-if="userName" class="flex items-center justify-between gap-4 p-4">
            <span class="text-base-content/60">{{ $t("mcp-authorize.user-label") }}</span>
            <span class="truncate">{{ userName }}</span>
          </div>
        </div>

        <p class="text-base-content/40 text-xs">{{ $t("mcp-authorize.hint") }}</p>

        <div class="flex flex-col gap-2">
          <button
            type="button"
            class="btn btn-primary h-12 font-medium shadow-none"
            :disabled="deciding"
            @click="decide(true)"
          >
            <span v-if="deciding" class="loading loading-spinner loading-sm"></span>
            {{ $t("mcp-authorize.allow") }}
          </button>
          <button type="button" class="btn h-12 font-medium shadow-none" :disabled="deciding" @click="decide(false)">
            {{ $t("mcp-authorize.deny") }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import Logo from "@/logo.svg";

const { t } = useI18n();

setTitle(t("mcp-authorize.title"));

interface ConsentRequest {
  clientName: string;
  redirectHost: string;
  scope: string;
}

interface Failure {
  error: string;
  description: string;
  redirectUrl?: string;
}

// The authorization request is replayed to the backend exactly as the client sent
// it, so nothing here parses or rebuilds it.
const query = window.location.search.replace(/^\?/, "");

const loading = ref(true);
const deciding = ref(false);
const redirecting = ref(false);
const request = ref<ConsentRequest>();
const failure = ref<Failure>();
const userName = config.user?.name || config.user?.username || config.user?.email;

async function readFailure(response: Response): Promise<Failure> {
  try {
    return (await response.json()) as Failure;
  } catch {
    return { error: "server_error", description: "" };
  }
}

onMounted(async () => {
  try {
    const response = await fetch(withBase(`/api/oauth/authorize?${query}`));
    if (response.ok) {
      request.value = await response.json();
    } else {
      failure.value = await readFailure(response);
    }
  } catch {
    failure.value = { error: "network", description: "" };
  } finally {
    loading.value = false;
  }
});

async function decide(approve: boolean) {
  deciding.value = true;
  try {
    const response = await fetch(withBase("/api/oauth/authorize"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query, approve }),
    });
    if (!response.ok) {
      failure.value = await readFailure(response);
      return;
    }
    const { redirectUrl } = (await response.json()) as { redirectUrl: string };
    redirecting.value = true;
    window.location.href = redirectUrl;
  } catch {
    failure.value = { error: "network", description: "" };
  } finally {
    deciding.value = false;
  }
}
</script>

<route lang="yaml">
meta:
  layout: splash
</route>
