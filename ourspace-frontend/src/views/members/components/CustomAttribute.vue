<script setup lang="ts">
import type { MemberAttribute } from '@/client'
import DynamicInput from '@/components/DynamicInput.vue'
import { computed } from 'vue'
import { OnyxIconButton } from 'sit-onyx'
import { iconTrash } from '@sit-onyx/icons'

const props = defineProps<{
  attribute: MemberAttribute
  isEdit: boolean
}>()

const model = defineModel<string>()

const emit = defineEmits<{
  delete: []
}>()

const determineType = (str?: string) => (str?.includes('\n') ? 'textarea' : 'text')

const typeLookup = {
  TYPE_UNKNOWN: 'text',
  TYPE_TEXT_SINGLE_LINE: 'text',
  TYPE_TEXT_MULI_LINE: 'textarea',
  TYPE_NUMBER: 'number',
  TYPE_DATE: 'date',
  TYPE_DATETIME: 'datetime-local',
} as const

const inputType = computed(() => typeLookup[props.attribute.type] ?? determineType(model.value))
</script>

<template>
  <div class="attribute">
    <div class="attribute-value">
      <DynamicInput
        :label="attribute.display_name"
        :is-edit="isEdit"
        :type="inputType"
        v-model="model"
      />
    </div>
    <OnyxIconButton
      label="Delete"
      :icon="iconTrash"
      color="danger"
      type="button"
      v-if="isEdit"
      @click="emit('delete')"
    />
  </div>
</template>

<style scoped>
.attribute {
  display: flex;
  flex-direction: row;
  justify-content: start;
  align-items: end;
}

.attribute-value {
  flex-grow: 1;
}
</style>
