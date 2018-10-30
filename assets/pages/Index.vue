<template lang="html">
  <div>
    <ul>
      <li v-for="item in containers" class="columns unstyled box">
        <div class="column is-6">
          <router-link :to="{name: 'container', params: {id: item.Id}}">{{ item.Names[0] }}</router-link>
        </div>
        <div class="column is-4">
          <code>{{ item.Image }}</code>
        </div>
        <div class="column is-2">
          {{ item.Status}}
        </div>
      </li>
    </ul>
  </div>
</template>

<script>
export default {
  name: "Index",
  data() {
    return {
      containers: []
    };
  },
  async created() {
    this.containers = await (await fetch(`/api/containers.json`)).json();
  }
};
</script>

<style lang="css">
</style>