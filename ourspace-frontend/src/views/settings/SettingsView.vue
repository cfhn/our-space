<script setup lang="ts">

import {OnyxPageLayout} from "sit-onyx";
import MemberAttributes from "@/views/settings/components/MemberAttributes.vue";
</script>

<template>
  <OnyxPageLayout>
    <h1 class="headline">Settings</h1>
    <div class="settings-section">
      <MemberAttributes/>
    </div>
  </OnyxPageLayout>
</template>

<style scoped>
.settings-section {
  margin-top: var(--onyx-density-lg);
  margin-bottom: var(--onyx-density-lg);
}
</style>
