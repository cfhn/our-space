<script setup lang="ts">
import {OnyxRadioGroup, type RadioButtonOption, OnyxHeadline, OnyxTag} from "sit-onyx";
import {computed} from "vue";

const props = defineProps<{
  label: string;
  options: RadioButtonOption[];
  isEdit: boolean;
}>();

const model = defineModel<string>();
const displayValue = computed(() => props.options.find(o => o.value === model.value)?.label ?? '')
const height = computed(() => props.isEdit ? "112px": "78px");
</script>

<template>
  <div class="animation-wrapper" :style="{height: height}">
  <Transition :duration="250">
    <div v-if="!isEdit">
      <OnyxHeadline is="h3" class="label">{{ label }}</OnyxHeadline>
      <OnyxTag class="tag" :label="displayValue"/>
    </div>
    <OnyxRadioGroup class="radio-group" :label :options v-model="model" v-else />
  </Transition>
  </div>
</template>

<style scoped>
.tag {
  margin: var(--onyx-spacing-sm) 0;
}

.animation-wrapper {
  position: relative;
}

.v-enter-active .tag, .v-leave-active .tag,
.v-enter-active :deep(.onyx-radio-group__content), .v-leave-active :deep(.onyx-radio-group__content) {
  transition: opacity .25s ease, height .25s ease;
  position: absolute;
}

.v-enter-active :deep(.onyx-radio-group__content), .v-leave-active :deep(.onyx-radio-group__content) {
  top: 32px;
  height: 80px;
}

.v-enter-from .tag, .v-leave-to .tag,
.v-enter-from :deep(.onyx-radio-group__content), .v-leave-to :deep(.onyx-radio-group__content) {
  opacity: 0;
  height: 24px;
}

.v-enter-active :deep(.onyx-radio-group__label), .v-leave-active :deep(.onyx-radio-group__label) {
  display: none;
}

@media (prefers-reduced-motion: reduce) {
  .v-enter-active .tag, .v-leave-active .tag,
  .v-enter-active :deep(.onyx-radio-group__content), .v-leave-active :deep(.onyx-radio-group__content) {
    transition: none;
  }
}

</style>
