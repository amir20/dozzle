<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container">
        <section class="columns is-centered section">
          <div class="column is-4">
            <div class="card">
              <div class="card-header">
                <div class="card-header-title is-size-4">Authentication Required</div>
              </div>
              <div class="card-content">
                <form action="" method="post" @submit.prevent="onLogin" ref="form">
                  <div class="field">
                    <label class="label">Username</label>
                    <div class="control">
                      <input class="input" type="text" autocomplete="username" v-model="username" />
                    </div>
                  </div>

                  <div class="field">
                    <label class="label">Password</label>
                    <div class="control">
                      <input class="input" type="password" autocomplete="current-password" v-model="password" />
                    </div>
                  </div>
                  <div class="field is-grouped is-grouped-centered mt-5">
                    <p class="control">
                      <button class="button is-light" type="reset">Reset</button>
                    </p>
                    <p class="control">
                      <button class="button is-primary" type="submit">Submit</button>
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

<script>
import config from "../store/config";
export default {
  name: "Login",
  data() {
    return {
      username: null,
      password: null,
    };
  },
  methods: {
    async onLogin() {
      const response = await fetch(`${config.base}/api/validateCredentials`, {
        body: new FormData(this.$refs.form),
        method: "post",
      });

      if (response.status == 200) {
        window.location.href = `${config.base}/`;
      } else {
        alert("fail");
      }
    },
  },
};
</script>
