<script setup lang="ts">
import {
  OnyxBottomBar,
  OnyxButton,
  OnyxDataGrid,
  OnyxInput,
  OnyxPageLayout,
  type ColumnConfig, createFeature, DataGridFeatures,
} from "sit-onyx";
import {
  cardServiceGetCard,
  cardServiceListCards,
  type CardServiceListCardsResponse, memberServiceGetMember
} from "@/client";
import {computed, h, ref, watch, watchEffect} from "vue";
import CardActions from "@/views/cards/components/CardActions.vue";
import CardInput from "@/views/cards/components/CardInput.vue";

type CardEntry = {
  id: string;
  member: string;
  memberID: string;
  rfidValue: string;
  validFrom: Date;
  validTo: Date;
}

const response = ref<CardServiceListCardsResponse>();
const memberLookup = ref<Map<string, string>>();
const currentPageToken = ref<string>("");
const searchValue = ref<string>("");

const columns: ColumnConfig<CardEntry>[] = [
  {key: "member", label: "Member"},
  {key: "rfidValue", label: "RFID Value"},
  {key: "validFrom", label: "Valid From", type: "date"},
  {key: "validTo", label: "Valid To", type: "date"},
  {key: "id", label: "Actions", type: "actions", width: "min-content"},
];

const data = computed<CardEntry[]>(() => {
  return response.value?.cards.map((card): CardEntry => ({
    id: card.id,
    member: memberLookup.value?.get(card.member_id) ?? "",
    memberID: card.member_id,
    rfidValue: [...atob(card.rfid_value)].map(c=> c.charCodeAt(0).toString(16).padStart(2,0)).join(''),
    validFrom: new Date(card.valid_from),
    validTo: new Date(card.valid_to),
  })) ?? [];
});

const withCustomType = createFeature(() => ({
  name: Symbol("cards table"),
  typeRenderer: {
    actions: DataGridFeatures.createTypeRenderer<{}, CardEntry>({
      cell: {
        tdAttributes: {
          style: { width: "calc(4rem + 2*var(--onyx-density-md))",}
        },
        component: (({modelValue}) => {
          return h(CardActions, {id: modelValue})
        }),
      },
    })
  }
}));

const features = [withCustomType()];

watchEffect(async () => {
  const resp = await cardServiceListCards({
    query: {
      sort_by: "CARD_FIELD_VALID_TO",
      sort_direction: "SORT_DIRECTION_DESCENDING",
      page_size: 10,
      page_token: currentPageToken.value,
      rfid_value: searchValue.value != "" ? searchValue.value : undefined,
    }
  });

  if (resp.error) {
    console.log(resp.error);
    return;
  }

  response.value = resp.data;
  memberLookup.value = new Map<string, string>();

  const members = await Promise.all(resp.data.cards.map((card) => memberServiceGetMember({
    path: {
      id: card.member_id,
    },
  })));
  for (const member of members) {
    if (member.error) {
      console.log(member.error);
      continue;
    }

    memberLookup.value?.set(member.data.id, member.data.name);
  }
});

watch(searchValue, () => {
  currentPageToken.value = "";
})
</script>

<template>
  <OnyxPageLayout>
    <div class="table-top-actions">
      <h1>Cards</h1>
      <OnyxInput label="Search" :hide-label="true" placeholder="Search" v-model="searchValue" density="compact" autofocus />
    </div>
    <OnyxDataGrid :columns :data :features class="onyx-density-compact"></OnyxDataGrid>
    <template #footer>
      <OnyxBottomBar>
        <CardInput v-model="searchValue" mode="plain" />
        <OnyxButton label="New" mode="plain" link="/cards/new" />
      </OnyxBottomBar>
    </template>
  </OnyxPageLayout>
</template>

<style scoped>
.table-top-actions {
  display: flex;
  flex-direction: row;
  align-items: baseline;
  justify-content: end;
  margin-bottom: 8px;
}

.table-top-actions h1 {
  flex-grow: 2;
}

.table-top-actions > *:not(:first-child) {
  margin-left: 10px;
}
</style>
