<template>
  <time :datetime="date.toISOString()">{{ date | relativeTime }}</time>
</template>

<script>
import { formatRelative } from "date-fns";
import { enGB, enUS } from "date-fns/locale";

const use24Hr =
  new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
  })
    .formatToParts(new Date(2020, 0, 1, 13))
    .find((part) => part.type === "hour").value.length === 2;

const locale = use24Hr ? enGB : enUS;

export default {
  props: {
    date: {
      required: true,
      type: Date,
    },
  },
  name: "RelativeTime",
  components: {},

  filters: {
    relativeTime(date) {
      return formatRelative(date, new Date(), { locale });
    },
  },
};
</script>
