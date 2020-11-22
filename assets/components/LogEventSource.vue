<template>
  <div>
    <infinite-loader :onLoadMore="loadOlderLogs" :enabled="messages.length > 100"></infinite-loader>
    <slot :messages="messages"></slot>
  </div>
</template>

<script>
import debounce from "lodash.debounce";
import InfiniteLoader from "./InfiniteLoader";
import config from "../store/config";

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
    this.flushBuffer = debounce(this.flushNow, 250, { maxWait: 1000 });
  },
  methods: {
    loadLogs(id) {
      this.reset();
      this.es = new EventSource(`${config.base}/api/logs/stream?id=${this.id}`);
      this.es.addEventListener("container-stopped", (e) => {
        this.es.close();
        this.buffer.push({ event: "container-stopped", message: "Container stopped", date: new Date() });
        this.flushBuffer();
        this.flushBuffer.flush();
      });
      this.es.addEventListener("error", (e) => console.error("EventSource failed: " + JSON.stringify(e)));
      this.es.onmessage = (e) => {
        this.buffer.push(this.parseMessage(e.data));
        this.flushBuffer();
      };
      this.$once("hook:beforeDestroy", () => this.es.close());
    },
    flushNow() {
      this.messages.push(...this.buffer);
      this.buffer = [];
    },
    reset() {
      if (this.es) {
        this.es.close();
        this.es = null;
        this.flushBuffer.cancel();
      }
      this.messages = [];
      this.buffer = [];
    },
    async loadOlderLogs() {
      if (this.messages.length < 300) return;

      this.$emit("loading-more", true);
      const to = this.messages[0].date;
      const last = this.messages[299].date;
      const delta = to - last;
      const from = new Date(to.getTime() + delta);
      const logs = await (
        await fetch(`${config.base}/api/logs?id=${this.id}&from=${from.toISOString()}&to=${to.toISOString()}`)
      ).text();
      if (logs) {
        const newMessages = logs
          .trim()
          .split("\n")
          .map((line) => this.parseMessage(line));
        this.messages.unshift(...newMessages);
      }
      this.$emit("loading-more", false);
    },
    parseMessage(data) {
      let i = data.indexOf(" ");
      if (i == -1) {
        i = data.length;
      }
      const key = data.substring(0, i);
      const date = new Date(key);
      const message = data.substring(i);
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
