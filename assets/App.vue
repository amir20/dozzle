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
<style lang="postcss">
html.has-custom-scrollbars {
  ::-webkit-scrollbar {
    width: 8px;
    display: content;
  }

  ::-webkit-scrollbar-thumb {
    background-color: rgba(128, 128, 128, 0.33);
    outline: 1px solid slategrey;
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb:active {
    background-color: #777;
  }

  ::-webkit-scrollbar-track {
    background-color: transparent;
  }

  ::-webkit-scrollbar-track:hover {
    background-color: rgba(64, 64, 64, 0.33);
  }

  section main {
    scrollbar-color: #353535 transparent;
    scrollbar-width: thin;
  }
}
</style>
