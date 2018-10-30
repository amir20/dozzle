<template lang="html">
  <div class="container">
    <div class="content">
      <section class="section">
        <ul class="is-marginless is-paddless">
      <li v-for="item in containers" class="columns unstyled box">
        <div class="column is-4">
          <router-link :to="{name: 'container', params: {id: item.Id}}">{{ item.Names[0] }}</router-link>
        </div>
        <div class="column is-4">
          <code>{{ item.Image }}</code>
        </div>
        <div class="column is-4">
          <span class="subtitle is-7">{{ item.Status}}</span>
        </div>
      </li>
    </ul>
      </section>
    </div>
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