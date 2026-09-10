<template>
  <!--
    One shape for everything Cloud hands us locked.

    The gate is never inferred here. Cloud computes it against the plan at the
    moment it answers and sends `locked` with the item, because reading a
    missing field as "not on your plan" is how a customer who already upgraded
    gets shown an advert for what they just bought. If the field is absent the
    content simply renders.

    A locked value is one muted line with a pill. Not a card, not a gradient,
    not a modal: the surface around it still wants the reader to press
    something else, and `primary` belongs to that.
  -->
  <div v-if="!locked" class="text-sm">
    <slot />
  </div>
  <div v-else class="text-base-content/40 flex items-center gap-2 text-sm">
    <mdi:lock-outline class="size-3.5 shrink-0" />
    <span class="min-w-0 flex-1">{{ reason || $t("cloud.locked-on-this-plan") }}</span>
    <span class="status-pill status-pill-secondary shrink-0">PRO</span>
  </div>
</template>

<script lang="ts" setup>
defineProps<{
  /**
   * Whether Cloud locked this value. Comes from the payload alongside the
   * content it gates (a finding's `fixLocked`, say) — never from `isPro`,
   * which exists for cosmetics and says nothing about a specific item.
   */
  locked?: boolean;
  /** Cloud's own explanation, when it sent one. */
  reason?: string;
}>();
</script>
