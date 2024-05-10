<template>
  <div class="card w-96 flex-shrink-0 bg-base-lighter shadow-2xl">
    <div class="card-body">
      <form action="" method="post" @submit.prevent="onLogin" ref="form">
        <div class="form-control">
          <label class="label">
            <span class="label-text"> {{ $t("label.username") }} </span>
          </label>
          <input
            class="input input-bordered"
            type="text"
            name="username"
            autocomplete="username"
            v-model="username"
            autofocus
          />
        </div>
        <div class="form-control">
          <label class="label">
            <span class="label-text">{{ $t("label.password") }}</span>
          </label>
          <input
            class="input input-bordered"
            type="password"
            name="password"
            autocomplete="current-password"
            v-model="password"
          />
        </div>
        <label class="label text-red" v-if="error">
          {{ $t("error.invalid-auth") }}
        </label>
        <div class="form-control mt-6">
          <button class="btn btn-primary" type="submit">{{ $t("button.login") }}</button>
        </div>
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
