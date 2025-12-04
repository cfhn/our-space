<script setup lang="ts">
import {OnyxSelect, OnyxSkeleton, type SelectOption} from "sit-onyx";
import {computed, ref, watch} from "vue";
import {type MemberReadable, memberServiceGetMember, memberServiceListMembers} from "@/client";

const props = defineProps<{
  label: string;
  isEdit: boolean;
}>();
const model = defineModel<string>();
const selection = ref<MemberReadable>();
const searchTerm = ref("");
const response = ref<MemberReadable[]>([]);
const options = computed<SelectOption<string>[]>(() => {
  const fromSearch = response.value?.map((member) => ({
    label: member.name,
    value: member.id,
  })) ?? [];

  const fromSelection = [];
  if (selection.value) {
    fromSelection.push({
      label: selection.value.name,
      value: selection.value.id,
    })
  }

  return [...fromSearch, ...fromSelection];
});

watch(model, async () => {
  if (!model.value) {
    return;
  }

  const result = response.value.find((member) => member.id === model.value);
  if (result) {
    selection.value = result;
    return;
  }

  const member = await memberServiceGetMember({
    path: {
      id: model.value,
    }
  });

  if (member.error) {
    console.error(member.error);
    return;
  }

  selection.value = member.data;
}, {immediate: true});

watch(searchTerm, async () => {
  const resp = await memberServiceListMembers({
    query: {
      name_contains: searchTerm.value,
      page_size: 20,
    }
  });

  if (resp.error) {
    console.error(resp.error);
    return;
  }

  response.value = resp.data.members;
}, {immediate: true});
</script>

<template>
  <OnyxSelect :label="props.label" listLabel="" :options v-model:searchTerm="searchTerm" :withSearch="true" v-model="model" v-if="isEdit"/>
  <div v-else>
    <div class="onyx-text--small label">{{props.label}}</div>
    <div v-if="selection">
      {{selection?.name}}
    </div>
    <OnyxSkeleton class="selection-skeleton" v-else />
  </div>
</template>

<style scoped>
.label {
  color: var(--onyx-color-text-icons-neutral-medium);
}
.selection-skeleton {
  width: 100px;
  height: 1rem;
}
</style>
