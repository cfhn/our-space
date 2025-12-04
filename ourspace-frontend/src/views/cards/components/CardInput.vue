<script setup lang="ts">

import {
  OnyxButton,
  OnyxModal,
  OnyxBottomBar,
  OnyxLoadingIndicator, OnyxIcon,
} from "sit-onyx";
import {ref, useTemplateRef, watch} from "vue";
import check from "@sit-onyx/icons/check.svg?raw";

const props = defineProps<{
  modelValue: string;
}>();
const emits = defineEmits(['update:modelValue']);

const isOpen = ref<boolean>(false);
const waiting = ref(true);
const inputRef = useTemplateRef<HTMLInputElement>("inputRef");

const onKey = (ev: KeyboardEvent) => {
  if (ev.code === "Enter") {
    ev.preventDefault();

    waiting.value = false;
    emits('update:modelValue', inputRef.value?.value);

    setTimeout(() => {
      isOpen.value = false;
      waiting.value = true;
      if (inputRef.value) {
        inputRef.value.value = "";
      }
    }, 500);
  }
};

watch([isOpen, inputRef], () => {
  if (!isOpen.value) {
    return;
  }

  console.log('focusing', inputRef.value);
  setTimeout(() => inputRef.value?.focus(), 0);
  console.log('current focus', document.activeElement);
})
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
      <input type="text" class="nfc-input" aria-label="NFC Scan value" ref="inputRef" @keydown="onKey" @focusout="isOpen = false"/>
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
  margin: var(--onyx-density-xl)
}

.nfc-input {
  height: 0;
  border: none;
  margin: 0;
  padding: 0;
  outline: none;
}
</style>
