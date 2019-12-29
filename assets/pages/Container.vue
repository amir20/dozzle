<template lang="html">
  <div>
    <scrollable-logs-with-source :id="id"></scrollable-logs-with-source>
  </div>
</template>

<script>
import ScrollableLogsWithSource from "../components/ScrollableLogsWithSource";

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    ScrollableLogsWithSource
  },
  metaInfo() {
    return {
      title: this.name,
      titleTemplate: "%s - Dozzle"
    };
  }
};
</script>
