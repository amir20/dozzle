<script setup lang="ts">
import { useHead } from '@vueuse/head'

import { isMobile } from '~/composables'

const menu = [
  {
    name: 'Introduction',
    subMenu: [
      {
        name: 'What is Dozzle?',
        path: '/guide/what-is-dozzle',
      },
      {
        name: 'Getting Started',
        path: '/guide/getting-started',
      },
    ],
  },
]
useHead({
  titleTemplate: '%s | Dozzle',
})
</script>

<template>
  <header py-2 px-4 bg-light dark:bg-dark z-10 container mx-auto sticky top-0>
    <nav flex my-3 gap-x-4 justify-end items-center>
      <a v-if="isMobile" text-2xl i-mdi-menu />
      <h1 font-playfair mr-auto text-4xl dark:text-brand>
        <a href="/">Dozzle</a>
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
  <div flex container mx-auto px-4 gap-4>
    <aside bg-light dark:bg-dark :class="{ fixed: isMobile }">
      <nav w-64>
        <ul>
          <li
            v-for="m in menu"
            :key="m.name"
          >
            <h2 class="text-lg font-bold">
              {{ m.name }}
            </h2>
            <ul mt-4 space-y-4 border="l-2 dark-50/50">
              <li v-for="item in m.subMenu" :key="item.path" pl-3>
                <router-link :to="item.path" active-class="text-teal-600" hover:text-teal-600>
                  {{ item.name }}
                </router-link>
              </li>
            </ul>
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
