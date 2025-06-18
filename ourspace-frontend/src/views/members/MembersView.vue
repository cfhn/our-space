<script setup lang="ts">
import {
  DataGridFeatures,
  OnyxBottomBar,
  OnyxButton,
  OnyxDataGrid,
  OnyxModalDialog,
  OnyxInput,
  OnyxPageLayout,
  OnyxIconButton,
  createFeature,
  type ColumnConfig,
} from "sit-onyx";
import {computed, ref, watchEffect, h, watch} from "vue";
import {
  memberServiceDeleteMember,
  memberServiceListMembers,
  type MemberServiceListMembersResponse, memberServiceUpdateMember
} from "@/client";
import MemberTags from "@/views/members/components/MemberTags.vue";
import MemberActions from "@/views/members/components/MemberActions.vue";

import sync from "@sit-onyx/icons/sync.svg?raw";

type MemberEntry = {
  id: string;
  name: string;
  membershipStart: Date;
  membershipEnd?: Date;
  ageCategory: string;
  tags: string[];
};

const reload = ref(0);
const response = ref<MemberServiceListMembersResponse>();
const currentPageToken = ref<string>("");
const searchValue = ref<string>("");

const nextPage = () => {
  console.log(currentPageToken.value, response.value?.next_page_token);

  if (response.value?.next_page_token) {
    currentPageToken.value = response.value?.next_page_token
  }
};

const shouldShowNextPage = computed((): boolean => response.value?.next_page_token !== undefined && response.value?.next_page_token !== "");

const firstPage = () => {
  currentPageToken.value = "";
}

const isFirstPage = computed(() => currentPageToken.value === "");

const deleteMemberDialogOpenFor = ref<string>();

const deleteMember = async (id: string) => {
  const resp = await memberServiceDeleteMember({
    path: {
      id: id,
    },
  });

  if (resp.error) {
    console.log(resp.error);
    return;
  }

  deleteMemberDialogOpenFor.value = undefined;
  reload.value++;
}

const endMembership = async (id: string) => {
  const resp = await memberServiceUpdateMember({
    path: {
      "member.id": id,
    },
    body: {
      name: "",
      membership_start: "0001-01-01T00:00:00.000Z",
      tags: [],
      age_category: "AGE_CATEGORY_UNKNOWN",
      membership_end: (new Date()).toISOString(),
    },
    query: {
      field_mask: "membership_end",
    },
  });
  if (resp.error) {
    console.log(resp.error);
    return;
  }

  deleteMemberDialogOpenFor.value = undefined;
  reload.value++;
}

const data = computed<MemberEntry[]>(() => {
  return response.value?.members.map((member): MemberEntry => ({
    id: member.id,
    membershipStart: new Date(member.membership_start),
    membershipEnd: member.membership_end ? new Date(member.membership_end) : undefined,
    ageCategory: member.age_category == "AGE_CATEGORY_ADULT" ? "Adult" : "Underage",
    name: member.name,
    tags: member.tags,
  })) ?? [];
});

const columns: ColumnConfig<MemberEntry>[] = [
  {key: "name", label: "Name"},
  {key: "ageCategory", label: "Age Category"},
  {key: "membershipStart", label: "Membership Start", type: "date"},
  {key: "membershipEnd", label: "Membership End", type: "date"},
  {key: "tags", label: "Tags", type: "tags"},
  {key: "id", label:"Actions", type: "actions", width: "min-content"},
];

const withCustomType = createFeature(() => ({
  name: Symbol("members table"),
  typeRenderer: {
    tags: {
      cell: {
        component: ({modelValue}) => {
          return h(MemberTags, {tags: modelValue});
        },
      }
    },
    actions: DataGridFeatures.createTypeRenderer<{}, MemberEntry>({
      cell: {
        tdAttributes: {
          style: { width: "calc(4rem + 2*var(--onyx-density-md))" },
        },
        component: ({modelValue}) => {
          const id = modelValue?.toString() ?? "";
          return h(MemberActions, {id: id, onDelete: () => deleteMemberDialogOpenFor.value = id});
        },
      },
    }),
  },
}))

const features = [withCustomType()];

watch([currentPageToken, searchValue, reload], async () => {
  const resp = await memberServiceListMembers({
    query: {
      sort_by: "MEMBER_FIELD_NAME",
      sort_direction: "SORT_DIRECTION_ASCENDING",
      page_size: 10,
      page_token: currentPageToken.value,
      name_contains: searchValue.value != "" ? searchValue.value : undefined,
    },
  });

  if (resp.error) {
    console.log(resp.error);
  } else {
    response.value = resp.data;
  }
}, {immediate: true});

watch(searchValue, () => {
  // Reset page token when the search value changes
  currentPageToken.value = "";
});

</script>

<template>
  <OnyxPageLayout>
    <div class="table-top-actions">
      <h1>Members</h1>
      <OnyxIconButton label="Refresh" :icon="sync" @click="reload++" density="compact" />
      <OnyxInput label="Search" :hide-label="true" placeholder="Search" v-model="searchValue"
                 density="compact" autofocus></OnyxInput>
    </div>
    <OnyxDataGrid :columns :data :features class="onyx-density-compact" />
    <div class="table-bottom-actions">
      <OnyxButton label="Back to start" density="compact" :disabled="isFirstPage" @click="firstPage" color="neutral" />
      <OnyxButton label="Next Page" density="compact" :disabled="!shouldShowNextPage" @click="nextPage"/>
    </div>
    <OnyxModalDialog label="Delete member" :open="deleteMemberDialogOpenFor !== undefined">
      <template #default>
        <div class="modal">
          <p>
            Do you really want to delete the member?
            If yes, click Delete.
          </p>
          <p>
            Alternatively, you can end the membership.
          </p>
        </div>
      </template>
      <template #footer>
        <OnyxBottomBar>
          <OnyxButton label="Cancel" color="neutral" mode="plain" @click="deleteMemberDialogOpenFor = undefined" />
          <OnyxButton label="End membership" color="primary" mode="plain" @click="endMembership(deleteMemberDialogOpenFor ?? '')" />
          <OnyxButton label="Delete" color="danger" mode="plain" @click="deleteMember(deleteMemberDialogOpenFor ?? '')" />
        </OnyxBottomBar>
      </template>
    </OnyxModalDialog>
    <template #footer>
      <OnyxBottomBar>
        <OnyxButton label="New" mode="plain" link="/members/new"/>
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

.modal {
  padding: var(--onyx-density-xl) var(--onyx-modal-dialog-padding-inline);
  color: var(--onyx-color-text-icons-neutral-intense);
}
</style>
