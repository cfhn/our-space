import {reactive, readonly, ref} from "vue";
import {authServiceLogout, authServiceRefresh} from "@/client";
import router from "@/router.ts";

const REFRESH_INTERVAL = 1000 * 60;

const token = ref<string>();
const userStoreRef = reactive({
  fullName: "",
});

export const getToken = () => {
  return token.value;
}

export const useToken = () => {
  return readonly(token);
}

export const userStore = () => {
  return readonly(userStoreRef);
}

export const setToken = (value: string) => {
  token.value = value;
  sessionStorage.setItem("access-token", value);

  processToken(value);
}

const loadToken = () => {
  const storageToken = sessionStorage.getItem("access-token");
  if (!storageToken) {
    return;
  }

  token.value = storageToken;
  processToken(storageToken);
}

const processToken = (token: string) => {
  const parts = token.split(".");
  if (parts.length != 3) {
    return;
  }
  const claims = JSON.parse(base64Decode(parts[1]));

  userStoreRef.fullName = claims["full_name"];
}

export const startRefreshTokenTask = () => {
  loadToken();

  if (isExpired(getToken() ?? "")) {
    refresh();
  }

  const refreshWrapper = async () => {
    await refresh();
    setTimeout(refreshWrapper, REFRESH_INTERVAL);
  }

  setTimeout(refreshWrapper, REFRESH_INTERVAL);
}

const refresh = async () => {
  const response = await authServiceRefresh({
    body: {},
    credentials: "include",
  });
  if (response.error || !response.data.success || !response.data.success.access_token) {
    if (router.currentRoute.value.meta["authenticated"] === false) {
      return;
    }

    return router.push("/login");
  }

  setToken(response.data.success.access_token);
}

export const isExpired = (token: string) => {
  const parts = token.split(".");
  if (parts.length < 3) {
    return true;
  }

  const claims = JSON.parse(atob(parts[1]));
  if (!("exp" in claims) || typeof claims.exp !== "number") {
    return false;
  }

  return (claims["exp"] * 1000) < Date.now();
}

export const isLoggedIn = (): boolean => {
  const token = getToken();
  if (!token) {
    return false;
  }

  return !isExpired(token);
}

export const logout = async () => {
  sessionStorage.removeItem("access-token");
  token.value = undefined;

  await authServiceLogout({
    body: {},
  });
  await router.push({name: "login"})
}

function base64Decode(str: string): string {
  return decodeURIComponent(atob(str).split('').map(function(c) {
    return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
  }).join(''));
}
