const { hostname } = config;
const subtitle = ref("");
const title = computed(
  () => (subtitle.value ? `${subtitle.value} - ` : "") + "Dozzle" + (hostname ? ` @ ${hostname}` : ""),
);

useTitle(title);

export function setTitle(t: string) {
  subtitle.value = t;
}
