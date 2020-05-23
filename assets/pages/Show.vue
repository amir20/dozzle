<template> </template>

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
          const [container, _] = this.visibleContainers.filter((c) => c.name == this.$route.query.name);
          if (container) {
            this.$router.push({ name: "container", params: { id: container.id } });
          } else {
            console.error(`No containers found matching name=${this.$route.query.name}. Redirecting to /`);
            this.$router.push({ name: "default" });
          }
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
