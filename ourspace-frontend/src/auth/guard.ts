import type {RouteLocationNormalized} from "vue-router";
import {isLoggedIn} from "@/auth/token.ts";

export const authGuard = (to: RouteLocationNormalized, from: RouteLocationNormalized) => {
  if (isLoggedIn()) {
    if (to.name === "login") {
      return {name: "home"};
    }

    return;
  }

  if (to.meta.authenticated !== false) {
    return {name: "login"};
  }
};
