import { type App } from "vue";
import { Autocomplete, Button, Dropdown, Skeleton, Field, Modal, Config } from "@oruga-ui/oruga-next";
import { bulmaConfig } from "@oruga-ui/theme-bulma";

export const install = (app: App) => {
  app
    .use(Autocomplete)
    .use(Button)
    .use(Dropdown)
    .use(Modal)
    .use(Field)
    .use(Skeleton)
    .use(Config, bulmaConfig);
};
