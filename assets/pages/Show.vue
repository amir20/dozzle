<template lang="html"> </template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
export default {
  props: [],
  name: "Show",
  computed: mapGetters(["visibleContainers"]),
  watch: {
    visibleContainers(newValue) {
      if (newValue) {
        if (this.$route.query.name) {
          const [container, _] = this.visibleContainers.filter((c) => c.name.includes(this.$route.query.name));
          this.$router.push({ name: "container", params: { id: container.id } });
        } else {
          console.error(`Expection query parameter name to be set. Redirecting to /`);
          this.$router.push({ name: "default" });
        }
      }
    },
  },
};
</script>
<style scoped></style>
