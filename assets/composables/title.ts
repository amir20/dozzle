let subtitle = $ref("");
const title = $computed(() => `${subtitle} - Dozzle`);

useTitle($$(title));

export function setTitle(t: string) {
  subtitle = t;
}
