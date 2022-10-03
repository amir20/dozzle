import "./styles.scss";
import { createApp, App as VueApp } from "vue";
import App from "./App.vue";

const app = createApp(App);
Object.values(import.meta.glob<{ install: (app: VueApp) => void }>("./modules/*.ts", { eager: true })).forEach((i) =>
  i.install?.(app)
);

app.mount("#app");

// const test = ref({ a: 1, b: 2 });

// const history = useThrottledRefHistory(test, {
//   deep: true,
//   throttle: 1000,
//   capacity: 300,
// });

// watch(history.last, (newStat) => {
//   console.log("newStats", newStat);
// }, { deep: true });



// setInterval(() => {
//   test.value.a++;
//   console.log("test", history.history.value[0].snapshot);
// }, 500);
