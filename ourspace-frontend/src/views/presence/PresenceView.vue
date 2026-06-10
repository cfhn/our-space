<script setup lang="ts">
import { createFeature, DataGridFeatures, OnyxPageLayout,
  type ColumnConfig,
  type ColumnTypesFromFeatures,
  
} from 'sit-onyx'

import sync from '@sit-onyx/icons/sync.svg?raw'
import { h, ref } from 'vue' 



const reload = ref(0)
const currentPageToken = ref<string>('')
const searchValue = ref<string>('')

type PresenceEntry = {
  id: string
  memberId: string
  checkinTime: Date
  checkoutTime?: Date
}


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
          return h(PresenceActions, {
            id: id,
            onDelete: () => (deleteMemberDialogOpenFor.value = id),
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
  </OnyxPageLayout>
  </template>


