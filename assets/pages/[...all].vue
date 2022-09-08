<template>
  <div class="hero is-halfheight">
    <div class="hero-body">
      <div class="container has-text-centered">
        <h1 class="title">
          404.
          <small class="subtitle">This page does not exist.</small>
        </h1>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { setTitle } from "@/composables/title";
export default {
  name: "PageNotFound",
  setup() {
    setTitle("Page not found");
  },
};
</script>
