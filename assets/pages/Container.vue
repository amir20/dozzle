<template lang="html">
  <pre ref="logs">
    
  </pre>
</template>

<script>
let ws;
export default {
  props: ["id"],
  name: "Container",
  mounted() {
    ws = new WebSocket(`ws://${window.location.host}/api/logs?id=${this.id}`);
    ws.onopen = e => console.log("Connection opened.");
    ws.onclose = e => console.log("Connection closed.");
    ws.onerror = e => console.error("Connection error: " + e.data);
    ws.onmessage = e => {
      const parent = this.$refs.logs;
      const item = document.createTextNode(e.data);
      parent.appendChild(item);
      parent.scrollIntoView({block: "end"});
    };
  }
};
</script>
<style>

</style>