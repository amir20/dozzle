<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container has-text-centered">
        <h1 class="title">
          Oops,
          <small class="subtitle">this page doesn't exist</small>
        </h1>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "PageNotFound",
  metaInfo() {
    return {
      title: "404 Error",
    };
  },
};
</script>
