<template>
  <time :datetime="date.toISOString()">{{ date | relativeTime(locale) }}</time>
</template>

<script>
import { formatRelative } from "date-fns";
import { enGB, enUS } from "date-fns/locale";
import { mapState } from "vuex";

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
    locale () {
      return this.settings.hour24 ? enGB : enUS;
    }
  },
  filters: {
    relativeTime(date, locale) {
      return formatRelative(date, new Date(), { locale });
    },
  },
};
</script>
