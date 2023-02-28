<template>
  <sub-section>
    <template #title> What's Dozzle </template>
    <p>
      Dozzle is a small Docker container which provides a web-based interface for viewing logs of other Docker
      containers running on the same system.
    </p>
    <p>Dozzle is open sourced and officially sponsored by Docker OSS.</p>

    <p>
      <strong>Real-time log streaming</strong> — allows you to view logs of other Docker containers in real-time. As new
      log entries are generated, they are streamed to the web interface without needing to refresh the page.
    </p>

    <p><strong>Live CPU and memory usage</strong> — see historical memory and cpu usage to easily see memory trend.</p>

    <p>
      <strong>Search across many containers</strong> — quickly search containers using <key-shortcut label="k" /> to
      find logs for a specifc container.
    </p>

    <p>
      <strong>Color-coded logs</strong> — log entries are color-coded and grouped based on their level (error, warning,
      info, etc.) to help you quickly identify issues.
    </p>

    <p>
      <strong>Search</strong> — you can search logs for specific keywords or <code>/regex/</code> making it easy to find
      relevant log entries.
    </p>

    <p>
      <strong>Split screen</strong> — able to view muliple logs at the same by spliting the screen between multiple
      containers.
    </p>

    <p>
      <strong>Simple Authentication</strong> — simple username and password authentication supported out of the box.
    </p>

    <p>
      <strong>Lightweight</strong> — a lightweight application written in Go which doesn't consume a lot of memory or
      cpu. It can be run alongside other containers without causing performance issues.
    </p>
  </sub-section>
</template>
