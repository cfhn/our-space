<script setup lang="ts">
import type { CardReadable } from '@/client'
import DynamicInput from '@/components/DynamicInput.vue'
import CardInput from '@/views/cards/components/CardInput.vue'
import MemberSelect from '@/views/members/components/MemberSelect.vue'
import { computed, watchEffect } from 'vue'
import { base64ToHex } from '@/views/cards/card-utilities.ts'

const props = defineProps<{
  isEdit: boolean
  memberId?: string
}>()

const card = defineModel<CardReadable>('card', { required: true })

const cardRfidValue = computed(() => {
  console.log(card.value)
  return base64ToHex(card.value.rfid_value)
})

watchEffect(() => {
  console.log(cardRfidValue.value)
})

watchEffect(() => {
  if (props.isEdit && props.memberId) {
    card.value.member_id = props.memberId
  }
})
</script>

<template>
  <div class="onyx-grid">
    <MemberSelect
      class="onyx-grid-span-12"
      label="Assigned to"
      :is-edit="isEdit"
      v-model="card.member_id"
    />
    <DynamicInput
      class="onyx-grid-span-6"
      type="date"
      v-model="card.valid_from"
      :is-edit="isEdit"
      label="Valid From"
    />
    <DynamicInput
      class="onyx-grid-span-6"
      type="date"
      v-model="card.valid_to"
      :is-edit="isEdit"
      label="Valid To"
    />
    <div class="onyx-grid-span-12 card-input">
      <DynamicInput
        type="text"
        label="RFID Value"
        :modelValue="cardRfidValue"
        :is-edit="isEdit"
        readonly
      />
      <CardInput v-model="card.rfid_value" v-if="isEdit" />
    </div>
  </div>
</template>

<style scoped>
.card-input {
  display: flex;
  flex-direction: row;
  align-items: flex-end;
  gap: var(--onyx-spacing-sm);
}

.card-input:deep(button) {
  margin-bottom: 2px;
}
</style>
