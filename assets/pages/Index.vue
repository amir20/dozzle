<template lang="html">
  <div>
    <ul>
      <li v-for="item in containers">        
        <router-link :to="{name: 'container', params: {id: item.Id}}">{{ item.Names[0] }} {{ item.Status}}</router-link>
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