<script setup lang="ts">
import {
  type ColumnConfig,
  type ColumnTypesFromFeatures,
  createFeature,
  DataGridFeatures,
  OnyxBottomBar,
  OnyxButton,
  OnyxDataGrid,
  OnyxForm,
  OnyxIconButton,
  OnyxInput,
  OnyxModal,
  OnyxSelect,
  OnyxTextarea,
} from "sit-onyx";
import {computed, h, ref, watch} from "vue";
import MemberAttributeActions from "@/views/settings/components/MemberAttributeActions.vue";
import {
  type MemberAttribute,
  memberServiceCreateMemberAttribute,
  memberServiceDeleteMemberAttribute,
  memberServiceListMemberAttributes,
  type MemberServiceListMemberAttributesResponse,
  memberServiceUpdateMemberAttribute
} from "@/client";
import {
  iconChevronFirstPage,
  iconChevronRightSmall,
  iconPlusSmall,
  iconSync
} from "@sit-onyx/icons";

type MemberAttributeEntry = {
  id: string,
  technicalName: string,
  displayName: string,
  type: string,
  description: string,
};

const reload = ref(0);
const response = ref<MemberServiceListMemberAttributesResponse>();
const currentPageToken = ref<string>('');

const nextPage = () => {
  if (response.value?.next_page_token) {
    currentPageToken.value = response.value.next_page_token;
  }
};

const shouldShowNextPage = computed(
  (): boolean => response.value?.next_page_token !== undefined && response.value.next_page_token !== ''
);

const firstPage = () => {
  currentPageToken.value = '';
}
const isFirstPage = computed(() => currentPageToken.value === '');

const withCustomType = createFeature(() => ({
  name: Symbol('member attribute row actions'),
  typeRenderer: {
    actions: DataGridFeatures.createTypeRenderer<object, MemberAttributeEntry>({
      cell: {
        tdAttributes: {
          style: {width: 'calc(4rem + 2*var(--onyx-density-md))'},
        },
        component: ({modelValue, row}) => {
          const id = modelValue?.toString() ?? ''
          return h(MemberAttributeActions, {
            id: id,
            onEdit: () => {
              createEditModal.value.mode = 'edit';
              createEditModal.value.value = {
                id: row.id,
                display_name: row.displayName,
                technical_name: row.technicalName,
                description: row.description,
                type: row.type as (MemberAttribute['type']),
              };
              createEditModal.value.original = {
                ...createEditModal.value.value,
              }
              createEditModal.value.id = row.id;
              createEditModal.value.open = true;
            },
            onDelete: () => {
              deleteModal.value.id = row.id;
              deleteModal.value.entry = row;
              deleteModal.value.open = true;
            },
          })
        },
      },
    }),
    attributeType: DataGridFeatures.createTypeRenderer<object, MemberAttributeEntry>({
      cell: {
        component: ({modelValue}) => {
          switch (modelValue) {
            case 'TYPE_TEXT_SINGLE_LINE':
              return 'Single line text';
            case 'TYPE_TEXT_MULI_LINE':
              return 'Multi line text'
            case 'TYPE_NUMBER':
              return 'Numerical text';
            case 'TYPE_DATE':
              return 'Date';
            case 'TYPE_DATETIME':
              return 'Date and time';
          }
        },
      },
    }),
  },
}));

const withCustomActions = createFeature(() => ({
  name: Symbol('member attribute actions'),
  actions: () => [
    {
      label: "Reload",
      icon: iconSync,
      color: "neutral",
      onClick: () => reload.value++,
    },
    {
      label: "Add attribute",
      icon: iconPlusSmall,
      displayAs: "button",
      mode: "plain",
      group: "group-1",
      onClick: () => {
        createEditModal.value.mode = 'create';
        createEditModal.value.value = {
          id: '',
          technical_name: '',
          display_name: '',
          description: '',
          type: 'TYPE_UNKNOWN',
        }
        createEditModal.value.open = true;
      },
    }
  ],
}))

const features = [withCustomType, withCustomActions];

const columns: ColumnConfig<
  MemberAttributeEntry,
  Record<string, never>,
  ColumnTypesFromFeatures<typeof features>
>[] = [
  {key: 'technicalName', label: 'Technical Name'},
  {key: 'displayName', label: 'Display Name'},
  {key: 'type', label: 'Data Type', type: 'attributeType'},
  {key: 'description', label: 'Description'},
  {key: 'id', label: 'Actions', type: 'actions', width: 'min-content'},
];

const data = computed(() => {
  return (
    response.value?.attributes?.map((attribute): MemberAttributeEntry => ({
        id: attribute.id,
        technicalName: attribute.technical_name,
        displayName: attribute.display_name,
        type: attribute.type,
        description: attribute.description,
      }),
    ) ?? [])
});

watch([currentPageToken, reload], async () => {
  const resp = await memberServiceListMemberAttributes({
    query: {
      page_size: 5,
      sort_by: "MEMBER_ATTRIBUTE_FIELD_TECHNICAL_NAME",
      sort_direction: "SORT_DIRECTION_ASCENDING",
      page_token: currentPageToken.value,
    }
  });

  if (resp.error) {
    console.error(resp.error);
  } else {
    response.value = resp.data;
  }
}, {immediate: true});

const createEditModal = ref<{
  open: boolean;
  mode?: 'create' | 'edit';
  id?: string;
  value: MemberAttribute;
  original?: MemberAttribute;
}>({
  open: false,
  value: {
    id: '',
    technical_name: '',
    display_name: '',
    description: '',
    type: 'TYPE_UNKNOWN',
  }
});

const handleSubmit = async () => {
  switch (createEditModal.value.mode) {
    case 'create':
      await createAttribute(createEditModal.value.value);
      break;
    case 'edit':
      if (createEditModal.value.id === undefined || createEditModal.value.original === undefined) {
        return;
      }

      await patchAttribute(createEditModal.value.id, createEditModal.value.value, createEditModal.value.original ?? {});
      break;
  }

  createEditModal.value.open = false
  reload.value++;
};

const createAttribute = async (value: MemberAttribute) => {
  return memberServiceCreateMemberAttribute({
    body: value,
  });
};

const patchAttribute = async (id: string, value: MemberAttribute, original: MemberAttribute) => {
  const fieldMask: string[] = [];

  if (value.display_name !== original.display_name) {
    fieldMask.push('display_name');
  }

  if (value.description !== original.description) {
    fieldMask.push('description');
  }

  return memberServiceUpdateMemberAttribute({
    body: value,
    path: {
      "attribute.id": id,
    },
    query: {
      field_mask: fieldMask.join(','),
    },
  })
};

const attributeTypeOptions: {
  value: MemberAttribute['type'],
  label: string,
}[] = [
  {
    value: 'TYPE_TEXT_SINGLE_LINE',
    label: 'Single line text'
  },
  {
    value: 'TYPE_TEXT_MULI_LINE',
    label: 'Multi line text'
  },
  {
    value: 'TYPE_NUMBER',
    label: 'Numerical text'
  },
  {
    value: 'TYPE_DATE',
    label: 'Date value'
  },
  {
    value: 'TYPE_DATETIME',
    label: 'Date and time'
  }
];

const deleteModal = ref<{
  open: boolean;
  id?: string;
  entry?: MemberAttributeEntry;
}>({
  open: false,
});

const handleDelete = async () => {
  if (deleteModal.value.id === undefined) {
    return;
  }

  const resp = await memberServiceDeleteMemberAttribute({
    path: {
      id: deleteModal.value.id,
    }
  });

  if (resp.error) {
    console.error(resp.error);
  }

  deleteModal.value.open = false;
  reload.value++;
}
</script>

<template>
  <h2 class="headline">Custom attributes</h2>
  <p class="info-text">
    Custom attributes allow you to extend the member data model with attributes that suit your
    needs.
  </p>
  <OnyxDataGrid headline="Attributes" :columns="columns" :data :features
                class="onyx-density-compact"/>
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
  <OnyxModal :label="createEditModal.mode === 'create' ? 'Create Attribute' : 'Edit Attribute'"
             :open="createEditModal.open"
             @update:open="createEditModal.open = false"
             :nonDismissible="true"
  >
    <template #default>
      <div class="edit-modal">
        <OnyxForm @submit.prevent="handleSubmit">
          <OnyxInput label="Technical Name"
                     v-model="createEditModal.value.technical_name"
                     required
                     :readonly="createEditModal.mode === 'edit'"
          />
          <OnyxInput label="Display Name"
                     v-model="createEditModal.value.display_name"
                     required
                     :maxlength="256"
                     withCounter
          />
          <OnyxSelect label="Type"
                      listLabel="Attribute data types"
                      :options="attributeTypeOptions"
                      v-model="createEditModal.value.type"
                      required
                      :readonly="createEditModal.mode === 'edit'"
          />
          <OnyxTextarea label="Description"
                        v-model="createEditModal.value.description"
                        required
                        :maxlenght="4096"
                        withCounter
          />
        </OnyxForm>
      </div>
    </template>
    <template #footer>
      <OnyxBottomBar>
        <OnyxButton label="Cancel" color="neutral" mode="plain"
                    @click="createEditModal.open = false"/>
        <OnyxButton label="Save" color="primary" mode="plain"
                    @click="handleSubmit"/>
      </OnyxBottomBar>
    </template>
  </OnyxModal>
  <OnyxModal :open="deleteModal.open" label="Delete Attribute">
    <template #default>
      <div class="delete-modal">
        <p>Do you really want to delete the attribute "{{ deleteModal.entry?.technicalName }}"?</p>
        <p>
          This will not delete the attribute values on all members. Instead they will be listed
          as an unknown attribute value and the attribute can't be added to any new members.
        </p>
      </div>
    </template>
    <template #footer>
      <OnyxBottomBar>
        <OnyxButton label="Cancel" color="neutral" mode="plain" @click="deleteModal.open = false" />
        <OnyxButton label="Delete" color="danger" mode="plain" @click="handleDelete()" />
      </OnyxBottomBar>
    </template>
  </OnyxModal>
</template>

<style scoped>
.headline {
  margin-bottom: var(--onyx-density-xs);
}

.info-text {
  color: var(--onyx-color-text-icons-neutral-medium);
}

.table-bottom-actions {
  margin-top: 8px;
  display: flex;
  flex-direction: row;
  justify-content: end;
  gap: var(--onyx-density-xs);
}

.edit-modal {
  padding: var(--onyx-density-xl) var(--onyx-modal-padding-inline);
  width: calc(100vw - 2 * var(--onyx-grid-margin));
}

.delete-modal {
  padding: var(--onyx-density-xl) var(--onyx-modal-padding-inline);
}
</style>
