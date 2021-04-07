<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container has-text-centered">
        <section class="columns is-centered section">
          <div class="column is-4">
            <div class="panel">
              <p class="panel-heading">Authentication Required</p>
              <div class="panel-block">
                <p class="control">
                  <input class="input" type="text" placeholder="User" />
                </p>
                <p class="control">
                  <input class="input" type="password" placeholder="Password" />
                </p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "Login",
};
</script>
