<script setup lang="ts">

import {
  OnyxPageLayout,
  OnyxCard,
  OnyxForm,
  OnyxInput,
  OnyxButton,
  useToast,
  OnyxToast
} from "sit-onyx";
import {authServiceLogin} from "@/client";
import {ref} from "vue";
import {useRouter} from "vue-router";
import {setToken} from "@/auth/token.ts";

const SSO_ENABLED = false;
const toast = useToast();
const router = useRouter();

const username = ref<string>("");
const password = ref<string>("");
const loading = ref(false);

const wrapLoading = (f: () => Promise<void>): (() => Promise<void>) => {
  return async () => {
    loading.value = true;
    await f();
    loading.value = false;
  }
}

const handleSubmit = wrapLoading(async () => {
  const response = await authServiceLogin({
    body: {
      password: {
        username: username.value,
        password: password.value,
      }
    },
    credentials: "include",
  });

  if (response.error || response.data.success === undefined || response.data.success.access_token === undefined) {
    console.log("Login failed, showing toast", response.error, response.data?.success, response.data?.success?.access_token);
    toast.show({
      color: "warning",
      headline: "Authentication failed",
      description: "Please check your credentials and try again.",
    });
    username.value = "";
    password.value = "";
    return;
  }

  setToken(response.data.success.access_token);

  const previousRoute = sessionStorage.getItem("login-previous-route");
  if (previousRoute) {
    router.push(previousRoute);
  } else {
    router.push({name: "home"});
  }
});
</script>

<template>
  <OnyxPageLayout>
    <OnyxCard class="login-card">
      <h1>Login</h1>
      <OnyxForm @submit.prevent="handleSubmit" class="login-form">
        <OnyxInput type="text" label="Username" autofocus v-model="username"></OnyxInput>
        <OnyxInput type="password" label="Password" v-model="password"></OnyxInput>
        <div class="login-form-actions">
          <OnyxButton type="submit" label="Login" class="login-button"/>
        </div>
      </OnyxForm>
      <div class="login-other-actions" v-if="SSO_ENABLED">
        <hr>
        <OnyxButton type="button" label="Single Sign On" color="neutral"/>
      </div>
    </OnyxCard>
  </OnyxPageLayout>
</template>

<style scoped>
.login-card {
  width: 24rem;
  margin: 0 auto;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;

  &-form-actions {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: .25rem;
  }
}

.login-button {
  width: 100%;
}

hr {
  border: none;
  border-bottom: 1px solid var(--onyx-color-component-border-neutral);
}

.login-other-actions .onyx-button {
  width: 100%;
}
</style>
