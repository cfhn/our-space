<script setup lang="ts">
import {OnyxIconButton} from "sit-onyx";
import editIcon from "@sit-onyx/icons/edit.svg?raw";
import deleteIcon from "@sit-onyx/icons/trash.svg?raw";
import {computed} from "vue";

const props = defineProps<{
  id: string
}>();

defineEmits(['delete']);

const link = computed(() => `/members/${props.id}`);

</script>

<template>
  <div>
    <OnyxIconButton label="Edit" :icon="editIcon" color="neutral" density="compact" :link="link" />
    <OnyxIconButton label="Delete" :icon="deleteIcon" color="neutral" density="compact" class="delete-button" @click="$emit('delete')" />
  </div>
</template>

<style scoped>
.delete-button:hover {
  --icon-button-color: var(--onyx-color-text-icons-danger-intense)
}
</style>
