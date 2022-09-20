<template>
  <svg width="100%" height="8" class="zigzag">
    <defs>
      <pattern id="zigzag" x="0" y="0" width="30" height="8" patternUnits="userSpaceOnUse">
        <line x1="0" y1="0" x2="15" y2="8" class="line" />
        <line x1="15" y1="8" x2="30" y2="0" class="line" />
      </pattern>
    </defs>
    <rect x="0" y="0" width="100%" height="100%" fill="url(#zigzag)"></rect>
  </svg>
</template>

<style lang="scss" scoped>
.line {
  stroke: var(--primary-color);
  stroke-width: 1;
  stroke-linecap: round;
}
</style>
