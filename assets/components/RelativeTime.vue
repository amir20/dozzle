<template>
  <time :datetime="date.toISOString()">{{ relativeTime(date, locale) }}</time>
</template>

<script>
import { mapState } from "vuex";
import { formatRelative } from "date-fns";
import enGB from "date-fns/locale/en-GB";
import enUS from "date-fns/locale/en-US";

const use24Hr =
  new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
  })
    .formatToParts(new Date(2020, 0, 1, 13))
    .find((part) => part.type === "hour").value.length === 2;

const auto = use24Hr ? enGB : enUS;
const styles = { auto, 12: enUS, 24: enGB };

export default {
  props: {
    date: {
      required: true,
      type: Date,
    },
  },
  name: "RelativeTime",
  components: {},
  computed: {
    ...mapState(["settings"]),
    locale() {
      const locale = styles[this.settings.hourStyle];
      const oldFormatter = locale.formatRelative;
      return {
        ...locale,
        formatRelative(token) {
          return oldFormatter(token) + "p";
        },
      };
    },
  },
  methods: {
    relativeTime(date, locale) {
      return formatRelative(date, new Date(), { locale });
    },
  },
};
</script>
