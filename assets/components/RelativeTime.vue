<template>
  <time :datetime="date.toISOString()">{{ date | relativeTime(locale) }}</time>
</template>

<script>
import { mapActions, mapState } from "vuex";
import { formatRelative } from "date-fns";
import { enGB, enUS } from "date-fns/locale";

const use24Hr =
  new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
  })
    .format(new Date(2020, 0, 1, 13))
    .replace(/[^0-9]/g, '')
    .length === 2;

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
      return styles[this.settings.hourStyle];
    },
  },
  filters: {
    relativeTime(date, locale) {
      return formatRelative(date, new Date(), { locale });
    },
  },
};
</script>
