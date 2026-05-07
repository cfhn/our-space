<script setup lang="ts">
import {OnyxIconButton} from "sit-onyx";
import editIcon from "@sit-onyx/icons/edit.svg?raw";
import deleteIcon from "@sit-onyx/icons/trash.svg?raw";

defineEmits(['edit', 'delete']);
</script>

<template>
  <div>
    <OnyxIconButton label="Edit" :icon="editIcon" color="neutral" density="compact" @click="$emit('edit')" />
    <OnyxIconButton label="Delete" :icon="deleteIcon" color="neutral" density="compact" @click="$emit('delete')" />
  </div>
</template>

<style scoped>

</style>
