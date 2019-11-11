<template lang="html">
  <span>
    <slot v-bind:messages="messages"></slot>
  </span>
</template>

<script>
let nextId = 0;
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
  created() {
    this.es = null;
    this.loadLogs(this.id);
  },
  methods: {
    loadLogs(id) {
      if (this.es) {
        this.es.close();
        this.messages = [];
        this.es = null;
      }
      this.es = new EventSource(`${BASE_PATH}/api/logs/stream?id=${this.id}`);
      this.es.onmessage = e => this.messages.push(parseMessage(e.data));
      this.es.onerror = function(e) {
        console.log("EventSource failed." + e);
      };
      this.$once("hook:beforeDestroy", () => this.es.close());
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
