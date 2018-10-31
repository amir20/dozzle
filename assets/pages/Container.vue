<template lang="html">
  <ul ref="events" class="events">
    
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
      const parent = this.$refs.events;
      const item = document.createElement("li");
      item.classList.add("event");
      item.innerHTML = e.data;
      parent.appendChild(item);
      item.scrollIntoView();
    };
  }
};
</script>
<style>
.events {
  color: #ddd;
  background-color: #111;
  padding: 10px;
}
.event {
  font-family: monaco, monospace;
  font-size: 12px;
  line-height: 16px;
  padding: 0 15px 0 30px;
  word-wrap: break-word;  
}
</style>