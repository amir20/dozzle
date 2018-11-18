<template lang="html">
  <div class="is-fullheight">
    <ul ref="events" class="events">
      <li v-for="item in messages" class="event" :key="item.key">
        <span class="date">{{ item.dateRelative }}</span> <span class="text">{{ item.message }}</span>
      </li>
    </ul>
    <scrollbar-notification :messages="messages"></scrollbar-notification>
  </div>
</template>

<script>
import { formatRelative } from "date-fns";
import ScrollbarNotification from "../components/ScrollbarNotification";

let ws = null;
let nextId = 0;
const parseMessage = data => {
  const date = new Date(data.substring(0, 30));
  const dateRelative = formatRelative(date, new Date());
  const message = data.substring(30);
  const key = nextId++;
  return {
    key,
    date,
    dateRelative,
    message
  };
};

export default {
  props: ["id"],
  name: "Container",
  components: {
    ScrollbarNotification
  },
  data() {
    return {
      messages: []
    };
  },
  created() {
    this.loadLogs(this.id);
  },
  beforeDestroy() {
    ws.close();
    ws = null;
  },
  watch: {
    id(newValue, oldValue) {
      if (oldValue !== newValue) {
        this.loadLogs(newValue);
      }
    }
  },
  methods: {
    loadLogs(id) {
      if (ws) {
        ws.close();
        ws = null;
        this.messages = [];
      }
      ws = new WebSocket(`ws://${window.location.host}/api/logs?id=${this.id}`);
      ws.onopen = e => console.log("Connection opened.");
      ws.onclose = e => console.log("Connection closed.");
      ws.onerror = e => console.error("Connection error: " + e.data);
      ws.onmessage = e => {
        const message = parseMessage(e.data);
        this.messages.push(message);
      };
    }
  }
};
</script>
<style>
.events {
  padding: 10px;
  font-family: "Roboto Mono", monaco, monospace;
}

.event {
  font-size: 13px;
  line-height: 16px;
  word-wrap: break-word;
}

.date {
  background-color: #262626;
  color: #258ccd;
}

.is-fullheight {
  min-height: 100vh;
}
</style>
