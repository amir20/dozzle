<template lang="html">
  <ul ref="logs">
    
  </ul>
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
      const item = document.createElement("li");
      item.innerHTML = e.data;
      parent.appendChild(item);
    };
  }
};
</script>
<style>
ul {
  padding: 0;
  margin: 0;
}
ul li {
  list-style-type: none;
}
</style>