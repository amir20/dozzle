<template>
  <header py-2 px-4 bg-light dark:bg-dark z-10>
    <nav flex my-3 gap-x-4 justify-end>
      <a
        icon-btn
        i-mdi-docker
        target="_blank"
        rel="noreferrer"
        href="https://hub.docker.com/r/amir20/dozzle/"
      />
      <a
        icon-btn
        i-mdi-github
        rel="noreferrer"
        href="https://github.com/amir20/dozzle"
        target="_blank"
        title="Dozzle GitHub"
      />
    </nav>
  </header>
  <main text="gray-700 dark:gray-200" mb-22>
    <RouterView />
  </main>
  <footer class="mx-auto mt-32 w-full px-4 sm:px-6 lg:px-8">
    <div class="py-10">
      <p class="mt-5 text-center text-sm leading-6 text-gray-400">
        Made with ❤️ by <a href="https://amirraminfar.me">Amir Raminfar</a>.
      </p>
      <nav flex my-3 gap-x-4 justify-center>
        <a
          icon-btn
          i-mdi-docker
          target="_blank"
          rel="noreferrer"
          href="https://hub.docker.com/r/amir20/dozzle/"
        />
        <a
          icon-btn
          i-mdi-github
          rel="noreferrer"
          href="https://github.com/amir20/dozzle"
          target="_blank"
          title="Dozzle GitHub"
        />
      </nav>
    </div>
  </footer>
</template>
