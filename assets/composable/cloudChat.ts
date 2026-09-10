import type { ViewContext, ViewLogLine } from "@/composable/viewContext";
import { i18n } from "@/modules/i18n";

const { t } = i18n.global;

/** A failure the reader sees, so it is phrased in their language. Cloud codes
 *  we do not know about fall back to whatever prose came with them. */
function errorText(code?: string, text?: string) {
  if (code === "not_configured") return t("cloud-chat.error-not-configured");
  if (code === "unavailable" || !text) return t("cloud-chat.error-unavailable");
  return text;
}

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
 * The panel is persistent, so "initiating" only happens once: after that the
 * user changes what they are asking about by navigating. That is why the thread
 * lives here rather than inside the panel component, and why closing the panel
 * does not end the conversation. Which panel is open is the rail's business.
 */
const messages = ref<ChatMessage[]>([]);
const status = ref("");
// What Dozzle is doing, as a token it sends instead of a sentence: the server
// has no locale, so the phrase is picked in the browser. Cloud's own status
// lines arrive as prose and land in `status` instead.
const activity = ref("");
const streaming = ref(false);
// The line the user pointed at, held until it is sent. Asking "why?" from a log
// row means that row, and nothing about the composer says so on its own.
const focused = ref<ViewLogLine>();

export function useCloudChat() {
  const { openRail, closeRail } = useCloudRail();

  function openPane() {
    openRail("chat");
  }

  /** Opens the assistant with one line attached, from a log row's menu. */
  function askAboutLine(line: ViewLogLine) {
    focused.value = line;
    openRail("chat");
  }

  function clearFocus() {
    focused.value = undefined;
  }

  function closePane() {
    // The thread survives. Reopening into a blank box loses whatever you were
    // half way through.
    closeRail();
  }

  async function ask(message: string, context: ViewContext) {
    if (!message.trim() || streaming.value) return;

    // The focus belongs to the turn that was composed with it on screen, not to
    // the thread: the next question is about whatever the user is looking at
    // then.
    const view: ViewContext = focused.value ? { ...context, focused: focused.value } : context;
    focused.value = undefined;

    messages.value.push({ role: "user", text: message, view });
    const reply = reactive<ChatMessage>({ role: "assistant", text: "" });
    messages.value.push(reply);
    streaming.value = true;
    status.value = "";
    activity.value = "";

    try {
      const res = await fetch(withBase("/api/cloud/chat"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message, view }),
      });

      if (!res.ok || !res.body) {
        reply.text = errorText();
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
            activity.value = "";
          } else if (event.kind === "reset") {
            // The model fell back, or a round turned into a tool call. What has
            // been shown is no longer part of the answer.
            reply.text = "";
          } else if (event.kind === "status") {
            // One or the other, never both stacked: whichever arrived last is
            // what is happening now.
            status.value = event.text ?? "";
            activity.value = event.activity ?? "";
          } else if (event.kind === "error") {
            reply.text = errorText(event.code, event.text);
            reply.error = true;
          }
        }
      }
    } catch {
      reply.text = errorText();
      reply.error = true;
    } finally {
      streaming.value = false;
      status.value = "";
      activity.value = "";
    }
  }

  return { messages, status, activity, streaming, focused, ask, askAboutLine, clearFocus, openPane, closePane };
}
