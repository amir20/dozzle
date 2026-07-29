<script lang="ts" setup>
const rows = [
  {
    cli: "One container at a time, per host.",
    dozzle: "Follow many containers side by side, merged across every host you connect.",
  },
  {
    cli: "Pipe to grep and start over when you get it wrong.",
    dozzle: "Search, filter by level, and jump to a timestamp while the logs keep streaming.",
  },
  {
    cli: "JSON and stack traces arrive as a wall of text.",
    dozzle: "Structured logs get parsed into fields, multi-line traces stay grouped.",
  },
  {
    cli: "Separate docker stats window for CPU and memory.",
    dozzle: "Live CPU, memory, and network charts on the same screen as the logs.",
  },
  {
    cli: "SSH into each machine to look around.",
    dozzle: "One browser tab over Docker, Swarm, and Kubernetes, with a shell when you need it.",
  },
];
</script>

<template>
  <section class="mx-auto mt-20 max-w-[1152px] px-6">
    <h2 class="m-0! text-center text-2xl font-semibold text-(--vp-c-text-1) md:text-3xl">
      Why not just <code class="bg-transparent! p-0! font-mono">docker logs</code>?
    </h2>
    <p class="mx-auto mt-3! mb-0! max-w-2xl text-center text-(--vp-c-text-2)">
      It works fine until you are tailing three containers on two hosts at 2am.
    </p>

    <div class="mt-10 grid gap-4 md:grid-cols-2">
      <div class="rounded-xl border border-solid border-(--vp-c-divider) bg-(--vp-c-bg-alt) p-6">
        <div class="flex items-center gap-2 text-sm font-semibold tracking-wide text-(--vp-c-text-3) uppercase">
          <Icon icon="mdi:console" class="size-4" />
          docker logs
        </div>
        <ul class="mt-4! mb-0! flex list-none flex-col gap-3 p-0!">
          <li v-for="row in rows" :key="row.cli" class="flex gap-2.5 text-(--vp-c-text-2)">
            <Icon icon="mdi:close-circle-outline" class="mt-1 size-4 shrink-0 text-(--vp-c-text-3)" />
            <span>{{ row.cli }}</span>
          </li>
        </ul>
      </div>

      <div class="rounded-xl border border-solid border-(--vp-c-brand-dimm) bg-(--vp-c-brand-dimm) p-6">
        <div class="flex items-center gap-2 text-sm font-semibold tracking-wide text-(--vp-c-brand) uppercase">
          <Icon icon="mdi:television-play" class="size-4" />
          Dozzle
        </div>
        <ul class="mt-4! mb-0! flex list-none flex-col gap-3 p-0!">
          <li v-for="row in rows" :key="row.dozzle" class="flex gap-2.5 text-(--vp-c-text-1)">
            <Icon icon="mdi:check-circle-outline" class="mt-1 size-4 shrink-0 text-(--vp-c-brand)" />
            <span>{{ row.dozzle }}</span>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>
