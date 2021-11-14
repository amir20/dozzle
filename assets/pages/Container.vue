<template>
  <div>
    <search></search>
    <log-container :id="id" show-title :scrollable="activeContainers.length > 0"> </log-container>
  </div>
</template>

<script lang="ts">
import { mapGetters } from "vuex";
import Search from "../components/Search.vue";
import LogContainer from "../components/LogContainer.vue";
import { setTitle } from "@/composables/title";

export default {
  props: ["id"],
  name: "Container",
  components: {
    LogContainer,
    Search,
  },
  created() {
    setTitle("loading");
  },

  mounted() {
    if (this.allContainersById[this.id]) {
      setTitle(this.allContainersById[this.id].name);
    }
  },
  computed: {
    ...mapGetters(["allContainersById", "activeContainers"]),
  },
  watch: {
    id() {
      setTitle(this.allContainersById[this.id].name);
    },
    allContainersById() {
      setTitle(this.allContainersById[this.id].name);
    },
  },
};
</script>
