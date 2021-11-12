/* eslint-disable */
declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<{}, {}, any>;
  export default component;
}

declare module "eventsourcemock" {
  import type { EventSource } from "eventsource";
  export default class extends EventSource {}
}
