import { type App } from "vue";
import { Autocomplete, Button, Dropdown, Switch, Skeleton, Field, Table, Modal, Config } from "@oruga-ui/oruga-next";
import { bulmaConfig } from "@oruga-ui/theme-bulma";

export const install = (app: App) => {
  app
    .use(Autocomplete)
    .use(Button)
    .use(Dropdown)
    .use(Switch)
    .use(Modal)
    .use(Field)
    .use(Skeleton)
    .use(Table)
    .use(Config, bulmaConfig);
};
