<template>
  <div class="card w-96 flex-shrink-0 bg-base-lighter shadow-2xl">
    <div class="card-body">
      <form action="" method="post" @submit.prevent="onLogin" ref="form" class="flex flex-col gap-8">
        <label class="form-control w-full">
          <label
            class="input input-bordered flex items-center gap-2 border-2 has-[:focus]:input-primary"
            :class="{ 'input-error': error }"
          >
            <mdi:account class="has-[+:focus]:text-primary" :class="{ 'text-error': error }" />
            <input
              type="text"
              class="grow"
              :class="{ 'text-error': error }"
              :placeholder="$t('label.username')"
              name="username"
              autocomplete="username"
              autofocus
              required
              :disabled="loading"
            />
          </label>
          <label class="label text-error" v-if="error">
            {{ $t("error.invalid-auth") }}
          </label>
        </label>
        <label class="form-control w-full">
          <label class="input input-bordered flex items-center gap-2 border-2 has-[:focus]:input-primary">
            <mdi:key class="has-[+:focus]:text-primary" />
            <input
              type="password"
              class="grow"
              :placeholder="$t('label.password')"
              name="password"
              autocomplete="current-password"
              autofocus
              required
              :disabled="loading"
            />
          </label>
        </label>

        <button class="btn btn-primary uppercase" type="submit">
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
