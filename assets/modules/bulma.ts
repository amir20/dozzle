import { type App } from "vue";
import {
  Autocomplete,
  Button,
  Dropdown,
  Switch,
  Radio,
  Skeleton,
  Field,
  Tooltip,
  Modal,
  Config,
} from "@oruga-ui/oruga-next";
import { bulmaConfig } from "@oruga-ui/theme-bulma";

export const install = (app: App) => {
  app
    .use(Autocomplete)
    .use(Button)
    .use(Dropdown)
    .use(Switch)
    .use(Tooltip)
    .use(Modal)
    .use(Radio)
    .use(Field)
    .use(Skeleton)
    .use(Config, bulmaConfig);
};
