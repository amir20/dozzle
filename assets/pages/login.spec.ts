/**
 * @vitest-environment jsdom
 */
import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, test, vi } from "vitest";
import { createI18n } from "vue-i18n";

const config = vi.hoisted(() => ({
  base: "",
  passwordLogin: false as boolean | undefined,
  oauthProviders: [] as { name: string; loginUrl: string; icon: string }[],
}));
vi.mock("@/stores/config", () => ({ __esModule: true, default: config, withBase: (p: string) => p }));

import Login from "./login.vue";

const sso = { name: "Keycloak", loginUrl: "/api/auth/login?provider=oidc", icon: "mdi:shield-account" };
const github = { name: "GitHub", loginUrl: "/api/auth/login?provider=github", icon: "mdi:github" };

const i18n = createI18n({ legacy: false, locale: "en", missingWarn: false, fallbackWarn: false, messages: { en: {} } });

function visit(search: string) {
  const replace = vi.fn();
  vi.stubGlobal("location", { search, origin: "http://localhost", replace });
  mount(Login, { global: { plugins: [i18n] } });
  return replace;
}

afterEach(() => {
  vi.unstubAllGlobals();
  sessionStorage.clear();
  config.passwordLogin = false;
  config.oauthProviders = [];
});

describe("login page", () => {
  test("goes straight to the only provider, keeping redirectUrl", () => {
    config.oauthProviders = [sso];
    const replace = visit("?redirectUrl=%2Fcontainer%2Fabc");
    expect(replace).toHaveBeenCalledWith("/api/auth/login?provider=oidc&redirectUrl=%2Fcontainer%2Fabc");
  });

  test("stays after a failed callback", () => {
    config.oauthProviders = [sso];
    expect(visit("?error=oauth")).not.toHaveBeenCalled();
  });

  test("stays on ?direct", () => {
    config.oauthProviders = [sso];
    expect(visit("?direct=1")).not.toHaveBeenCalled();
  });

  test("stays once right after logout, then redirects again", () => {
    config.oauthProviders = [sso];
    markLoggedOut();
    expect(visit("")).not.toHaveBeenCalled();
    expect(visit("")).toHaveBeenCalledOnce();
  });

  test("stays when there is a password form", () => {
    config.passwordLogin = true;
    config.oauthProviders = [github];
    expect(visit("")).not.toHaveBeenCalled();
  });

  test("stays when there is more than one provider to choose from", () => {
    config.oauthProviders = [sso, github];
    expect(visit("")).not.toHaveBeenCalled();
  });
});
