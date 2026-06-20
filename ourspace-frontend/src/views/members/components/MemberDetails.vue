<script setup lang="ts">
import { type MemberAttribute, memberServiceListMemberTags, type MemberWritable } from '@/client'
import DynamicInput from '../../../components/DynamicInput.vue'
import RadioGroup from '@/components/RadioGroup.vue'
import TagInput from '@/views/members/components/TagInput.vue'
import { watch, ref, watchEffect, computed } from 'vue'
import {
  OnyxHeadline,
  OnyxSwitch,
  OnyxCard,
  OnyxIconButton,
  OnyxFlyoutMenu,
  OnyxMenuItem,
} from 'sit-onyx'
import { iconPlusSmall } from '@sit-onyx/icons'
import CustomAttribute from '@/views/members/components/CustomAttribute.vue'

const props = defineProps<{
  isEdit: boolean
  customAttributes: MemberAttribute[]
}>()

const member = defineModel<MemberWritable>('member', { required: true })

const memberAuth = ref<boolean>(member.value.member_login !== undefined)
const username = ref<string>(member.value.member_login?.username ?? '')
const password = ref<string>(member.value.member_login?.password ?? '')

watch(
  () => member.value.name,
  () => {
    memberAuth.value = member.value.member_login !== undefined
    username.value = member.value.member_login?.username ?? ''
    password.value = member.value.member_login?.password ?? ''
  },
)
watch(memberAuth, () => {
  if (memberAuth.value) {
    member.value.member_login = {
      username: username.value,
      password: password.value,
    }
  } else {
    member.value.member_login = undefined
  }
})
watch(username, () => {
  if (member.value.member_login) {
    member.value.member_login.username = username.value
  }
})
watch(password, () => {
  if (member.value.member_login) {
    member.value.member_login.password = password.value
  }
})

const ageCategoryOptions = [
  {
    label: 'Underage',
    value: 'AGE_CATEGORY_UNDERAGE',
  },
  {
    label: 'Adult',
    value: 'AGE_CATEGORY_ADULT',
  },
]

const tagOptions = ref<string[]>([])

watchEffect(() => {
  memberServiceListMemberTags({
    query: {
      page_size: 100,
    },
  }).then((resp) => {
    tagOptions.value = resp.data?.tags ?? []
  })
})

const presentAttributes = computed(() =>
  props.customAttributes.filter(
    (a) => a.technical_name in (member.value.additional_attributes ?? {}),
  ),
)
const absentAttributes = computed(() =>
  props.customAttributes.filter(
    (a) => !(a.technical_name in (member.value.additional_attributes ?? {})),
  ),
)

const addCustomAttribute = (attribute: MemberAttribute): void => {
  if (member.value.additional_attributes === undefined) {
    member.value.additional_attributes = {}
  }
  member.value.additional_attributes[attribute.technical_name] = ''
}

const removeCustomAttribute = (attribute: MemberAttribute): void => {
  if (member.value.additional_attributes === undefined) {
    return
  }
  delete member.value.additional_attributes[attribute.technical_name]
}
</script>

<template>
  <OnyxCard class="section">
    <OnyxHeadline is="h2" class="headline">Basic information</OnyxHeadline>
    <DynamicInput
      type="text"
      class="form-row"
      label="Name"
      :is-edit="props.isEdit"
      v-model="member.name"
    />
    <div class="onyx-grid form-row">
      <DynamicInput
        type="date"
        label="Membership Start"
        v-model="member.membership_start"
        required
        :is-edit="props.isEdit"
        class="onyx-grid-span-12 onyx-grid-lg-span-6"
      />
      <DynamicInput
        type="date"
        label="Membership End"
        v-model="member.membership_end"
        :is-edit="props.isEdit"
        class="onyx-grid-span-12 onyx-grid-lg-span-6"
      />
    </div>

    <RadioGroup
      label="Age Category"
      :options="ageCategoryOptions"
      :is-edit="isEdit"
      v-model="member.age_category"
      class="form-row"
    />

    <TagInput label="Tags" v-model="member.tags" :options="tagOptions" :is-edit="isEdit" />
  </OnyxCard>

  <OnyxCard class="section">
    <OnyxHeadline is="h2" class="headline">Authentication</OnyxHeadline>
    <OnyxSwitch label="Has Account" :disabled="!isEdit" v-model="memberAuth" />
    <template v-if="member.member_login">
      <DynamicInput type="text" label="Username" :is-edit="isEdit" v-model="username" />
      <DynamicInput type="password" label="Password" :is-edit="isEdit" v-model="password" />
    </template>
  </OnyxCard>
  <OnyxCard class="section">
    <OnyxHeadline is="h2" class="headline other-attributes">
      Other attributes
      <OnyxFlyoutMenu
        label="Add other attribute"
        trigger="click"
        v-if="isEdit && absentAttributes.length !== 0"
      >
        <template #button="{ trigger }">
          <OnyxIconButton
            label="Add"
            :icon="iconPlusSmall"
            color="neutral"
            density="compact"
            v-bind="trigger"
          />
        </template>
        <template #options>
          <OnyxMenuItem
            v-for="attribute in absentAttributes"
            @click="addCustomAttribute(attribute)"
            :key="attribute.id"
          >
            Add {{ attribute.display_name }}
          </OnyxMenuItem>
        </template>
      </OnyxFlyoutMenu>
    </OnyxHeadline>
    <div v-if="presentAttributes.length === 0" class="no-attributes">
      This member has no other attributes. Edit to add some.
    </div>
    <CustomAttribute
      v-for="attribute in presentAttributes"
      :key="attribute.id"
      :attribute
      :is-edit="isEdit"
      v-model="(member.additional_attributes ?? {})[attribute.technical_name]"
      @delete="removeCustomAttribute(attribute)"
    />
  </OnyxCard>
</template>

<style scoped>
.form-row {
  margin-bottom: var(--onyx-density-sm);
}

.section {
  margin: var(--onyx-density-md) 0;
}

.headline {
}

.other-attributes {
  display: flex;
  flex-direction: row;
  align-items: center;
}

.no-attributes {
  color: var(--onyx-color-text-icons-neutral-medium);
}
</style>
