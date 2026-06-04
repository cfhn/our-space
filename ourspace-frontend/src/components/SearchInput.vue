<script setup lang="ts">
import { ref, watch } from 'vue'
import { OnyxIconButton, OnyxInput } from 'sit-onyx'
import { iconSearch, iconSearchX } from '@sit-onyx/icons'

const model = defineModel<string>()
const props = defineProps<{
  iconOpen?: string
  iconClose?: string
  placeholder?: string
}>()
const emit = defineEmits<{
  open: []
}>()

const extended = ref(false)

const onBlur = () => {
  if (!model.value) {
    extended.value = false
  }
}

const onClose = () => {
  model.value = ''
  extended.value = false
}

const onOpen = () => {
  extended.value = true
  emit('open')
}

watch(model, () => {
  if (model.value) {
    extended.value = true
  }
})
</script>

<template>
  <OnyxIconButton
    v-if="!extended"
    label="Search"
    :icon="props.iconOpen ?? iconSearch"
    mode="plain"
    density="compact"
    color="neutral"
    @click="onOpen"
    class="search-button"
  />
  <OnyxIconButton
    v-else
    label="Search"
    :icon="props.iconClose ?? iconSearchX"
    mode="plain"
    density="compact"
    color="neutral"
    @click="onClose"
    class="search-button"
  />
  <Transition>
    <OnyxInput
      v-if="extended"
      label="Search"
      type="search"
      v-model="model"
      density="compact"
      hideLabel
      hideClearIcon
      autofocus
      @blur="onBlur"
      class="search-input"
      :placeholder="placeholder"
    />
  </Transition>
</template>

<style scoped>
.search-input {
  width: 10rem;
  transition: width 0.125s;
}

.search-input.v-enter-from,
.search-input.v-leave-to {
  width: 0;
}
</style>
