<template>
  <Popover placement="bottom-end" panel-class="nav-menu-panel">
    <template #trigger>
      <button
        type="button"
        class="icon-btn nav-btn"
        :title="$t('label.more-options')"
        :aria-label="$t('label.more-options')"
      >
        <ion:ellipsis-vertical class="size-4" />
      </button>
    </template>
    <slot />
  </Popover>
</template>
