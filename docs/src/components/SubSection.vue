<script setup></script>

<template>
  <div space-y-20 p="x-4 t-10">
    <section lg:grid-cols-3 gap="y-1 x-6">
      <h2 text="2xl" tracking-tight>
        <slot name="title" />
      </h2>
      <div grid="lg:col-span-2" prose text-lg max-w-7xl>
        <slot />
      </div>
    </section>
  </div>
</template>
