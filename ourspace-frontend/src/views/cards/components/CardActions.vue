<script setup lang="ts">
import {OnyxIconButton} from "sit-onyx";
import editIcon from "@sit-onyx/icons/edit.svg?raw";
import deleteIcon from "@sit-onyx/icons/trash.svg?raw";
import {computed} from "vue";

const props = defineProps<{
  id: string
}>();

const link = computed(() => `/cards/${props.id}`);

</script>

<template>
  <div>
    <OnyxIconButton label="Edit" :icon="editIcon" color="neutral" density="compact" :link />
    <OnyxIconButton label="Delete" :icon="deleteIcon" color="neutral" density="compact" class="delete-button" />
  </div>
</template>

<style scoped>

</style>
