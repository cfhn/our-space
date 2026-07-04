<script setup lang="ts">
import { type Nullable, OnyxDatePicker, OnyxInput, OnyxStepper, OnyxTextarea } from 'sit-onyx'
import { computed } from 'vue'

const props = defineProps<{
  label: string
  isEdit: boolean
  type: 'text' | 'date' | 'password' | 'number' | 'textarea' | 'datetime-local'
}>()

const model = defineModel<string>()
const date = computed(() => {
  if (props.type !== 'date') {
    return ''
  }

  if (model.value === null) {
    return '-'
  }

  return new Date(model.value as string).toLocaleString()
})

const toNumber = (str?: string) => {
  if (!str) {
    return undefined
  }

  const num = Number.parseInt(str)
  if (Number.isNaN(num)) {
    return undefined
  }
}

const fromNumber = (num: Nullable<number>) => {
  if (!num) {
    return undefined
  }

  return num.toString()
}
</script>

<template>
  <OnyxInput
    :label="props.label"
    v-model="model"
    v-if="isEdit && (type === 'text' || type === 'password')"
    :type
    v-bind="$attrs"
    class="input"
  />
  <OnyxStepper
    :label="props.label"
    :modelValue="toNumber(model)"
    @update:modelValue="(num) => (model = fromNumber(num))"
    v-bind="$attrs"
    v-if="isEdit && type === 'number'"
    class="input"
  />
  <OnyxDatePicker
    type="datetime-local"
    :label="props.label"
    v-model="model"
    v-if="isEdit && type == 'date'"
    v-bind="$attrs"
    class="input"
  />
  <OnyxTextarea
    :label="props.label"
    v-model="model"
    v-if="isEdit && type === 'textarea'"
    v-bind="$attrs"
    class="input"
  />
  <div v-if="!isEdit" v-bind="$attrs">
    <p class="onyx-text--small label">{{ props.label }}</p>
    <p class="value" v-if="type === 'date'">{{ date }}</p>
    <p class="value" v-else-if="type === 'text' || type === 'textarea'">{{ model }}</p>
    <p class="value" v-else-if="type === 'password'">***</p>
    <p class="value numeric" v-else-if="type === 'number'">{{ model }}</p>
  </div>
</template>

<style scoped>
.label {
  color: var(--onyx-color-text-icons-neutral-medium);
  margin-top: var(--onyx-density-sm);
}

.value {
  padding-bottom: var(--onyx-density-xs);
  margin: calc(2 * var(--onyx-1px-in-rem)) 0;
}

.numeric {
  text-align: left;
}

.input {
  margin-top: var(--onyx-density-sm);
}
</style>
