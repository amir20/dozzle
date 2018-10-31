<template lang="html">
  <div class="container">
    <div class="content">
      <section class="section">
        <ul class="is-marginless is-paddless">
      <li v-for="item in containers" class="columns unstyled box">
        <div class="column is-4">
          <router-link :to="{name: 'container', params: {id: item.Id}}"><h2 class="is-2">{{ item.Names[0] }}</h2></router-link>
          <span class="subtitle is-6">{{ item.Command}}</span>
        </div>
        <div class="column is-4 image-name">
          <code>{{ item.Image }} {{ item.Image }}</code>
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
.image-name {
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}
</style>