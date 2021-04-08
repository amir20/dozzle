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
                    <label class="label">Username</label>
                    <div class="control">
                      <input class="input" type="text" name="username" autocomplete="username" v-model="username" />
                    </div>
                  </div>

                  <div class="field">
                    <label class="label">Password</label>
                    <div class="control">
                      <input
                        class="input"
                        type="password"
                        name="password"
                        autocomplete="current-password"
                        v-model="password"
                      />
                    </div>
                    <p class="help is-danger" v-if="error">Username and password are not valid.</p>
                  </div>
                  <div class="field is-grouped is-grouped-centered mt-5">
                    <p class="control">
                      <button class="button is-primary" type="submit">Login</button>
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
      error: false,
    };
  },
  metaInfo() {
    return {
      title: "Authentication Required",
    };
  },
  methods: {
    async onLogin() {
      const response = await fetch(`${config.base}/api/validateCredentials`, {
        body: new FormData(this.$refs.form),
        method: "post",
      });

      if (response.status == 200) {
        this.error = false;
        window.location.href = `${config.base}/`;
      } else {
        this.error = true;
      }
    },
  },
};
</script>
