<script setup></script>

<template>
  <div p="x-4 lg:y-10" mb-4 text="gray-700 dark:gray-200">
    <div grid="~ lg:cols-3 lg:gap-8" h-full mx-auto>
      <div flex="~ col" text="center" p="x-4">
        <div m="t-8 lg:y-auto">
          <h1 text-4xl font-light>
            <span font-playfair dark:text-brand>Dozzle</span> is a modern log viewer for Docker providing efficient and
            intuitive ways to
            view and search logs from containers
          </h1>
          <div mt-8>
            <a href="/guide/getting-started" btn-primary>
              Get Started
            </a>
          </div>
        </div>
      </div>
      <div grid="lg:col-span-2" flex m="t-8 lg:t-0">
        <div my-auto>
          <browser-window drop-shadow-md>
            <video muted loop autoplay playsinline poster="../media/poster.png">
              <source src="../media/dozzle.webm" type="video/webm">
              <source src="../media/dozzle.mp4" type="video/mp4">
              <img src="../media/poster.png" alt="">
            </video>
          </browser-window>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
