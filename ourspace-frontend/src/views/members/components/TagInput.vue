<script setup lang="ts">
import {OnyxBasicPopover, OnyxTag, OnyxListItem, OnyxIcon} from "sit-onyx";
import {computed, ref} from "vue";

import xSmall from "@sit-onyx/icons/x-small.svg?raw";
import plusSmall from "@sit-onyx/icons/plus-small.svg?raw";
import tag from "@sit-onyx/icons/tag.svg?raw";

const props = defineProps<{
  label: string;
  options: string[];
  isEdit: boolean;
}>();
const model = defineModel<string[]>();

const inputFocus = ref(false);
const inputValue = ref("");
const selection = ref(0);

const flyoutOpen = computed(() => inputValue.value !== "");
const filteredOptions = computed(() => props.options.filter(t => t.startsWith(inputValue.value) && !model.value?.includes(t)));

const chooseOption = (option: string) => {
  console.log("before", model.value);
  model.value = [...(model.value ?? []), option];
  console.log("after", model.value);
  inputValue.value = "";
  selection.value = 0;
};
const removeTag = (option: string) => {
  model.value = model.value?.filter(t => t !== option);
}
const onKeyDown = (e: KeyboardEvent) => {
  const len = filteredOptions.value.length + 1;
  switch (e.code) {
    case "Tab":
    case "Enter":
      if (selection.value === len-1) {
        if (model.value?.includes(inputValue.value)) {
          break;
        }
        chooseOption(inputValue.value);
      } else {
        chooseOption(filteredOptions.value[selection.value]);
      }
      e.preventDefault();
      break;
    case "Backspace":
      if (inputValue.value === "") {
        model.value = model.value?.slice(0, -1);
      }
      break;
    case "ArrowDown":
      selection.value = (selection.value + 1) % len;
      break;
    case "ArrowUp":
      selection.value = (((selection.value - 1) % len) + len) % len;
      break;
  }
}

const clickable = {
  label: "Click to remove",
  actionIcon: xSmall,
};
</script>

<template>
  <div class="form-elem">
    <label class="label onyx-text--small" for="taginput">{{label}}</label>

    <OnyxBasicPopover label="Tag Input" :open="flyoutOpen" position="bottom" :fit-parent="true" v-if="isEdit">
      <template #default>
        <div class="textbox">
          <OnyxTag v-for="tag in model" :label="tag" :clickable density="compact" @click="removeTag(tag)" />
          <input class="input"
                 type="text"
                 name="taginput"
                 autocomplete="off"
                 @focus="inputFocus = true"
                 @blur="inputFocus = false"
                 @keydown="onKeyDown"
                 v-model="inputValue"
          />
        </div>
      </template>
      <template #content>
          <OnyxListItem
            v-for="(option, index) in filteredOptions"
            @click="chooseOption(option)"
            :active="index === selection"
          ><OnyxIcon label="Tag" :icon="tag" />{{option}}</OnyxListItem>
          <OnyxListItem
            class="add-tag"
            :active="selection === filteredOptions.length"
            @click="chooseOption(inputValue)"
            v-if="!model?.includes(inputValue)"
          ><OnyxIcon label="New" :icon="plusSmall" />Add new tag "{{inputValue}}"</OnyxListItem>
      </template>
    </OnyxBasicPopover>
    <div class="tag-list" v-else>
      <OnyxTag v-for="tag in model" :label="tag" density="compact" />
    </div>
  </div>
</template>

<style scoped>
.form-elem {
  display: flex;
  flex-direction: column;
  gap: var(--onyx-density-3xs)
}

.label {
  display: block;
  color: var(--onyx-color-text-icons-neutral-medium);
}

.input {
  border: none;
  width: 100%;
  outline: none;
  background: transparent;
}

.textbox {
  border: var(--onyx-1px-in-rem) solid var(--onyx-color-component-border-neutral);
  border-radius: var(--onyx-radius-sm);
  padding: var(--onyx-density-xs) var(--onyx-density-sm);
  height: calc(1lh + 2 * var(--onyx-density-xs));
  background-color: var(--onyx-color-base-background-blank);
  display: flex;
  flex-direction: row;
  flex-grow: 1;
  gap: var(--onyx-density-3xs);
}

.textbox:hover {
  border-color: var(--onyx-color-component-border-primary-hover);
}

.textbox:has(input:focus) {
  outline: var(--onyx-outline-width) solid var(--onyx-color-component-focus-primary);
}

.add-tag:not(:first-child) {
  border-top: var(--onyx-1px-in-rem) solid var(--onyx-color-component-border-neutral);
}

.tag-list {
  display: flex;
  flex-direction: row;
  gap: var(--onyx-density-3xs);
}
</style>
