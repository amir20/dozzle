export type PayloadFormat = "slack" | "discord" | "ntfy" | "custom";

export const PAYLOAD_TEMPLATES: Record<PayloadFormat, string> = {
  slack: JSON.stringify(
    {
      text: "{{ .Container.Name }}",
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: "*{{ .Container.Name }}*\n{{ .Log.Message }}",
          },
        },
        {
          type: "context",
          elements: [
            {
              type: "mrkdwn",
              text: "Host: {{ .Container.Host }} | Image: {{ .Container.Image }}",
            },
          ],
        },
      ],
    },
    null,
    2,
  ),
  discord: JSON.stringify(
    {
      content: "{{ .Container.Name }}",
      embeds: [
        {
          title: "{{ .Container.Name }}",
          description: "{{ .Log.Message }}",
          fields: [
            { name: "Host", value: "{{ .Container.Host }}", inline: true },
            { name: "Image", value: "{{ .Container.Image }}", inline: true },
          ],
        },
      ],
    },
    null,
    2,
  ),
  ntfy: JSON.stringify(
    {
      topic: "dozzle-{{ .Container.Host }}",
      title: "{{ .Container.Name }}",
      message: "{{ .Log.Message }}",
    },
    null,
    2,
  ),
  custom: JSON.stringify(
    {
      container: "{{ .Container.Name }}",
      level: "{{ .Log.Level }}",
      message: "{{ .Log.Message }}",
    },
    null,
    2,
  ),
};
