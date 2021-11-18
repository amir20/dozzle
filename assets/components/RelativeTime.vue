<template>
  <time :datetime="date.toISOString()">{{ relativeTime(date, locale) }}</time>
</template>

<script lang="ts">
import { formatRelative } from "date-fns";
import { hourStyle } from "@/composables/settings";
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
    locale() {
      const locale = styles[hourStyle.value];
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
