<template lang="html">
  <div>
    <infinite-loader :onLoadMore="loadOlderLogs" :enabled="messages.length > 100"></infinite-loader>
    <slot :messages="messages"></slot>
  </div>
</template>

<script>
import debounce from "lodash.debounce";
import InfiniteLoader from "./InfiniteLoader";

export default {
  props: ["id"],
  name: "LogEventSource",
  components: {
    InfiniteLoader,
  },
  data() {
    return {
      messages: [],
      buffer: [],
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
      const flushBuffer = debounce(
        () => {
          this.messages.push(...this.buffer);
          this.buffer = [];
        },
        250,
        { maxWait: 1000 }
      );
      this.es.onmessage = (e) => {
        this.buffer.push(this.parseMessage(e.data));
        flushBuffer();
      };
      this.es.onerror = (e) => console.log("EventSource failed." + e);
      this.$once("hook:beforeDestroy", () => this.es.close());
    },
    async loadOlderLogs() {
      if (this.messages.length < 300) return;

      const to = this.messages[0].date;
      const last = this.messages[299].date;
      const delta = to - last;
      const from = new Date(to.getTime() + delta);
      const logs = await (
        await fetch(`/api/logs?id=${this.id}&from=${from.toISOString()}&to=${to.toISOString()}`)
      ).text();
      if (logs) {
        const newMessages = logs
          .trim()
          .split("\n")
          .map((line) => this.parseMessage(line));
        this.messages.unshift(...newMessages);
      }
    },
    parseMessage(data) {
      let i = data.indexOf(" ");
      if (i == -1) {
        i = data.length;
      }
      const key = data.substring(0, i);
      const date = new Date(key);
      const message = data.substring(i).trim();
      return { key, date, message };
    },
  },
  watch: {
    id(newValue, oldValue) {
      if (oldValue !== newValue) {
        this.loadLogs(newValue);
      }
    },
  },
};
</script>
