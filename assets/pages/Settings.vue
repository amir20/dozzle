<template lang="html">
  <div class="is-fullheight">
    <h1>This is settings page</h1>
    <br />
    <br />
    <br />
    <br />
    <br />
  </div>
</template>

<script>
export default {
  props: [],
  name: "Settings",
  components: {},
  data() {
    return {};
  },
  metaInfo() {
    return {
      title: "Settings",
      titleTemplate: "%s - Dozzle"
    };
  },

  watch: {},
  methods: {},
  computed: {},
  filters: {}
};
</script>
<style scoped>
.is-fullheight {
  min-height: calc(100vh - 24px);
}
</style>
