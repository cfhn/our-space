<script setup lang="ts">
import { OnyxIconButton } from 'sit-onyx'
import editIcon from '@sit-onyx/icons/edit.svg?raw'
import deleteIcon from '@sit-onyx/icons/trash.svg?raw'
</script>

<template>
  <OnyxIconButton
    label="Edit"
    :icon="editIcon"
    color="neutral"
    density="compact"
    disabled
    title="Not yet supported"
  />
  <OnyxIconButton
    label="Delete"
    :icon="deleteIcon"
    color="neutral"
    density="compact"
    disabled
    title="Not yet supported"
  />
</template>

<style scoped></style>
