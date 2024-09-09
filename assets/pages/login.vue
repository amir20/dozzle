<template>
  <div class="card w-96 flex-shrink-0 bg-base-lighter shadow-2xl">
    <div class="card-body">
      <form action="" method="post" @submit.prevent="onLogin" ref="form" class="flex flex-col gap-8">
        <label class="input input-bordered flex items-center gap-2 has-[:focus]:input-primary">
          <ph:user-circle class="has-[+:focus]:text-primary" />
          <input
            type="text"
            class="grow"
            :placeholder="$t('label.username')"
            v-model="username"
            name="username"
            autocomplete="username"
            autofocus
          />
        </label>
        <label class="input input-bordered flex items-center gap-2 has-[:focus]:input-primary">
          <ph:lock-key class="has-[+:focus]:text-primary" />
          <input
            type="password"
            class="grow"
            :placeholder="$t('label.password')"
            name="password"
            autocomplete="current-password"
            v-model="password"
            autofocus
          />
        </label>
        <label class="label text-red" v-if="error">
          {{ $t("error.invalid-auth") }}
        </label>

        <button class="btn btn-primary" type="submit">{{ $t("button.login") }}</button>
      </form>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { t } = useI18n();

setTitle(t("title.login"));

let error = $ref(false);
let username = $ref("");
let password = $ref("");
let form: HTMLFormElement | undefined = $ref();
const params = new URLSearchParams(window.location.search);

async function onLogin() {
  const response = await fetch(withBase("/api/token"), {
    body: new FormData(form),
    method: "POST",
  });

  if (response.status == 200) {
    error = false;
    if (params.has("redirectUrl")) {
      window.location.href = withBase(params.get("redirectUrl")!);
    } else {
      window.location.href = withBase("/");
    }
  } else {
    error = true;
  }
}
</script>
<route lang="yaml">
meta:
  layout: splash
</route>
