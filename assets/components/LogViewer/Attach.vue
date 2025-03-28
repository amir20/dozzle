<template>
  <aside>
    <header class="flex items-center gap-4">
      <h1 class="text-2xl max-md:hidden">{{ container.name }}</h1>
      <h2 class="text-sm"><DistanceTime :date="container.created" /></h2>
    </header>

    <div class="mt-8 flex flex-col gap-2">
      <section>
        <div ref="terminal"></div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Container } from "@/models/Container";
import "@xterm/xterm/css/xterm.css";
const { container } = defineProps<{ container: Container }>();

const { Terminal } = await import("@xterm/xterm");

const terminal = useTemplateRef<HTMLDivElement>("terminal");
const term = new Terminal({
  cursorBlink: true,
  cursorStyle: "block",
});
const ws = new WebSocket(withBase(`/api/hosts/${container.host}/containers/${container.id}/attach`));

onMounted(() => {
  term.open(terminal.value!);
});

onUnmounted(() => {
  term.dispose();
  ws.close();
  console.log("WebSocket closed");
});

ws.onopen = () => {
  term.writeln("Attached to container 🚀");
  term.onData((data) => {
    ws.send(data);
  });
};

ws.onmessage = (event) => term.write(event.data);
</script>
<style scoped></style>
