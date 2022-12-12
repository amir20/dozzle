const { hostname } = config;
let subtitle = $ref("");
const title = $computed(() => `${subtitle} - Dozzle` + (hostname ? ` @ ${hostname}` : ""));

useTitle($$(title));

export function setTitle(t: string) {
  subtitle = t;
}
