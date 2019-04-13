<template lang="html">
  <div class="is-fullheight">
    <div class="search" v-show="showHelp">
      <p class="control has-icons-left">
        <input class="input" type="text" placeholder="Filter" ref="filter" v-model="filter" />
        <span class="icon is-small is-left"><i class="fas fa-search"></i></span>
      </p>
    </div>

    <ul ref="events" class="events">
      <li v-for="item in filtered" class="event" :key="item.key">
        <span class="date">{{ item.dateRelative }}</span> <span class="text" v-html="item.message"></span>
      </li>
    </ul>
    <scrollbar-notification :messages="messages"></scrollbar-notification>
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
      showHelp: false,
      title: "",
      filter: ""
    };
  },
  metaInfo() {
    return {
      title: this.title,
      titleTemplate: "%s - Dozzle"
    };
  },
  mounted() {
    window.addEventListener("keydown", this.onKeyDown);
  },
  destroyed() {
    window.removeEventListener("keydown", this.onKeyDown);
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
    },
    onKeyDown(e) {
      if ((e.metaKey || e.ctrlKey) && e.key === "f") {
        this.showHelp = true;
        this.$nextTick(() => this.$refs.filter.focus());
        e.preventDefault();
      } else if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        this.messages = [];
      } else if (e.key === "Escape") {
        this.showHelp = false;
        this.filter = "";
      }
    }
  },
  computed: {
    filtered() {
      if (this.filter) {
        return this.messages
          .filter(d => d.message.includes(this.filter))
          .map(d => {
            return {
              ...d,
              message: d.message.replace(this.filter, text => `<mark>${text}</mark>`)
            };
          });
      } else {
        return this.messages;
      }
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

.text {
  white-space: pre-wrap;
}

.search {
  width: 300px;
  position: fixed;
  padding: 10px;
  background: rgba(50, 50, 50, 0.9);
  top: 0;
  right: 0;
  border-radius: 0 0 0 5px;
}

/deep/ mark {
  border-radius: 2px;
  background-color: #ffdd57;
}
</style>
