<script setup lang="ts">
import type {CardReadable, MemberReadable} from "@/client";
import Input from "@/components/Input.vue";
import CardInput from "@/views/cards/components/CardInput.vue";
import MemberSelect from "@/views/members/components/MemberSelect.vue";
import {watchEffect} from "vue";

const props = defineProps<{
  card: CardReadable;
  isEdit: boolean;
  memberId?: string;
}>();

watchEffect( () => {
  if (props.isEdit && props.memberId) {
    props.card.member_id = props.memberId;
  }
});

</script>

<template>
  <div class="onyx-grid">
    <MemberSelect class="onyx-grid-span-12" label="Assigned to" :is-edit="isEdit" v-model="card.member_id" />
    <Input class="onyx-grid-span-6" type="date" v-model="card.valid_from" :is-edit="isEdit" label="Valid From" />
    <Input class="onyx-grid-span-6" type="date" v-model="card.valid_to" :is-edit="isEdit" label="Valid To" />
    <div class="onyx-grid-span-12 card-input">
      <Input type="text" label="RFID Value" v-model="card.rfid_value" :is-edit="isEdit" />
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
