<template>
  <div class="card bg-base-100 w-96 shrink-0 shadow-2xl">
    <div class="card-body">
      <form action="" method="post" @submit.prevent="onLogin" ref="form" class="flex flex-col gap-8">
        <label class="form-control w-full">
          <label
            class="input input-lg floating-label input-bordered has-[:focus]:input-primary flex w-full items-center gap-2 border-2"
            :class="{ 'input-error': error }"
          >
            <span class="ml-5">{{ $t("label.username") }}</span>
            <mdi:account class="has-[+:focus]:text-primary" :class="{ 'text-error': error }" />
            <input
              type="text"
              :class="{ 'text-error': error }"
              :placeholder="$t('label.username')"
              name="username"
              autocomplete="username"
              autofocus
              required
              :disabled="loading"
            />
          </label>
          <label class="label" v-if="error">
            <span class="label-text-alt text-error">
              {{ $t("error.invalid-auth") }}
            </span>
          </label>
        </label>
        <label class="form-control w-full">
          <label
            class="input input-lg floating-label input-bordered has-[:focus]:input-primary flex w-full items-center gap-2 border-2"
          >
            <span class="ml-5">{{ $t("label.password") }}</span>
            <mdi:key class="has-[+:focus]:text-primary" />
            <input
              type="password"
              :placeholder="$t('label.password')"
              name="password"
              autocomplete="current-password"
              autofocus
              required
              :disabled="loading"
            />
          </label>
        </label>

        <button class="btn btn-primary mt-2 uppercase" type="submit" :disabled="loading">
          <span class="loading loading-spinner" v-if="loading"></span>
          {{ $t("button.login") }}
        </button>
      </form>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { t } = useI18n();

setTitle(t("title.login"));

const error = ref(false);
const loading = ref(false);
const form = ref<HTMLFormElement>();
const params = new URLSearchParams(window.location.search);

async function onLogin() {
  loading.value = true;
  const response = await fetch(withBase("/api/token"), {
    body: new FormData(form.value),
    method: "POST",
  });

  if (response.status == 200) {
    error.value = false;
    if (params.has("redirectUrl")) {
      window.location.href = withBase(params.get("redirectUrl")!);
    } else {
      window.location.href = withBase("/");
    }
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
