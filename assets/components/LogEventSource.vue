<template lang="html">
  <span>
    <slot v-bind:messages="messages"></slot>
  </span>
</template>

<script>
let nextId = 0;
let es = null;
function parseMessage(data) {
  const date = new Date(data.substring(0, 30));
  const message = data.substring(30);
  const key = nextId++;
  return {
    key,
    date,
    message
  };
}

export default {
  props: ["id"],
  name: "LogEventSource",
  data() {
    return {
      messages: []
    };
  },
  mounted() {
    this.loadLogs(this.id);
  },
  methods: {
    loadLogs(id) {
      if (es) {
        es.close();
        this.messages = [];
        es = null;
      }
      es = new EventSource(`${BASE_PATH}/api/logs/stream?id=${this.id}`);
      es.onmessage = e => this.messages.push(parseMessage(e.data));
      this.$once("hook:beforeDestroy", () => es.close());
    }
  },
  watch: {
    id(newValue, oldValue) {
      if (oldValue !== newValue) {
        this.loadLogs(newValue);
      }
    }
  }
};
</script>
