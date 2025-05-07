<template>
  <aside>
    <header class="flex items-center gap-4">
      <material-symbols:terminal class="size-8" />
      <h1 class="text-2xl max-md:hidden">{{ container.name }}</h1>
      <h2 class="text-sm">Started <RelativeTime :date="container.created" /></h2>
    </header>

    <div class="mt-8 flex flex-col gap-2">
      <section>
        <div ref="host" class="shell"></div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import "@xterm/xterm/css/xterm.css";
const { container, action } = defineProps<{ container: Container; action: "attach" | "exec" }>();

const { Terminal } = await import("@xterm/xterm");
const { WebLinksAddon } = await import("@xterm/addon-web-links");

const host = useTemplateRef<HTMLDivElement>("host");
const terminal = new Terminal({
  cursorBlink: true,
  cursorStyle: "block",
});
terminal.loadAddon(new WebLinksAddon());

let ws: WebSocket | null = null;

onMounted(() => {
  terminal.open(host.value!);
  terminal.resize(100, 40);
  ws = new WebSocket(withBase(`/api/hosts/${container.host}/containers/${container.id}/${action}`));
  ws.onopen = () => {
    terminal.writeln(`Attaching to ${container.name} 🚀`);
    if (action === "attach") {
      ws?.send("\r");
    }
    terminal.onData((data) => {
      ws?.send(data);
    });
    terminal.focus();
  };
  ws.onmessage = (event) => terminal.write(event.data);
  ws.addEventListener("close", () => {
    terminal.writeln("⚠️ Connection closed");
  });
});

onUnmounted(() => {
  console.log("Closing WebSocket");
  terminal.dispose();
  ws?.close();
});
</script>
<style scoped>
@reference "@/main.css";

.shell {
  & :deep(.terminal) {
    @apply overflow-hidden rounded border-1 p-2;
    &:is(.focus) {
      @apply border-primary;
    }
  }

  & :deep(.xterm-viewport) {
    @apply bg-base-200!;
  }

  & :deep(.xterm-rows) {
    @apply text-base-content;
  }

  & :deep(.xterm-cursor-block.xterm-cursor-blink) {
    animation-name: blink !important;
  }
}

@keyframes blink {
  0% {
    background-color: var(--color-base-content);
    color: #000000;
  }

  50% {
    background-color: inherit;
    color: var(--color-base-content);
  }
}
</style>
