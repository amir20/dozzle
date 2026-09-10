import type { ViewContext } from "@/composable/viewContext";

export type ChatMessage = {
  role: "user" | "assistant";
  text: string;
  /** The context the turn was asked with, shown above a user message so the
   *  thread records what the assistant was looking at when it answered. */
  view?: ViewContext;
  error?: boolean;
};

/**
 * One assistant thread, shared across the app.
 *
 * The pane is persistent, so "initiating" only happens once: after that the
 * user changes what they are asking about by navigating. That is why the thread
 * lives here rather than inside the pane component, and why closing the pane
 * does not end the conversation.
 */
const open = ref(false);
const messages = ref<ChatMessage[]>([]);
const status = ref("");
const streaming = ref(false);

export function useCloudChat() {
  function openPane() {
    open.value = true;
  }

  function closePane() {
    // The thread survives. Reopening into a blank box loses whatever you were
    // half way through.
    open.value = false;
  }

  async function ask(message: string, view: ViewContext) {
    if (!message.trim() || streaming.value) return;

    messages.value.push({ role: "user", text: message, view });
    const reply = reactive<ChatMessage>({ role: "assistant", text: "" });
    messages.value.push(reply);
    streaming.value = true;
    status.value = "";

    try {
      const res = await fetch(withBase("/api/cloud/chat"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message, view }),
      });

      if (!res.ok || !res.body) {
        reply.text = "The assistant is unavailable right now.";
        reply.error = true;
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        // SSE frames are separated by a blank line; each carries one JSON event.
        const frames = buffer.split("\n\n");
        buffer = frames.pop() ?? "";
        for (const frame of frames) {
          const line = frame.split("\n").find((l) => l.startsWith("data: "));
          if (!line) continue;
          const event = JSON.parse(line.slice(6));
          if (event.kind === "delta") {
            reply.text += event.text;
            status.value = "";
          } else if (event.kind === "status") {
            status.value = event.text;
          } else if (event.kind === "error") {
            reply.text = event.text || "Something went wrong.";
            reply.error = true;
          }
        }
      }
    } catch {
      reply.text = "The assistant is unavailable right now.";
      reply.error = true;
    } finally {
      streaming.value = false;
      status.value = "";
    }
  }

  return { open, messages, status, streaming, ask, openPane, closePane };
}
