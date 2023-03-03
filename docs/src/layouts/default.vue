<template>
  <header py-2 px-4 bg-light dark:bg-dark z-10>
    <nav flex my-3 gap-x-4 justify-end>
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
  <RouterView />
</template>
