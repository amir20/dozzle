<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container">
        <section class="columns is-centered section">
          <div class="column is-4">
            <div class="card">
              <div class="card-content">
                <form action="" method="post" @submit.prevent="onLogin" ref="form">
                  <div class="field">
                    <label class="label">{{ $t("label.username") }}</label>
                    <div class="control">
                      <input
                        class="input"
                        type="text"
                        name="username"
                        autocomplete="username"
                        v-model="username"
                        autofocus
                      />
                    </div>
                  </div>

                  <div class="field">
                    <label class="label">{{ $t("label.password") }}</label>
                    <div class="control">
                      <input
                        class="input"
                        type="password"
                        name="password"
                        autocomplete="current-password"
                        v-model="password"
                      />
                    </div>
                    <p class="help is-danger" v-if="error">{{ $t("error.invalid-auth") }}</p>
                  </div>
                  <div class="field is-grouped is-grouped-centered mt-5">
                    <p class="control">
                      <button class="button is-primary" type="submit">{{ $t("button.login") }}</button>
                    </p>
                  </div>
                </form>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
const { t } = useI18n();

setTitle(t("title.login"));

let error = $ref(false);
let username = $ref("");
let password = $ref("");
let form: HTMLFormElement = $ref();

async function onLogin() {
  const response = await fetch(`${config.base}/api/validateCredentials`, {
    body: new FormData(form),
    method: "post",
  });

  if (response.status == 200) {
    error = false;
    window.location.href = `${config.base}/`;
  } else {
    error = true;
  }
}
</script>
<route lang="yaml">
meta:
  layout: splash
</route>
