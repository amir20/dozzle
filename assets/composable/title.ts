const { hostname } = config;
let subtitle = $ref("");
const title = $computed(() => (subtitle ? `${subtitle} - ` : "") + "Dozzle" + (hostname ? ` @ ${hostname}` : ""));

useTitle($$(title));

export function setTitle(t: string) {
  subtitle = t;
}
