<template lang="html">
<div class="container">
    <div class="content">
        <section class="section">
            <ul class="is-marginless is-paddless">
                <li v-for="item in containers" class=" unstyled box">
                    <router-link :to="{name: 'container', params: {id: item.Id}}" class="columns">
                        <div class="column is-7">
                            <h2 class="is-2 hide-overflow">{{ item.Names[0] }}</h2>
                            <span class="subtitle is-6 code hide-overflow">{{ item.Command}}</span>
                        </div>
                        <div class="column is-4">
                            <span class="code hide-overflow">{{ item.Image }}</span>
                        </div>
                        <div class="column is-narrow">
                            <span class="subtitle is-7">{{ item.Status}}</span>
                        </div>
                    </router-link>
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
.hide-overflow {
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}
.code {
  background-color: #f5f5f5;
  color: #ff3860;
  font-size: 0.875em;
  font-weight: 400;
  padding: 0.25em 0.5em 0.25em;
  display: block;
  border-radius: 2px;
}
</style>
