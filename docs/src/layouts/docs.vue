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

const showMenu = ref(false)
</script>

<template>
  <header py-2 lg:px-4 bg-light dark:bg-dark container mx-auto sticky top-0>
    <nav flex my-3 gap-x-4 justify-end items-center>
      <a v-if="isMobile" cursor-pointer text-2xl i-mdi-menu @click="showMenu = !showMenu" />
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
    <Teleport to="body" :disabled="!isMobile">
      <div v-if="showMenu" backdrop-blur-sm inset-0 fixed @click="showMenu = false" />
      <aside class="transition-transform lg:translate-x-0" main-bg :class="isMobile ? 'fixed inset-y-0 left-0 p-4' : ''" :translate-x="showMenu ? 0 : -64">
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
    </Teleport>
    <main>
      <article prose text-lg max-w-full>
        <RouterView />
      </article>
    </main>
  </div>
</template>
