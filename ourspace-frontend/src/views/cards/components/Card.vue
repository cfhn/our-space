<script setup lang="ts">
import {OnyxAvatar, OnyxCard} from "sit-onyx";
import type {CardReadable, MemberReadable} from "@/client";
import mseCard from "@/assets/makerspace-card-min.svg";
import {computed} from "vue";
import {useRouter} from "vue-router";

const props = defineProps<{
  card: CardReadable;
  member: MemberReadable;
}>();
const router = useRouter();

const validFrom = computed(() => {
  const date = new Date(props.card.valid_from);
  return `${date.getFullYear()}-${("00"+(date.getMonth() + 1)).slice(-2)}-${("00"+date.getDate()).slice(-2)}`;
});

const validTo = computed(() => {
  const date = new Date(props.card.valid_to);
  return `${date.getFullYear()}-${("00"+(date.getMonth() + 1)).slice(-2)}-${("00"+date.getDate()).slice(-2)}`;
});

const rfidValue = computed(() => [...atob(props.card.rfid_value)].map(c=> c.charCodeAt(0).toString(16).padStart(2,"0")).join(''))

const isValid = computed(() => {
  const today = new Date();
  const validFrom = new Date(props.card.valid_from);
  const validTo = new Date(props.card.valid_to);

  return today > validFrom && today < validTo;
});

const openCard = (id: string) => {
  router.push(`/cards/${id}`);
};
</script>

<template>
  <OnyxCard :class="{'makerspace-card': true, 'makerspace-card-invalid': !isValid }" :style="{'background-image': `url('${mseCard}')`}" @click="openCard(card.id)">
    <OnyxAvatar :full-name="member.name" size="48px" />
    <div class="name">{{member.name}}</div>
    <div class="validity">
      {{validFrom}} - {{validTo}}
    </div>
    <div class="rfid">{{rfidValue}}</div>
    <div class="invalid">invalid</div>
  </OnyxCard>
</template>

<style scoped>
.makerspace-card {
  position: relative;
  width: 324px;
  height: 204px;
  cursor: pointer;
}
.makerspace-card-invalid {
  filter: grayscale(100%) brightness(110%);
}
.name {
  position: absolute;
  top: 96px;
  left: 24px;
  font-weight: bold;
}

.validity {
  position: absolute;
  top: 120px;
  left: 24px;
}
.rfid {
  position: absolute;
  top: 148px;
  left: 24px;
  font-family: 'Source Code Pro', monospace;
  font-size: 10pt;
}
.invalid {
  display: none;
}

.makerspace-card-invalid .invalid {
  display: block;
  position: absolute;
  top: 32px;
  left: 96px;
  text-transform: uppercase;
  font-size: 24px;
}

</style>
