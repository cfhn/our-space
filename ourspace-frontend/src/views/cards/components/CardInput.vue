<script setup lang="ts">
import { OnyxButton, OnyxModal, OnyxBottomBar, OnyxLoadingIndicator, OnyxIcon } from 'sit-onyx'
import { ref } from 'vue'
import check from '@sit-onyx/icons/check.svg?raw'

const modelValue = defineModel<string>()

const isOpen = ref(false)
const waiting = ref(true)
const inputValue = ref('')

const onSubmit = () => {
  waiting.value = false
  modelValue.value = inputValue.value
  inputValue.value = ''

  setTimeout(() => {
    isOpen.value = false
    waiting.value = true
  }, 500)
}
</script>

<template>
  <OnyxButton label="Scan Card" @click="isOpen = true" v-bind="$attrs" />
  <OnyxModal label="Scan Card" :open="isOpen" @close="isOpen = false">
    <template #default>
      <div class="loading-indicator">
        <OnyxLoadingIndicator type="circle" v-if="waiting" />
        <OnyxIcon v-else :icon="check" color="success" />
      </div>
      <div class="modal">
        Ensure a NFC Scanner is connected. Then hold the card onto the scanner.
      </div>
      <form @submit.prevent="onSubmit">
        <input
          type="text"
          class="nfc-input"
          aria-label="NFC Scan value"
          v-model="inputValue"
          autofocus
          @focusout="isOpen = false"
        />
      </form>
    </template>

    <template #footer>
      <OnyxBottomBar>
        <OnyxButton label="Close" color="neutral" mode="plain" @click="isOpen = false" />
      </OnyxBottomBar>
    </template>
  </OnyxModal>
</template>

<style scoped>
.modal {
  padding: var(--onyx-density-xl) var(--onyx-modal-dialog-padding-inline);
  color: var(--onyx-color-text-icons-neutral-medium);
}

.loading-indicator {
  display: flex;
  justify-content: center;
  margin: var(--onyx-density-xl);
}

.nfc-input {
  height: 0;
  border: none;
  margin: 0;
  padding: 0;
  outline: none;
}
</style>
