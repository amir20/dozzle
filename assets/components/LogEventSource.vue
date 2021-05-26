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
import containerMixin from "./mixins/container";

export default {
  props: ["id"],
  mixins: [containerMixin],
  name: "LogEventSource",
  components: {
    InfiniteLoader,
  },
  data() {
    return {
      messages: [],
      buffer: [],
      es: null,
      lastEventId: null,
    };
  },
  created() {
    this.flushBuffer = debounce(this.flushNow, 250, { maxWait: 1000 });
    this.loadLogs();
  },
  beforeDestroy() {
    this.es.close();
  },
  methods: {
    loadLogs() {
      this.reset();
      this.connect();
    },
    onContainerStopped() {
      this.es.close();
      this.buffer.push({ event: "container-stopped", message: "Container stopped", date: new Date(), key: new Date() });
      this.flushBuffer();
      this.flushBuffer.flush();
    },
    onMessage(e) {
      this.lastEventId = e.lastEventId;
      this.buffer.push(this.parseMessage(e.data));
      this.flushBuffer();
    },
    onContainerStateChange(newValue, oldValue) {
      if (newValue == "running" && newValue != oldValue) {
        this.buffer.push({
          event: "container-started",
          message: "Container started",
          date: new Date(),
          key: new Date(),
        });
        this.connect();
      }
    },
    connect() {
      this.es = new EventSource(`${config.base}/api/logs/stream?id=${this.id}&lastEventId=${this.lastEventId ?? ""}`);
      this.es.addEventListener("container-stopped", (e) => this.onContainerStopped());
      this.es.addEventListener("error", (e) => console.error("EventSource failed: " + JSON.stringify(e)));
      this.es.onmessage = (e) => this.onMessage(e);
    },
    flushNow() {
      this.messages.push(...this.buffer);
      this.buffer = [];
    },
    reset() {
      if (this.es) {
        this.es.close();
      }
      this.flushBuffer.cancel();
      this.es = null;
      this.messages = [];
      this.buffer = [];
      this.lastEventId = null;
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
      const message = data.substring(i + 1);
      return { key, date, message };
    },
  },
  watch: {
    id(newValue, oldValue) {
      if (oldValue !== newValue) {
        this.loadLogs();
      }
    },
  },
};
</script>
