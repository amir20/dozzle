<template lang="html">
  <div class="is-fullheight">
    <ul ref="events" class="events">
      <li v-for="item in messages" class="event" :key="item.key">
        <span class="date">{{ item.dateRelative }}</span> <span class="text item-message">{{ item.message }}</span>
      </li>
    </ul>
    <scrollbar-notification :messages="messages"></scrollbar-notification>
    <vue-headful :title="title" />
  </div>
</template>

<script>
import { formatRelative } from "date-fns";
import ScrollbarNotification from "../components/ScrollbarNotification";

let es = null;
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
  props: ["id", "name"],
  name: "Container",
  components: {
    ScrollbarNotification
  },
  data() {
    return {
      messages: [],
      title: ""
    };
  },
  created() {
    this.loadLogs(this.id);
  },
  beforeDestroy() {
    if (es) {
      es.close();
      es = null;
    }
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
      if (es) {
        es.close();
        es = null;
        this.messages = [];
      }
      es = new EventSource(`${BASE_PATH}/api/logs/stream?id=${id}`);
      es.onmessage = e => this.messages.push(parseMessage(e.data));
      this.title = `${this.name} - Dozzle`;
    }
  }
};
</script>
<style scoped>
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

.item-message {
  white-space: pre;
}
</style>
