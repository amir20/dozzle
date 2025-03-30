<template>
  <aside>
    <header class="flex items-center gap-4">
      <h1 class="text-2xl max-md:hidden">{{ container.name }}</h1>
      <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
    </header>

    <div class="mt-8 flex flex-col gap-2">
      <section>
        <div ref="terminal" class="shell"></div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import "@xterm/xterm/css/xterm.css";
const { container, action } = defineProps<{ container: Container; action: "attach" | "exec" }>();

const { Terminal } = await import("@xterm/xterm");

const terminal = useTemplateRef<HTMLDivElement>("terminal");
const term = new Terminal({
  cursorBlink: true,
  cursorStyle: "block",
});
let ws: WebSocket | null = null;

onMounted(() => {
  term.open(terminal.value!);
  term.resize(100, 40);
  ws = new WebSocket(withBase(`/api/hosts/${container.host}/containers/${container.id}/${action}`));
  ws.onopen = () => {
    term.writeln(`Attached to ${container.name} 🚀`);
    if (action === "attach") {
      ws?.send("\r");
    }
    term.onData((data) => {
      ws?.send(data);
    });
    term.focus();
  };
  ws.onmessage = (event) => term.write(event.data);
});

onUnmounted(() => {
  console.log("Closing WebSocket");
  term.dispose();
  ws?.close();
});
</script>
<style scoped>
@import "@/main.css" reference;

.shell {
  & :deep(.terminal) {
    @apply border-primary overflow-hidden rounded border-1 p-2;
  }

  & :deep(.xterm-viewport) {
    @apply bg-base-200!;
  }

  & :deep(.xterm-rows) {
    @apply text-base-content;
  }
}
</style>
