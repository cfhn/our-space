<script setup lang="ts">
import {
  OnyxPageLayout,
  OnyxForm,
  OnyxHeadline,
  OnyxIconButton,
  OnyxButton,
  OnyxBottomBar
} from "sit-onyx";
import {computed, ref, watch, watchEffect} from "vue";
import iconEdit from "@sit-onyx/icons/edit.svg?raw";
import CardDetails from "@/views/cards/components/CardDetails.vue";
import {
  type CardReadable, cardServiceCreateCard,
  cardServiceGetCard, cardServiceUpdateCard,
  type MemberReadable,
  memberServiceGetMember
} from "@/client";
import {useRouter} from "vue-router";

const addYear = (d: Date, year: number): Date => {
  d.setFullYear(d.getFullYear() + year);
  return d;
};
const props = defineProps<{
  id: string;
}>();
const isEdit = ref(false);
const isCreate = computed(() => !props.id);
const card = ref<CardReadable>({
  id: "",
  member_id: "",
  rfid_value: "",
  valid_from: new Date().toISOString(),
  valid_to: addYear(new Date(), 1).toISOString(),
});
let cardOriginal: CardReadable = {...card.value};
const member = ref<MemberReadable>();
const router = useRouter();

watchEffect(async () => {
  if (isCreate.value) {
    return;
  }

  const cardResp = await cardServiceGetCard({
    path: {
      id: props.id,
    }
  });

  if (cardResp.error) {
    console.error(cardResp.error);
    return;
  }

  const memberResp = await memberServiceGetMember({
    path: {
      id: cardResp.data.member_id,
    },
  });

  if (memberResp.error) {
    console.error(memberResp.error);
    return;
  }

  card.value = cardResp.data;
  member.value = memberResp.data;
  cardOriginal = {...cardResp.data};
})

const back = () => {
  router.back();
};

const shouldEnableSave = computed(() => {
  if (!isEdit.value && !isCreate.value) {
    return false;
  }

  if (!card.value) {
    return false;
  }

  return changedFields(card.value, cardOriginal);
});

const changedFields = (card: CardReadable, cardOriginal: CardReadable): string[] => {
  console.log(card, cardOriginal);

  const changedFields = [];
  if (card.member_id !== cardOriginal.member_id) {
    changedFields.push("member_id");
  }
  if (card.rfid_value !== cardOriginal.rfid_value) {
    changedFields.push("rfid_value");
  }
  if (card.valid_from !== cardOriginal.valid_from) {
    changedFields.push("valid_from");
  }
  if (card.valid_to !== cardOriginal.valid_to) {
    changedFields.push("valid_to");
  }
  return changedFields;
}

const cancelEditing = () => {
  if (!card.value) {
    card.value = {...cardOriginal};
    isEdit.value = false;

    return;
  }

  const changed = changedFields(card.value, cardOriginal)

  if (changed.length !== 0) {
    console.log(changed);
    if(!confirm("Unsaved changes. Do you really want to cancel editing?")) {
      return;
    }
  }

  card.value = {...cardOriginal};
  isEdit.value = false;
};

const save = async () => {
  if (!card.value) {
    return;
  }

  if (isCreate.value) {
    const resp = await cardServiceCreateCard({
      body: {
        ...card.value,
      },
    });
    if (resp.error) {
      console.error(resp.error);
      return;
    }

    await router.push(`/cards/${resp.data.id}`);
  } else if (isEdit.value) {
    const resp = await cardServiceUpdateCard({
      body: {
        ...card.value,
      },
      path: {
        "card.id": props.id,
      },
      query: {
        field_mask: changedFields(card.value, cardOriginal).join(","),
      }
    });
    if (resp.error) {
      console.error(resp.error);
      return;
    }

    card.value = resp.data;
    cardOriginal = {...resp.data};
    isEdit.value = false;
  }
};
</script>

<template>
  <OnyxPageLayout>
    <div class="onyx-grid">
      <div class="onyx-grid-span-6">
        <OnyxForm>
          <OnyxHeadline is="h1" v-if="!isCreate">Card
            <OnyxIconButton label="Edit" :icon="iconEdit" @click="isEdit = true" v-if="!isEdit"/>
          </OnyxHeadline>
          <OnyxHeadline is="h1" v-else>Create new Card</OnyxHeadline>

          <CardDetails v-if="card" :card="card" :is-edit="isEdit || isCreate" />
        </OnyxForm>
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

</style>
