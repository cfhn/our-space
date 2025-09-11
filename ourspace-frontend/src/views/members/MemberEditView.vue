<script setup lang="ts">
import {
  OnyxBottomBar,
  OnyxButton,
  OnyxEmpty,
  OnyxForm,
  OnyxHeadline,
  OnyxIcon,
  OnyxIconButton,
  OnyxPageLayout,
} from "sit-onyx";
import {computed, ref, watchEffect} from "vue";
import {
  cardServiceListCards,
  type CardServiceListCardsResponse,
  type MemberReadable,
  memberServiceCreateMember,
  memberServiceGetMember,
  type MemberServiceGetMemberResponse,
  memberServiceUpdateMember
} from "@/client";
import MemberDetails from "@/views/members/components/MemberDetails.vue";

import userCompanyId from "@sit-onyx/icons/user-company-id.svg?raw";
import iconEdit from "@sit-onyx/icons/edit.svg?raw";
import {useRouter} from "vue-router";
import Card from "@/views/cards/components/Card.vue";

const props = defineProps<{ id: string }>();
const member = ref<MemberServiceGetMemberResponse>({
  id: "",
  name: "",
  age_category: "AGE_CATEGORY_UNKNOWN",
  membership_start: new Date().toISOString(),
  membership_end: undefined,
  tags: [],
});
let memberOriginal: MemberServiceGetMemberResponse = {...member.value};
const cards = ref<CardServiceListCardsResponse>();
const isEdit = ref<boolean>(false);
const isCreate = computed<boolean>(() => !props.id)
const router = useRouter();

const back = () => {
  router.back();
};

const cancelEditing = () => {
  if (!member.value) {
    member.value = {...memberOriginal};
    isEdit.value = false;

    return;
  }

  if (changedFields(member.value, memberOriginal).length !== 0) {
    if (!confirm("Unsaved changes. Do you really want to cancel editing?")) {
      return;
    }
  }

  member.value = {...memberOriginal};
  isEdit.value = false;
}

const changedFields = (member: MemberReadable, memberOriginal: MemberReadable): string[] => {
  const changedFields = [];
  if (member.name !== memberOriginal.name) {
    changedFields.push("name");
  }
  if (member.membership_start !== memberOriginal.membership_start) {
    changedFields.push("membership_start");
  }
  if (member.membership_end !== memberOriginal.membership_end) {
    changedFields.push("membership_end");
  }
  if (member.age_category !== memberOriginal.age_category) {
    changedFields.push("age_category");
  }
  if (!member.tags.every(value => memberOriginal.tags.includes(value))) {
    changedFields.push("tags");
  }
  return changedFields;
}

const shouldEnableSave = computed(() => {
  if (!isEdit.value && !isCreate.value) {
    return false;
  }

  if (!member.value) {
    return false;
  }

  return changedFields(member.value, memberOriginal).length !== 0;
})

const save = async () => {
  if (!member.value) {
    return;
  }

  if (isCreate.value) {
    const resp = await memberServiceCreateMember({
      body: {
        ...member.value,
        membership_end: member.value.membership_end !== "" ? member.value.membership_end : undefined,
      },
    })
    if (resp.error) {
      console.log(resp.error);
      return;
    }

    await router.push(`/members/${resp.data.id}`);
  } else if (isEdit.value) {
    const resp = await memberServiceUpdateMember({
      body: {
        ...member.value,
        membership_end: member.value.membership_end !== "" ? member.value.membership_end : undefined,
      },
      query: {
        field_mask: changedFields(member.value, memberOriginal).join(","),
      },
      path: {
        "member.id": props.id,
      }
    })
    if (resp.error) {
      console.log(resp.error);
      return
    }

    member.value = resp.data;
    memberOriginal = resp.data;
    isEdit.value = false;
  }
};

watchEffect(async () => {
  if (!props.id) {
    return;
  }

  const resp = await memberServiceGetMember({
    path: {id: props.id},
  });

  if (resp.error) {
    console.log(resp.error);
  } else {
    member.value = resp.data;
    memberOriginal = {...resp.data};
  }
});

watchEffect(async () => {
  if (isCreate.value) {
    return;
  }

  const resp = await cardServiceListCards({
    query: {
      member_id: props.id,
      page_size: 10,
      sort_by: "CARD_FIELD_VALID_TO",
      sort_direction: "SORT_DIRECTION_DESCENDING"
    },
  });

  if (resp.error) {
    console.log(resp.error);
  } else {
    cards.value = resp.data;
  }
})
</script>

<template>
  <OnyxPageLayout>
    <div class="onyx-grid">
      <div class="onyx-grid-span-6">
        <OnyxForm v-if="member" ref="memberForm">
          <OnyxHeadline is="h1" v-if="!isCreate">Member
            <OnyxIconButton label="Edit" :icon="iconEdit" @click="isEdit = true" v-if="!isEdit" />
          </OnyxHeadline>
          <OnyxHeadline is="h1" v-if="isCreate">Create new Member</OnyxHeadline>
          <MemberDetails :member="member" :is-edit="isEdit || isCreate"/>
        </OnyxForm>
      </div>
      <div class="onyx-grid-span-6" v-if="!isCreate">
        <OnyxHeadline is="h1">Cards</OnyxHeadline>
        <OnyxEmpty v-if="cards?.cards.length == 0" class="card-empty">
          No cards assigned to this member
          <template #icon>
            <OnyxIcon :icon="userCompanyId" size="48px"/>
          </template>
          <template #buttons>
            <OnyxButton label="Add Card" :link="`/cards/new?memberId=${member.id}`"></OnyxButton>
          </template>
        </OnyxEmpty>
        <Card v-for="card of cards?.cards" :card :member v-if="member" class="card" />
      </div>
    </div>
    <template #footer>
      <OnyxBottomBar>
        <OnyxButton label="Back" mode="plain" color="neutral" @click="back" v-if="!isEdit"></OnyxButton>
        <OnyxButton label="Cancel" mode="plain" color="neutral" @click="cancelEditing" v-if="isEdit"></OnyxButton>
        <OnyxButton label="Save" mode="plain" color="primary" :disabled="!shouldEnableSave" @click="save"></OnyxButton>
      </OnyxBottomBar>
    </template>
  </OnyxPageLayout>
</template>

<style scoped>
.card-empty {
  margin: 0 auto;
}
.card {
  margin-bottom: var(--onyx-spacing-sm);
}
</style>
