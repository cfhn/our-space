import "sit-onyx/global.css";
import "sit-onyx/style.css";
import "@fontsource-variable/source-code-pro";
import "@fontsource-variable/source-sans-3";
import './assets/main.css'

import {createApp} from 'vue'
import {createOnyx} from "sit-onyx";
import App from './App.vue'
import router from './router.ts'
import {client} from "./client/client.gen";
import {getToken, startRefreshTokenTask} from "@/auth/token.ts";
import {authGuard} from "@/auth/guard.ts";

client.setConfig({
  baseUrl: location.origin + "/api",
  auth: getToken,
})

startRefreshTokenTask();
router.beforeEach(authGuard);

const onyx = createOnyx({
  router: router,
})
const app = createApp(App)

app.use(onyx);
app.use(router)

app.mount('#app')
