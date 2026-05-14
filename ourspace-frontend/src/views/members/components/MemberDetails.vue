<script setup lang="ts">
import { memberServiceListMemberTags, type MemberWritable } from '@/client'
import DynamicInput from '../../../components/DynamicInput.vue'
import RadioGroup from '@/components/RadioGroup.vue'
import TagInput from '@/views/members/components/TagInput.vue'
import { watch, ref, watchEffect } from 'vue'
import { OnyxHeadline, OnyxSwitch } from 'sit-onyx'

const props = defineProps<{
  isEdit: boolean
}>()

const member = defineModel<MemberWritable>('member', { required: true })

const memberAuth = ref<boolean>(member.value.member_login !== undefined)
const username = ref<string>(member.value.member_login?.username ?? '')
const password = ref<string>(member.value.member_login?.password ?? '')

watch(
  () => member.value.name,
  () => {
    console.log(member.value.member_login)
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
</script>

<template>
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

  <OnyxHeadline is="h2" class="headline">Authentication</OnyxHeadline>
  <OnyxSwitch label="Has Account" :disabled="!isEdit" v-model="memberAuth" />
  <template v-if="member.member_login">
    <DynamicInput type="text" label="Username" :is-edit="isEdit" v-model="username" />
    <DynamicInput type="password" label="Password" :is-edit="isEdit" v-model="password" />
  </template>
</template>

<style scoped>
.form-row {
  margin-bottom: var(--onyx-density-sm);
}

.headline {
  margin-top: var(--onyx-density-md);
}
</style>
