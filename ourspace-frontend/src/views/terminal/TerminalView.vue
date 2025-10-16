<script setup lang="ts">
import {useEventSource, useTimeout, useTimeoutFn} from "@vueuse/core";
import {ref, watch} from "vue";

type Member = {
  id: string;
  name: string;
}

type Card = {
  id: string;
  validFrom: string;
  validTo: string;
};

type Update = {
  token: string;
  card: Card;
  member: Member;
}

const {data} = useEventSource<string[], string>("http://localhost:8081/card-events", ['data'], {
  autoReconnect: true,
});
const member = ref<Member>();
const card = ref<Card>();
const backgroundColor = ref<string>("green");
const cardValidTo = ref<string>("");
const countdown = ref<boolean>(false);

const {start, stop} = useTimeoutFn(() => {
  backgroundColor.value = "green";
  cardValidTo.value = "";
  member.value = undefined;
  card.value = undefined;
  countdown.value = false;
  data.value = "";
}, 9920);

watch(data, () => {
  console.log(data.value);
  if (!data.value) {
    return;
  }

  stop();

  const update: Update = JSON.parse(data.value);

  member.value = update.member;
  card.value = update.card;

  const cardExpires = new Date(update.card.validTo);

  console.log(cardExpires.getTime(), Date.now());
  console.log(((cardExpires.getTime() - Date.now()) / (1000 * 60 * 60 * 24)));

  if (cardExpires.getTime() - Date.now() <= 0) {
    backgroundColor.value = "red"
  } else if (((cardExpires.getTime() - Date.now()) / (1000 * 60 * 60 * 24)) <= 14) {
    backgroundColor.value = "orange"
  } else {
    backgroundColor.value = "green"
  }

  cardValidTo.value = `${("00" + cardExpires.getDate()).slice(-2)}.${("00" + (cardExpires.getMonth() + 1)).slice(-2)}.${cardExpires.getFullYear()}`;
  countdown.value = true;
  start();
});

</script>

<template>
  <div class="progress">
    <div :class="{'progress-inner': true, 'progress-inner-running': countdown}"></div>
  </div>
  <div
    :class="{hero:true, 'hero-green': backgroundColor == 'green', 'hero-red': backgroundColor == 'red', 'hero-orange': backgroundColor=='orange'}">
    <p v-if="!member && !card" class="text-big">Karte auflegen</p>
    <div v-if="member && card">
      <p class="text-medium">Hallo {{ member.name }}</p>
      <p class="text-small">Deine Karte ist gültig bis zum {{ cardValidTo }}</p>
    </div>
  </div>
</template>

<style scoped>
.progress {
  height: 4px;
  width: 100vw;
  position: absolute;
  top: 0;
  background-color: transparent;
}

.progress-inner {
  background-color: white;
  width: 100vw;
  height: 0;
}

@keyframes countdown {
  0% {
    width: 100vw;
  }
  100% {
    width: 0;
  }
}

.progress .progress-inner-running {
  animation: 10s linear 0s 1 countdown;
  transition: height .5s;
  height: 4px;
}

.hero {
  --gradient-bg-1: #2AAA79;
  --gradient-bg-2: #A2C73B;
  height: 100vh;
  text-align: center;
  line-height: initial;
  color: #fff;
  padding: 24px;
  background-image: linear-gradient(90deg, var(--gradient-bg-1) 0%, var(--gradient-bg-2) 100%);
  transition: --gradient-bg-1 2s, --gradient-bg-2 2s;
}

.text-big {
  font-size: 256px;
}

.text-medium {
  font-size: 128px;
}

.text-small {
  font-size: 64px;
}

.hero-green {
  --gradient-bg-1: #2AAA79;
  --gradient-bg-2: #A2C73B;
}

.hero-red {
  --gradient-bg-1: #e6007e;
  --gradient-bg-2: #e40521;
}

.hero-orange {
  --gradient-bg-1: #fcb900;
  --gradient-bg-2: #ff6900;
}
</style>
