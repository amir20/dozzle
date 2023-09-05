<template>
  <div class="is-hoverable dropdown">
    <div class="dropdown-trigger">
      <slot name="trigger">
        <button class="button" aria-haspopup="true" aria-controls="dropdown-menu">
          <span class="icon">
            <mdi:dots-vertical />
          </span>
        </button>
      </slot>
    </div>
    <div class="dropdown-menu" id="dropdown-menu" role="menu">
      <div class="dropdown-content">
        <slot></slot>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup></script>

<style lang="scss" scoped>
.minimal .button {
  background-color: rgba(0, 0, 0, 0);
  border: none;
  padding: 0.1em;
  height: 100%;

  & > .icon {
    height: 100%;

    & > svg {
      align-self: flex-start;
      height: 0.85em;
    }
  }
}

.is-top {
  & .dropdown-menu {
    top: 0;
  }

  &.is-last .dropdown-menu {
    top: -30px;
  }
}
</style>
