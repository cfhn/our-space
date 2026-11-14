<script setup lang="ts">
import {
  type ColumnConfig,
  type ColumnTypesFromFeatures,
  createFeature,
  DataGridFeatures,
  OnyxPageLayout,
  OnyxDataGrid,
  OnyxIconButton,
  OnyxButton,
  OnyxUnstableDatePickerV2,
  OnyxSelect,
  type DateRange,
  type SelectOption,
} from 'sit-onyx'
import { computed, h, ref, warn, watch } from 'vue'
import PresenceActions from '@/views/presence/components/PresenceActions.vue'
import {
  memberServiceGetMember,
  presenceServiceListPresences,
  type PresenceServiceListPresencesResponse,
} from '@/client'
import { iconChevronFirstPage, iconChevronRightSmall, iconPlusSmall } from '@sit-onyx/icons'
import SearchInput from '@/components/SearchInput.vue'
import sync from '@sit-onyx/icons/sync.svg?raw'
import MemberSelect from '@/views/members/components/MemberSelect.vue'

type PresenceEntry = {
  id: string
  member?: string
  memberId: string
  checkinTime: string
  checkoutTime?: string
}

type PresenceFilters = {
  memberId?: string
  dateRange?: DateRange
  checkedOut?: boolean
}

const response = ref<PresenceServiceListPresencesResponse>()
const deletePresenceDialogOpenFor = ref<string>()
const currentPageToken = ref('')
const memberLookup = ref<Map<string, string>>(new Map<string, string>())
const filters = ref<PresenceFilters>({
  dateRange: undefined,
  memberId: '',
  checkedOut: undefined,
})

const withCustomType = createFeature(() => ({
  name: Symbol('members table'),
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
            onDelete: () => (deletePresenceDialogOpenFor.value = id),
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
  { key: 'member', label: 'Member', type: 'string' },
  { key: 'checkinTime', label: 'Check-In', type: 'datetime-local' },
  { key: 'checkoutTime', label: 'Check-Out', type: 'datetime-local' },
  { key: 'id', label: 'Actions', type: 'actions', width: 'max-content' },
]

const data = computed<PresenceEntry[]>(
  () =>
    response.value?.presence.map(
      (presence): PresenceEntry => ({
        id: presence.id,
        member: memberLookup.value?.get(presence.member_id),
        memberId: presence.member_id,
        checkinTime: presence.checkin_time,
        checkoutTime: presence.checkout_time,
      }),
    ) ?? [],
)

type PresenceOption = 'all' | 'checked_in' | 'checked_out'

const presenceOptions: SelectOption<PresenceOption>[] = [
  { label: 'All', value: 'all' },
  { label: 'Checked In', value: 'checked_in' },
  { label: 'Checked Out', value: 'checked_out' },
]

const filterToPresenceOption = computed(() => {
  switch (filters.value.checkedOut) {
    case undefined:
      return 'all'
    case false:
      return 'checked_in'
    case true:
      return 'checked_out'
  }
})

const presenceOptionToFilter = (option: PresenceOption): boolean | undefined => {
  switch (option) {
    case 'all':
      return undefined
    case 'checked_in':
      return false
    case 'checked_out':
      return true
  }
}

const nextPage = () => {
  if (response.value?.next_page_token) {
    currentPageToken.value = response.value.next_page_token
  }
}

const shouldShowNextPage = computed(
  (): boolean =>
    response.value?.next_page_token !== undefined && response.value.next_page_token !== '',
)

const firstPage = () => {
  currentPageToken.value = ''
}

const isFirstPage = computed(() => currentPageToken.value === '')

const reload = async () => {
  console.log('reload')

  let filter = {}
  if (filters.value.memberId) {
    filter = {
      member_id: filters.value.memberId,
    }
  }
  if (filters.value.dateRange) {
    filter = {
      ...filter,
      checkin_time_after: filters.value.dateRange.start,
    }

    if (filters.value.dateRange.end) {
      filter = {
        ...filter,
        checkout_time_before: new Date(
          new Date(filters.value.dateRange.end).setHours(23, 59, 59),
        ).toISOString(),
      }
    }
  }

  if (filters.value.checkedOut !== undefined) {
    filter = {
      ...filter,
      checked_out: filters.value.checkedOut,
    }
  }

  const resp = await presenceServiceListPresences({
    query: {
      page_size: 15,
      page_token: currentPageToken.value,
      sort_by: 'PRESENCE_FIELD_CHECKIN_TIME',
      sort_direction: 'SORT_DIRECTION_DESCENDING',
      ...filter,
    },
  })

  if (resp.error) {
    console.error(resp.error)
    return
  } else {
    response.value = resp.data
  }

  const members = await Promise.all(
    resp.data?.presence.map((presence) =>
      memberServiceGetMember({
        path: {
          id: presence.member_id,
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
}

watch([currentPageToken, filters], reload, { immediate: true, deep: true })
</script>

<template>
  <OnyxPageLayout>
    <div class="table-top-actions">
      <h1>Presence</h1>
      <OnyxIconButton
        label="Refresh"
        :icon="sync"
        @click="reload"
        density="compact"
        color="neutral"
      />
      <SearchInput>
        <div class="search-inputs">
          <MemberSelect
            :label="{ label: 'Member', position: 'left' }"
            :isEdit="true"
            density="compact"
            v-model="filters.memberId"
          />
          <OnyxUnstableDatePickerV2
            selectionMode="range"
            :label="{ label: 'In Range', position: 'left' }"
            density="compact"
            hideLabel
            v-model="filters.dateRange"
          />
          <OnyxSelect
            :label="{ label: 'Status', position: 'left' }"
            listLabel="Presence status: All, currently checked in or currently checked out"
            :options="presenceOptions"
            density="compact"
            :modelValue="filterToPresenceOption"
            @update:modelValue="filters.checkedOut = presenceOptionToFilter($event ?? 'all')"
            hideClearIcon
          />
        </div>
      </SearchInput>
      <div class="separator" />
      <OnyxButton
        label="New presence"
        :icon="iconPlusSmall"
        density="compact"
        mode="plain"
        disabled
        title="Not yet supported"
      />
    </div>
    <OnyxDataGrid :columns="columns" :data :features class="onyx-density-compact"></OnyxDataGrid>
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

.search-inputs {
  display: flex;
  flex-direction: row;
  gap: 1.5rem;
}
</style>
