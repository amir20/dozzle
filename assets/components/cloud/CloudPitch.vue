<template>
  <!-- What linking gets you: findings, alerts, ask, and acting from chat.
       The parent owns the list's spacing and text size. -->
  <li class="flex items-start gap-2">
    <mdi:lightbulb-on-outline class="text-info mt-0.5 size-4 shrink-0" />
    <span>{{ $t("cloud.pitch.findings") }}</span>
  </li>
  <li class="flex items-start gap-2">
    <mdi:bell-ring-outline class="text-info mt-0.5 size-4 shrink-0" />
    <span>{{ $t("cloud.pitch.alerts") }}</span>
  </li>
  <li class="flex items-start gap-2">
    <mdi:message-outline class="text-info mt-0.5 size-4 shrink-0" />
    <span>{{ $t("cloud.pitch.ask") }}</span>
  </li>
  <li class="flex items-start gap-2">
    <mdi:remote class="text-info mt-0.5 size-4 shrink-0" />
    <span>{{ $t("cloud.pitch.control") }}</span>
  </li>
</template>
