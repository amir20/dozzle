<template lang="html">
  <div class="is-fullheight">
    <div class="search columns is-gapless is-vcentered" v-show="showSearch">
      <div class="column">
        <p class="control has-icons-left">
          <input class="input" type="text" placeholder="Filter" ref="filter" v-model="filter" />
          <span class="icon is-small is-left"><i class="fas fa-search"></i></span>
        </p>
      </div>
      <div class="column is-1 has-text-centered">
        <button class="delete is-medium" @click="resetSearch()"></button>
      </div>
    </div>

    <ul class="events">
      <li v-for="item in filtered" class="event" :key="item.key">
        <span class="date">{{ item.dateRelative }}</span> <span class="text" v-html="colorize(item.message)"></span>
      </li>
    </ul>
    <scrollbar-notification :messages="messages"></scrollbar-notification>
  </div>
</template>

<script>
import { formatRelative } from "date-fns";
import AnsiConvertor from "ansi-to-html";
import ScrollbarNotification from "../components/ScrollbarNotification";

const ansiConvertor = new AnsiConvertor();

let es = null;
let nextId = 0;

function parseMessage(data) {
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
}

export default {
  props: ["id", "name"],
  name: "Container",
  components: {
    ScrollbarNotification
  },
  data() {
    return {
      messages: [],
      showSearch: false,
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
      this.title = `${this.name}`;
    },
    onKeyDown(e) {
      if ((e.metaKey || e.ctrlKey) && e.key === "f") {
        this.showSearch = true;
        this.$nextTick(() => this.$refs.filter.focus());
        e.preventDefault();
      } else if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        this.messages = [];
      } else if (e.key === "Escape") {
        this.resetSearch();
      }
    },
    resetSearch() {
      this.showSearch = false;
      this.filter = "";
    },
    colorize: function(value) {
      return ansiConvertor.toHtml(value);
    }
  },
  computed: {
    filtered() {
      const { filter } = this;
      if (filter) {
        const isSmartCase = filter === filter.toLowerCase();
        const regex = isSmartCase ? new RegExp(filter, "i") : new RegExp(filter);
        return this.messages
          .filter(d => d.message.match(regex))
          .map(d => ({
            ...d,
            message: d.message.replace(regex, "<mark>$&</mark>")
          }));
      }
      return this.messages;
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
  width: 350px;
  position: fixed;
  padding: 10px;
  background: rgba(50, 50, 50, 0.9);
  top: 0;
  right: 0;
  border-radius: 0 0 0 5px;
}
.delete {
  margin-left: 1em;
}

/deep/ mark {
  border-radius: 2px;
  background-color: #ffdd57;
  animation: pops 0.2s ease-out;
  display: inline-block;
}

@keyframes pops {
  0% {
    transform: scale(1.5);
  }
  100% {
    transform: scale(1.05);
  }
}
</style>
