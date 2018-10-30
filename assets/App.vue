<template lang="html">
  <div class="content">
    <p>
      <router-link to="/">Go back</router-link>      
    </p>
    <router-view></router-view>
  </div>
</template>

<script>
export default {
  name: "App"
};
</script>

<style lang="css">
</style>