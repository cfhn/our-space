<script setup lang="ts">
import {OnyxDatePicker, OnyxInput} from "sit-onyx";
import {computed} from "vue";

const props = defineProps<{
  label: string;
  isEdit: boolean;
  type: "text" | "date";
}>();

const model = defineModel<string>();
const date = computed(() => {
  if (props.type !== 'date') {
    return "";
  }

  if (model.value === null) {
    return "-";
  }

  return new Date(model.value as string).toLocaleString();
})
</script>

<template>
  <OnyxInput :label="props.label" v-model="model" v-if="isEdit && type == 'text'" v-bind="$attrs" />
  <OnyxDatePicker type="datetime-local" :label="props.label" v-model="model" v-if="isEdit && type == 'date'" v-bind="$attrs" />
  <div v-if="!isEdit" v-bind="$attrs">
    <p class="onyx-text--small label">{{props.label}}</p>
    <p class="value" v-if="type !== 'date'">{{model}}</p>
    <p class="value" v-if="type === 'date'">{{date}}</p>
  </div>
</template>

<style scoped>
.label {
  color: var(--onyx-color-text-icons-neutral-medium);
}
.value {
  padding: var(--onyx-density-xs) 0;
  margin: calc(2*var(--onyx-1px-in-rem)) 0;
}
</style>
