<template>
  <div class="dropdown dropdown-end">
    <button
      tabindex="0"
      type="button"
      class="icon-btn nav-btn"
      :title="$t('label.more-options')"
      :aria-label="$t('label.more-options')"
    >
      <ion:ellipsis-vertical class="size-4" />
    </button>
    <div tabindex="0" class="nav-menu-panel">
      <slot />
    </div>
  </div>
</template>
