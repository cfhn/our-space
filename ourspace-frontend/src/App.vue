<script setup lang="ts">
import {RouterView, useRoute} from 'vue-router';
import {useColorMode} from "@vueuse/core"
import {
  OnyxAppLayout,
  OnyxNavBar,
  OnyxNavItem,
  OnyxMenuItem,
  OnyxUserMenu,
  OnyxColorSchemeMenuItem,
  OnyxIcon,
  useThemeTransition, OnyxToast, OnyxButton
} from "sit-onyx";
import {isExpired, logout, userStore, useToken} from "@/auth/token.ts";
import {computed} from "vue";
import iconLogout from "@sit-onyx/icons/logout.svg?raw";

const {store: colorScheme} = useColorMode({disableTransition: false});
useThemeTransition(colorScheme);

const route = useRoute();
const token = useToken();
const loggedIn = computed(() => !!token.value && !isExpired(token.value));
const user = userStore();
</script>

<template>
  <OnyxAppLayout>
    <template #navBar>
      <OnyxNavBar app-name="OurSpace" v-if="route.meta.navbar ?? true">
        <OnyxNavItem label="Members" link="/members"></OnyxNavItem>
        <OnyxNavItem label="Cards" link="/cards"></OnyxNavItem>
        <template #contextArea>
          <OnyxUserMenu v-if="loggedIn" :full-name="user.fullName">
            <OnyxColorSchemeMenuItem v-model="colorScheme"/>
            <OnyxMenuItem color="danger" @click="logout()">
              <OnyxIcon :icon="iconLogout"/>
              Logout
            </OnyxMenuItem>
          </OnyxUserMenu>
          <OnyxButton v-else label="Log in"></OnyxButton>
        </template>
      </OnyxNavBar>
    </template>
    <RouterView/>
    <OnyxToast/>
  </OnyxAppLayout>
</template>

<style scoped>
</style>
