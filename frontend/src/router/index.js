import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";
import LoginPage from "../pages/LoginPage.vue";
import RegisterPage from "../pages/RegisterPage.vue";
import FriendsPage from "../pages/FriendsPage.vue";
import GroupsPage from "../pages/GroupsPage.vue";
import ChatPage from "../pages/ChatPage.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/friends" },
    { path: "/login", component: LoginPage },
    { path: "/register", component: RegisterPage },
    { path: "/friends", component: FriendsPage, meta: { requiresAuth: true } },
    { path: "/groups", component: GroupsPage, meta: { requiresAuth: true } },
    { path: "/chat", component: ChatPage, meta: { requiresAuth: true } }
  ]
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  auth.restore();
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return "/login";
  }
  if ((to.path === "/login" || to.path === "/register") && auth.isAuthenticated) {
    return "/friends";
  }
  return true;
});

export default router;
