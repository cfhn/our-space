<script setup lang="ts">
import {OnyxBadge} from "sit-onyx";

const props = defineProps<{
  tags: string[];
}>()
</script>

<template>
<div>
  <OnyxBadge v-for="tag in props.tags"
             color="neutral"
             density="compact"
             class="tag-badge"
  >{{tag}}</OnyxBadge>
</div>
</template>

<style scoped>
.tag-badge {
  margin-right: 2px;
}
</style>
