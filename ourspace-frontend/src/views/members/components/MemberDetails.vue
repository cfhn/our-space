<script setup lang="ts">
import type {MemberServiceGetMemberResponse} from "@/client";
import Input from "../../../components/Input.vue";
import RadioGroup from "@/components/RadioGroup.vue";
import TagInput from "@/views/members/components/TagInput.vue";

const props = defineProps<{
  member: MemberServiceGetMemberResponse,
  isEdit: boolean,
}>();

const ageCategoryOptions = [
  {
    label: 'Underage',
    value: 'AGE_CATEGORY_UNDERAGE'
  }, {
    label: 'Adult',
    value: 'AGE_CATEGORY_ADULT'
  },
];

const tagOptions = ["cfhn", "volunteer", "employee", "employee 2", "employee 3"];

</script>

<template>
  <Input type="text" class="form-row" label="Name" :is-edit="props.isEdit"
         v-model="props.member.name"/>
  <div class="onyx-grid form-row">
    <Input type="date" label="Membership Start" v-model="member.membership_start" required
           :is-edit="props.isEdit" class="onyx-grid-span-12 onyx-grid-lg-span-6"/>
    <Input type="date" label="Membership End" v-model="member.membership_end"
           :is-edit="props.isEdit" class="onyx-grid-span-12 onyx-grid-lg-span-6"/>
  </div>

  <RadioGroup label="Age Category" :options="ageCategoryOptions" :is-edit="isEdit" v-model="member.age_category" class="form-row" />

  <TagInput label="Tags" v-model="member.tags" :options="tagOptions" :is-edit="isEdit" />
</template>

<style scoped>
.form-row {
  margin-bottom: var(--onyx-density-sm);
}
</style>
