import { type App } from "vue";
import urql, { cacheExchange, fetchExchange, Client } from "@urql/vue";
import { withBase } from "@/stores/config";

export const client = new Client({
  url: withBase("/api/graphql"),
  exchanges: [cacheExchange, fetchExchange],
});

export const install = (app: App) => {
  app.use(urql, client);
};
