<script setup lang="ts">
const menu = [
  {
    name: 'What is Dozzle?',
    path: '/guide/what-is-dozzle',
  },
  {
    name: 'Getting Started',
    path: '/guide/getting-started',
  },
]
</script>

<template>
  <header py-2 px-4 bg-light dark:bg-dark z-10 container mx-auto sticky top-0>
    <nav flex my-3 gap-x-4 justify-end>
      <h1 font-playfair dark:text-brand mr-auto text-4xl>
        Dozzle
      </h1>
      <a
        icon-btn
        i-mdi-docker
        text-xl
        target="_blank"
        rel="noreferrer"
        href="https://hub.docker.com/r/amir20/dozzle/"
      />
      <a
        icon-btn
        i-mdi-github
        text-xl
        rel="noreferrer"
        href="https://github.com/amir20/dozzle"
        target="_blank"
        title="Dozzle GitHub"
      />
    </nav>
  </header>
  <div flex container mx-auto px-4>
    <aside>
      <nav w-64>
        <ul>
          <li v-for="item in menu" :key="item.path">
            <router-link :to="item.path">
              {{ item.name }}
            </router-link>
          </li>
        </ul>
      </nav>
    </aside>
    <main>
      <article prose text-lg max-w-full>
        <RouterView />
      </article>
    </main>
  </div>
</template>
