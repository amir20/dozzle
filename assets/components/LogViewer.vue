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
      <li v-for="item in filtered" :key="item.key">
        <span class="date">{{ item.date | relativeTime }}</span>
        <span class="text" v-html="colorize(item.message)"></span>
      </li>
    </ul>
    <scrollbar-notification :messages="messages"></scrollbar-notification>
  </div>
</template>

<script>
import { formatRelative } from "date-fns";
import AnsiConvertor from "ansi-to-html";
import ScrollbarNotification from "./ScrollbarNotification";

const ansiConvertor = new AnsiConvertor({ escapeXML: true });

export default {
  props: ["messages"],
  name: "LogViewer",
  components: {
    ScrollbarNotification
  },
  data() {
    return {
      showSearch: false,
      filter: ""
    };
  },
  mounted() {
    window.addEventListener("keydown", this.onKeyDown);
  },
  destroyed() {
    window.removeEventListener("keydown", this.onKeyDown);
  },
  methods: {
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
      return ansiConvertor
        .toHtml(value)
        .replace("&lt;mark&gt;", "<mark>")
        .replace("&lt;/mark&gt;", "</mark>");
    }
  },
  computed: {
    filtered() {
      const { filter, messages } = this;

      if (filter) {
        const isSmartCase = filter === filter.toLowerCase();
        const regex = isSmartCase ? new RegExp(filter, "i") : new RegExp(filter);
        return messages
          .filter(d => d.message.match(regex))
          .map(d => ({
            ...d,
            message: d.message.replace(regex, "<mark>$&</mark>")
          }));
      }
      return messages;
    }
  },
  filters: {
    relativeTime(date) {
      return formatRelative(date, new Date());
    }
  }
};
</script>
<style scoped lang="scss">
.events {
  padding: 10px;
  font-family: "Roboto Mono", monaco, monospace;

  & > li {
    font-size: 13px;
    line-height: 16px;
    word-wrap: break-word;
  }
}

.date {
  background-color: #262626;
  color: #258ccd;
}

.is-fullheight {
  height: 100vh;
  overflow: auto;
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

>>> mark {
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
