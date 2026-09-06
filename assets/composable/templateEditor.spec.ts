import { describe, expect, test } from "vitest";
import { payloadMode } from "./templateEditor";
import { PAYLOAD_TEMPLATES } from "@/components/Notification/payloadTemplates";

// payloadMode has to agree with executeJSONTemplate in internal/notification/dispatcher/webhook.go,
// which unmarshals the raw template text (actions still in place) and only falls back to running
// the whole body through text/template when that fails.
describe("payloadMode", () => {
  test("every shipped preset is valid JSON with its actions in place", () => {
    for (const [format, template] of Object.entries(PAYLOAD_TEMPLATES)) {
      expect(payloadMode(template), format).toBe("json");
    }
  });

  test("actions inside string values keep the template JSON", () => {
    expect(payloadMode('{"text": "{{ .Detail }}"}')).toBe("json");
  });

  test("an action used as a bare value falls back to raw text", () => {
    // Valid Go template, not valid JSON — the backend renders the whole body instead.
    expect(payloadMode('{"count": {{ .Count }}}')).toBe("raw");
  });

  test("plain text is raw", () => {
    expect(payloadMode("{{ .Container.Name }} went down")).toBe("raw");
  });

  test("an unclosed action is reported regardless of JSON validity", () => {
    expect(payloadMode('{"text": "{{ .Detail "}')).toBe("unbalanced");
    expect(payloadMode("{{ .Detail")).toBe("unbalanced");
  });

  test("blank templates report empty", () => {
    expect(payloadMode("")).toBe("empty");
    expect(payloadMode("   \n ")).toBe("empty");
  });
});
