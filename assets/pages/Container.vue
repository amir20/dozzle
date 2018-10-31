<template lang="html">
    <ul ref="events" class="events"></ul>
</template>

<script>
import { formatRelative } from "date-fns";
let ws;

const parseMessage = data => {
  const date = new Date(data.substring(0, 30));
  const dateRelative = formatRelative(date, new Date());
  const message = data.substring(30);
  return {
    date,
    dateRelative,
    message
  };
};

export default {
  props: ["id"],
  name: "Container",
  mounted() {
    ws = new WebSocket(`ws://${window.location.host}/api/logs?id=${this.id}`);
    ws.onopen = e => console.log("Connection opened.");
    ws.onclose = e => console.log("Connection closed.");
    ws.onerror = e => console.error("Connection error: " + e.data);
    ws.onmessage = e => {
      const data = parseMessage(e.data);
      const parent = this.$refs.events;
      const item = document.createElement("li");
      item.className = "event";

      const date = document.createElement("span");
      date.className = "date";
      date.innerHTML = data.dateRelative;
      item.appendChild(date);

      const message = document.createElement("span");
      message.className = "text";
      message.innerHTML = data.message;
      item.appendChild(message);

      parent.appendChild(item);

      this.$nextTick(() => item.scrollIntoView());
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

.date {
  background-color: #262626;
  color: #258ccd;
}
</style>
