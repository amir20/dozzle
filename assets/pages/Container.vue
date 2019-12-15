<template lang="html">
  <div>
    <search></search>
    <scrollable-logs-with-source :id="id"></scrollable-logs-with-source>
  </div>
</template>

<script>
import ScrollableLogsWithSource from "../components/ScrollableLogsWithSource";
import Search from "../components/Search";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    ScrollableLogsWithSource,
    Search
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  }
};
</script>
