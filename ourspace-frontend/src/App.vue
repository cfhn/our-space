<script setup lang="ts">
import {RouterView, useRoute} from 'vue-router';
import {useColorMode} from "@vueuse/core"
import {
  OnyxAppLayout,
  OnyxNavBar,
  OnyxNavItem,
  OnyxUserMenu,
  OnyxColorSchemeMenuItem,
  useThemeTransition
} from "sit-onyx";

const {store: colorScheme} = useColorMode({disableTransition: false});
useThemeTransition(colorScheme);

const route = useRoute();
</script>

<template>
  <OnyxAppLayout>
    <template #navBar>
      <OnyxNavBar app-name="OurSpace" v-if="route.meta.navbar ?? true">
        <OnyxNavItem label="Members" link="/members"></OnyxNavItem>
        <OnyxNavItem label="Cards" link="/cards"></OnyxNavItem>
        <template #contextArea>
          <OnyxUserMenu full-name="John Doe">
            <OnyxColorSchemeMenuItem v-model="colorScheme"/>
          </OnyxUserMenu>
        </template>
      </OnyxNavBar>
    </template>
    <RouterView />
  </OnyxAppLayout>
</template>

<style scoped>
</style>
