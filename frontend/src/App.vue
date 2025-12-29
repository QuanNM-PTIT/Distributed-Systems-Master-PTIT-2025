<template>
  <a-layout class="app-shell">
    <template v-if="auth.isAuthenticated">
      <a-layout-sider class="app-sider" width="240">
        <div class="brand-block">
          <div class="brand-mark">G11</div>
          <div>
            <div class="brand-title">Group 11</div>
            <div class="brand-subtitle">Hybrid P2P Chat</div>
          </div>
        </div>
        <a-menu
          mode="inline"
          class="app-menu"
          :selectedKeys="[selectedKey]"
          @click="onMenuClick"
        >
          <a-menu-item key="/friends">Friends</a-menu-item>
          <a-menu-item key="/chat">Chats</a-menu-item>
        </a-menu>
        <div class="sider-footer">
          <a-typography-text type="secondary" class="muted">
            {{ auth.userId }}
          </a-typography-text>
          <a-button type="primary" danger block @click="logout">Logout</a-button>
        </div>
      </a-layout-sider>
      <a-layout>
        <a-layout-header class="app-header">
          <div class="page-title">{{ pageTitle }}</div>
        </a-layout-header>
        <a-layout-content class="app-content">
          <RouterView />
        </a-layout-content>
      </a-layout>
    </template>
    <template v-else>
      <a-layout-content class="auth-content">
        <RouterView />
      </a-layout-content>
    </template>
  </a-layout>
</template>

<script setup>
import { computed, onMounted } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import { useAuthStore } from "./stores/auth";

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

onMounted(() => {
  auth.restore();
});

const selectedKey = computed(() => route.path);
const pageTitle = computed(() => {
  switch (route.path) {
    case "/friends":
      return "Friends";
    case "/groups":
      return "Groups";
    case "/chat":
      return "Chats";
    default:
      return "Group 11";
  }
});

const onMenuClick = ({ key }) => {
  router.push(key);
};

const logout = () => {
  auth.logout();
  router.push("/login");
};
</script>
