<template lang="html">
  <div>
    <slot v-bind:messages="messages"></slot>
  </div>
</template>

<script>
function parseMessage(data) {
  const date = new Date(data.substring(0, 30));
  const key = data.substring(0, 30);
  const message = data.substring(30).trim();
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
    },
    async fetchMore() {
      const to = this.messages[0].date;
      const from = new Date(to);
      from.setMinutes(from.getMinutes() - 10);
      const logs = await (
        await fetch(`/api/logs?id=${this.id}&from=${from.toISOString()}&to=${to.toISOString()}`)
      ).text();
      const newMessages = logs
        .trim()
        .split("\n")
        .map(line => parseMessage(line));
      this.messages.unshift(...newMessages);
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
