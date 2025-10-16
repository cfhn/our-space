import {createRouter, createWebHistory} from 'vue-router'
import HomeView from './views/home/HomeView.vue'
import MembersView from "@/views/members/MembersView.vue";
import MembersEditView from "@/views/members/MemberEditView.vue";
import CardsView from "@/views/cards/CardsView.vue";
import CardEditView from "@/views/cards/CardEditView.vue";
import TerminalView from "@/views/terminal/TerminalView.vue";

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/members',
      name: 'members',
      component: MembersView,
    },
    {
      path: '/members/:id',
      name: 'member-edit',
      component: MembersEditView,
      props: true,
    },
    {
      path: '/members/new',
      name: 'member-create',
      component: MembersEditView,
    },
    {
      path: '/cards',
      name: 'cards',
      component: CardsView,
    },
    {
      path: '/cards/:id',
      name: 'card-edit',
      component: CardEditView,
      props: true,
    },
    {
      path: '/cards/new',
      name: 'card-create',
      component: CardEditView,
    },
    {
      path: '/terminal',
      name: 'terminal',
      component: TerminalView,
      meta: {
        navbar: false,
      }
    }
  ],
})

export default router
