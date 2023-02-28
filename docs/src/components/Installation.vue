<template>
  <sub-section>
    <template #title>
      Installation
    </template>
    <p>
      Dozzle can be install using the Docker command line. Dozzle needs access to <code>docker.sock</code> file to
      access the logs of other containers. This file is usually located at <code>/var/run/docker.sock</code>. You can
      mount this file using the <code>-v</code> flag.
    </p>

    <code-block>
      docker run --detach --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle
    </code-block>
  </sub-section>
</template>
