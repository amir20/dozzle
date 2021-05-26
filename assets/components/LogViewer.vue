<template>
  <ul class="events" :class="settings.size">
    <li v-for="item in filtered" :key="item.key" :data-event="item.event">
      <span class="date" v-if="settings.showTimestamp"><relative-time :date="item.date"></relative-time></span>
      <span class="text" v-html="colorize(item.message)"></span>
    </li>
  </ul>
</template>
<script>
import { mapState } from "vuex";
import AnsiConvertor from "ansi-to-html";
import DOMPurify from "dompurify";
import RelativeTime from "./RelativeTime";

const ansiConvertor = new AnsiConvertor({ escapeXML: true });

if (window.trustedTypes && trustedTypes.createPolicy) {
  trustedTypes.createPolicy("default", {
    createHTML: (string, sink) => DOMPurify.sanitize(string, { RETURN_TRUSTED_TYPE: true }),
  });
}

export default {
  props: ["messages"],
  name: "LogViewer",
  components: { RelativeTime },
  data() {
    return {
      showSearch: false,
    };
  },
  methods: {
    colorize: function (value) {
      return ansiConvertor.toHtml(value).replace("&lt;mark&gt;", "<mark>").replace("&lt;/mark&gt;", "</mark>");
    },
  },
  computed: {
    ...mapState(["searchFilter", "settings"]),
    filtered() {
      const { searchFilter, messages } = this;
      if (searchFilter) {
        const isSmartCase = searchFilter === searchFilter.toLowerCase();
        try {
          const regex = isSmartCase ? new RegExp(searchFilter, "i") : new RegExp(searchFilter);
          return messages
            .filter((d) => d.message.match(regex))
            .map((d) => ({
              ...d,
              message: d.message.replace(regex, "<mark>$&</mark>"),
            }));
        } catch (e) {
          if (e instanceof SyntaxError) {
            console.info(`Ignoring SytaxError from search.`, e);
            return messages;
          }
          throw e;
        }
      }
      return messages;
    },
  },
};
</script>
<style scoped lang="scss">
.events {
  padding: 1em;
  font-family: SFMono-Regular, Consolas, Liberation Mono, monaco, Menlo, monospace;

  & > li {
    word-wrap: break-word;
    line-height: 130%;
    &:last-child {
      scroll-snap-align: end;
      scroll-margin-block-end: 5rem;
    }
    &[data-event="container-stopped"] {
      color: #f14668;
    }
    &[data-event="container-started"] {
      color: hsl(141, 53%, 53%);
    }
  }

  &.small {
    font-size: 60%;
  }

  &.medium {
    font-size: 80%;
  }

  &.large {
    font-size: 120%;
  }
}

.date {
  background-color: #262626;
  color: #258ccd;

  [data-theme="light"] & {
    background-color: #f0f0f0;
    color: #009900;
    padding-left: 5px;
    padding-right: 5px;
  }
}

.text {
  white-space: pre-wrap;
}

::v-deep mark {
  border-radius: 2px;
  background-color: var(--secondary-color);
  animation: pops 200ms ease-out;
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
