<template>
  <router-view></router-view>
</template>

<script lang="ts" setup>
watchEffect(() => {
  if (smallerScrollbars.value) {
    document.documentElement.classList.add("has-custom-scrollbars");
  } else {
    document.documentElement.classList.remove("has-custom-scrollbars");
  }

  switch (lightTheme.value) {
    case "dark":
      document.documentElement.setAttribute("data-theme", "dark");
      break;
    case "light":
      document.documentElement.setAttribute("data-theme", "light");
      break;
    default:
      document.documentElement.removeAttribute("data-theme");
  }
});
</script>

<style scoped lang="scss"></style>
