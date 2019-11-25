<template lang="html">
  <ul class="events">
    <li v-for="item in filtered" :key="item.key">
      <span class="date">{{ item.date | relativeTime }}</span>
      <span class="text" v-html="colorize(item.message)"></span>
    </li>
  </ul>
</template>

<script>
import { mapActions, mapGetters, mapState } from "vuex";
import { formatRelative } from "date-fns";
import AnsiConvertor from "ansi-to-html";

const ansiConvertor = new AnsiConvertor({ escapeXML: true });

export default {
  props: ["messages"],
  name: "LogViewer",
  components: {},
  data() {
    return {
      showSearch: false
    };
  },
  methods: {
    colorize: function(value) {
      return ansiConvertor
        .toHtml(value)
        .replace("&lt;mark&gt;", "<mark>")
        .replace("&lt;/mark&gt;", "</mark>");
    }
  },
  computed: {
    ...mapState(["searchFilter"]),
    filtered() {
      const { searchFilter, messages } = this;
      if (searchFilter) {
        const isSmartCase = searchFilter === searchFilter.toLowerCase();
        const regex = isSmartCase ? new RegExp(searchFilter, "i") : new RegExp(searchFilter);
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

.text {
  white-space: pre-wrap;
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
