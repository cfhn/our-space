<script setup lang="ts">
import { DataGridFeatures,
  OnyxDataGrid,
  OnyxInput,
  OnyxPageLayout,
  OnyxIconButton,
  createFeature,
  type ColumnConfig,
  type ColumnTypesFromFeatures,  
} from 'sit-onyx'

import sync from '@sit-onyx/icons/sync.svg?raw'
import { computed, h, ref } from 'vue' 
import type { PresenceServiceListPresencesResponse } from '@/client'
import { iconChevronFirstPage } from '@sit-onyx/icons'



const firstPage = () => {
  currentPageToken.value = ''
}
const isFirstPage = computed(() => currentPageToken.value === '')

const reload = ref(0)
const response = ref<PresenceServiceListPresencesResponse>()
const currentPageToken = ref<string>('')
const searchValue = ref<string>('')

type PresenceEntry = {
  id: string
  memberId: string
  checkinTime: Date
  checkoutTime?: Date
}

const data = computed<PresenceEntry[]>(() => {
  return (
    response.value?.presence.map(
      (presence): PresenceEntry => ({
        id: presence.id,
        memberId: presence.member_id,
        checkinTime: new Date(presence.checkin_time),
        checkoutTime: presence.checkout_time ? new Date(presence.checkout_time) : undefined,
      }),
    ) ?? []
  )
})

const withCustomType = createFeature(() => ({
  name: Symbol('Presence table'),
  typeRenderer: {
    actions: DataGridFeatures.createTypeRenderer<object, PresenceEntry>({
      cell: {
        tdAttributes: {
          style: { width: 'calc(4rem + 2*var(--onyx-density-md))' },
        },
        component: ({ modelValue }) => {
          const id = modelValue?.toString() ?? ''
          return h( {
            id: id,
          })
        },
      },
    }),
  },
}))

const features = [withCustomType]

const columns: ColumnConfig<
  PresenceEntry,
  Record<string, never>,
  ColumnTypesFromFeatures<typeof features>
>[] = [
  { key: 'memberId', label: 'Member' },
  { key: 'checkinTime', label: 'Checkin Time', type: 'date' },
  { key: 'checkoutTime', label: 'Checkout Time', type: 'date' },

]
</script>


<template>
  <OnyxPageLayout>
    <div class="table-top-actions">
      <h1>Presences</h1>
      <OnyxIconButton label="Refresh" :icon="sync" @click="reload++" density="compact" />
      <OnyxInput
        label="Search"
        :hide-label="true"
        placeholder="Search"
        v-model="searchValue"
        density="compact"
        autofocus
      />
      </div>
      <OnyxDataGrid :columns="columns" :data :features class="onyx-density-compact" />
    <div class="table-bottom-actions">
      <OnyxIconButton
        :icon="iconChevronFirstPage"
        label="Back to start"
        density="compact"
        :disabled="isFirstPage"
        @click="firstPage"
        color="neutral"
      />
    </div>
  </OnyxPageLayout>
  </template>


