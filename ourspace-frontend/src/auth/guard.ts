import type { NavigationGuard } from 'vue-router'
import { isLoggedIn } from '@/auth/token.ts'

export const authGuard: NavigationGuard = (to) => {
  if (isLoggedIn()) {
    if (to.name === 'login') {
      return { name: 'home' }
    }

    return
  }

  if (to.meta.authenticated !== false) {
    return { name: 'login' }
  }
}
