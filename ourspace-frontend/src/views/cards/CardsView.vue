<script setup lang="ts">
import {
  type ColumnConfig,
  type ColumnTypesFromFeatures,
  createFeature,
  DataGridFeatures,
  OnyxBottomBar,
  OnyxButton,
  OnyxDataGrid,
  OnyxIconButton,
  OnyxPageLayout,
} from 'sit-onyx'
import {
  cardServiceListCards,
  type CardServiceListCardsResponse,
  memberServiceGetMember,
} from '@/client'
import { computed, h, ref, watch, watchEffect } from 'vue'
import CardActions from '@/views/cards/components/CardActions.vue'
import CardInput from '@/views/cards/components/CardInput.vue'
import { base64ToHex, hexToBase64 } from '@/views/cards/card-utilities.ts'
import {
  iconChevronFirstPage,
  iconChevronRightSmall,
  iconPlusSmall,
  iconUserId,
} from '@sit-onyx/icons'
import SearchInput from '@/components/SearchInput.vue'

type CardEntry = {
  id: string
  member: string
  memberID: string
  rfidValue: string
  validFrom: Date
  validTo: Date
}

const response = ref<CardServiceListCardsResponse>()
const memberLookup = ref<Map<string, string>>()
const currentPageToken = ref<string>('')
const searchValue = ref<string>('')

const columns: ColumnConfig<
  CardEntry,
  Record<string, never>,
  ColumnTypesFromFeatures<typeof features>
>[] = [
  { key: 'member', label: 'Member' },
  { key: 'rfidValue', label: 'RFID Value', type: 'base64ToHex' },
  { key: 'validFrom', label: 'Valid From', type: 'date' },
  { key: 'validTo', label: 'Valid To', type: 'date' },
  { key: 'id', label: 'Actions', type: 'actions', width: 'min-content' },
]

const data = computed<CardEntry[]>(() => {
  return (
    response.value?.cards.map(
      (card): CardEntry => ({
        id: card.id,
        member: memberLookup.value?.get(card.member_id) ?? '',
        memberID: card.member_id,
        rfidValue: [...atob(card.rfid_value)]
          .map((c) => c.charCodeAt(0).toString(16).padStart(2, '0'))
          .join(''),
        validFrom: new Date(card.valid_from),
        validTo: new Date(card.valid_to),
      }),
    ) ?? []
  )
})

const withCustomType = createFeature(() => ({
  name: Symbol('cards table'),
  typeRenderer: {
    actions: DataGridFeatures.createTypeRenderer<object, CardEntry>({
      cell: {
        tdAttributes: {
          style: { width: 'calc(4rem + 2*var(--onyx-density-md))' },
        },
        component: ({ modelValue }) => {
          return h(CardActions, { id: modelValue?.toString() ?? '' })
        },
      },
    }),
    base64ToHex: DataGridFeatures.createTypeRenderer<object, CardEntry>({
      cell: {
        component: ({ modelValue }) => {
          return base64ToHex((modelValue ?? '').toString())
        },
      },
    }),
  },
}))

const features = [withCustomType]

watchEffect(async () => {
  const resp = await cardServiceListCards({
    query: {
      sort_by: 'CARD_FIELD_VALID_TO',
      sort_direction: 'SORT_DIRECTION_DESCENDING',
      page_size: 20,
      page_token: currentPageToken.value,
      rfid_value: searchValue.value != '' ? hexToBase64(searchValue.value) : undefined,
    },
  })

  if (resp.error) {
    console.log(resp.error)
    return
  }

  response.value = resp.data
  memberLookup.value = new Map<string, string>()

  const members = await Promise.all(
    resp.data.cards.map((card) =>
      memberServiceGetMember({
        path: {
          id: card.member_id,
        },
      }),
    ),
  )
  for (const member of members) {
    if (member.error) {
      console.log(member.error)
      continue
    }

    memberLookup.value?.set(member.data.id, member.data.name)
  }
})

watch(searchValue, () => {
  currentPageToken.value = ''
})

const isFirstPage = computed(() => currentPageToken.value === '')

const firstPage = () => {
  currentPageToken.value = ''
}

const shouldShowNextPage = computed(
  (): boolean =>
    response.value?.next_page_token !== undefined && response.value?.next_page_token !== '',
)

const nextPage = () => {
  if (response.value?.next_page_token) {
    currentPageToken.value = response.value?.next_page_token
  }
}
</script>

<template>
  <OnyxPageLayout>
    <div class="table-top-actions">
      <h1>Cards</h1>
      <CardInput v-model="searchValue" v-slot="slot">
        <OnyxIconButton
          label="Scan card"
          :icon="iconUserId"
          mode="plain"
          density="compact"
          color="neutral"
          @click="slot.open"
        />
      </CardInput>
      <SearchInput
        :modelValue="base64ToHex(searchValue)"
        @update:modelValue="($e) => (searchValue = hexToBase64($e ?? ''))"
        placeholder="Exact card Hex ID"
      />
      <div class="separator" />
      <OnyxButton
        label="New card"
        :icon="iconPlusSmall"
        density="compact"
        mode="plain"
        link="/cards/new"
      />
    </div>
    <OnyxDataGrid :columns :data :features class="onyx-density-compact"></OnyxDataGrid>
    <div class="table-bottom-actions">
      <OnyxIconButton
        :icon="iconChevronFirstPage"
        label="Back to start"
        density="compact"
        :disabled="isFirstPage"
        @click="firstPage"
        color="neutral"
      />
      <OnyxIconButton
        :icon="iconChevronRightSmall"
        label="Next Page"
        density="compact"
        :disabled="!shouldShowNextPage"
        @click="nextPage"
        color="neutral"
      />
    </div>
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
  align-items: end;
  justify-content: end;
  margin-bottom: 8px;
}

.table-top-actions h1 {
  flex-grow: 2;
  margin-bottom: 4px;
}

.table-top-actions > *:not(:first-child) {
  margin-left: 10px;
}

.table-bottom-actions {
  margin-top: 8px;
  display: flex;
  flex-direction: row;
  justify-content: end;
}

.table-bottom-actions > *:not(:first-child) {
  margin-left: 10px;
}

.separator {
  border-left: 1px solid var(--onyx-color-text-icons-neutral-soft);
  height: 24px;
  line-height: 24px;
  margin: 4px -10px 4px 4px;
}
</style>
