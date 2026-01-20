import { type App } from "vue";
import urql, { cacheExchange, fetchExchange } from "@urql/vue";
import { withBase } from "@/stores/config";

export const install = (app: App) => {
  app.use(urql, {
    url: withBase("/api/graphql"),
    exchanges: [cacheExchange, fetchExchange],
  });
};
