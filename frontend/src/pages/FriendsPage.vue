<template>
  <a-row :gutter="[24, 24]" class="friends-layout">
    <a-col :xs="24" :lg="8" class="left-stack">
      <a-card class="panel-card profile-card" :bordered="false">
        <div class="profile-header">
          <div>
            <a-typography-title :level="4" class="section-title">Your Profile</a-typography-title>
            <a-typography-text type="secondary">{{ profileUsername || "Loading..." }}</a-typography-text>
          </div>
          <a-badge :status="signalConnected ? 'success' : 'error'" :text="signalConnected ? 'Signaling online' : 'Signaling offline'" />
        </div>
        <div class="profile-id">
          <span class="profile-label">User ID</span>
          <span class="profile-value">{{ auth.userId }}</span>
        </div>
      </a-card>

      <a-card class="panel-card" :bordered="false">
        <a-typography-title :level="4" class="section-title">Add Friend</a-typography-title>
        <a-form layout="vertical" @submit.prevent="sendRequest">
          <a-form-item label="Friend username">
            <a-input v-model:value="friendQuery" placeholder="Enter username" />
          </a-form-item>
          <a-button type="primary" block html-type="submit" :loading="loading">Send request</a-button>
        </a-form>
        <div v-if="searchVisible" class="search-panel">
          <a-spin :spinning="searchLoading">
            <a-list
              size="small"
              :data-source="searchResults"
              :locale="{ emptyText: 'No users found.' }"
            >
              <template #renderItem="{ item }">
                <a-list-item class="search-item" @click="selectUser(item)">
                  <a-space>
                    <a-avatar class="avatar-red">{{ item.username?.slice(0, 1)?.toUpperCase() || "U" }}</a-avatar>
                    <div>
                      <div class="search-name">{{ item.username }}</div>
                      <div class="search-id">{{ item.userId }}</div>
                    </div>
                  </a-space>
                </a-list-item>
              </template>
            </a-list>
          </a-spin>
        </div>
        <a-alert v-if="requestStatus" class="section-title" type="info" :message="requestStatus" show-icon />
      </a-card>
    </a-col>

    <a-col :xs="24" :lg="16">
      <a-row :gutter="[24, 24]">
        <a-col :xs="24" :lg="12">
          <a-card class="panel-card" :bordered="false">
            <div class="card-title-row">
              <a-typography-title :level="4" class="section-title">Lời mời kết bạn</a-typography-title>
              <a-button type="link" @click="fetchRequests">Refresh</a-button>
            </div>
            <a-skeleton v-if="loadingRequests" active :paragraph="{ rows: 3 }" />
            <a-empty v-else-if="requests.length === 0" description="No pending requests." />
            <a-list v-else :data-source="requests" item-layout="horizontal" class="list-compact">
              <template #renderItem="{ item }">
                <a-list-item>
                  <a-list-item-meta :title="item.fromUserId" :description="formatTime(item.createdAt)" />
                  <template #actions>
                    <a-button type="primary" @click="acceptRequest(item.fromUserId)">Accept</a-button>
                  </template>
                </a-list-item>
              </template>
            </a-list>
          </a-card>
        </a-col>

        <a-col :xs="24" :lg="12">
          <a-card class="panel-card" :bordered="false">
            <div class="card-title-row">
              <a-typography-title :level="4" class="section-title">Danh sách bạn bè</a-typography-title>
              <a-button type="link" @click="fetchFriends">Refresh</a-button>
            </div>
            <a-skeleton v-if="loadingFriends" active :paragraph="{ rows: 4 }" />
            <a-empty v-else-if="friends.length === 0" description="No friends yet.">
              <template #image>
                <div class="empty-illustration">🤝</div>
              </template>
              <a-button type="primary" ghost @click="fetchFriends">Find friends</a-button>
            </a-empty>
            <a-list v-else :data-source="friends" item-layout="horizontal" class="list-compact">
              <template #renderItem="{ item }">
                <a-list-item>
                  <a-list-item-meta>
                    <template #avatar>
                      <a-avatar class="avatar-red">{{ item.username?.slice(0, 1)?.toUpperCase() || "U" }}</a-avatar>
                    </template>
                    <template #title>
                      <div class="friend-title">
                        <span>{{ item.username || item.userId || "Unknown" }}</span>
                        <span class="friend-id">{{ item.userId }}</span>
                      </div>
                    </template>
                  </a-list-item-meta>
                  <template #actions>
                    <a-tag :color="presenceMap[item.userId] === 'online' ? 'green' : 'volcano'">
                      {{ presenceMap[item.userId] || 'offline' }}
                    </a-tag>
                  </template>
                </a-list-item>
              </template>
            </a-list>
          </a-card>
        </a-col>
      </a-row>
    </a-col>
  </a-row>
</template>

<script setup>
import { onMounted, ref, reactive, watch } from "vue";
import { useAuthStore } from "../stores/auth";
import api from "../services/api";
import signaling from "../services/signaling";
import { message } from "ant-design-vue";

const auth = useAuthStore();
const friends = ref([]);
const requests = ref([]);
const profileUsername = ref("");
const presenceMap = reactive({});
const friendQuery = ref("");
const selectedUser = ref(null);
const searchResults = ref([]);
const searchLoading = ref(false);
const searchVisible = ref(false);
let searchTimer = null;
const loading = ref(false);
const requestStatus = ref("");
const signalConnected = ref(false);
const loadingFriends = ref(true);
const loadingRequests = ref(true);

const fetchFriends = async () => {
  loadingFriends.value = true;
  try {
    const res = await api.get("/friends/list");
    friends.value = res.data.friends || [];
  } finally {
    loadingFriends.value = false;
  }
};

const fetchRequests = async () => {
  loadingRequests.value = true;
  try {
    const res = await api.get("/friends/requests");
    requests.value = res.data.requests || [];
  } finally {
    loadingRequests.value = false;
  }
};

const fetchPresence = async () => {
  const res = await api.get("/presence");
  (res.data.presence || []).forEach((item) => {
    presenceMap[item.userId] = item.status;
  });
};

const fetchProfile = async () => {
  try {
    const res = await api.get("/users/me");
    profileUsername.value = res.data.username || "";
  } catch {
    profileUsername.value = "";
  }
};

const sendRequest = async () => {
  if (!selectedUser.value) {
    message.warning("Please select a user from search results.");
    return;
  }
  loading.value = true;
  requestStatus.value = "";
  try {
    await api.post("/friends/request", { toUserId: selectedUser.value.userId });
    requestStatus.value = "Request sent.";
    message.success("Friend request sent.");
    friendQuery.value = "";
    selectedUser.value = null;
    searchResults.value = [];
    searchVisible.value = false;
    await fetchRequests();
  } catch (err) {
    requestStatus.value = err?.response?.data?.error || "Request failed";
    message.error(requestStatus.value);
  } finally {
    loading.value = false;
  }
};

const acceptRequest = async (fromUserId) => {
  try {
    await api.post("/friends/accept", { fromUserId });
    message.success("Friend request accepted.");
    await fetchRequests();
    await fetchFriends();
  } catch (err) {
    message.error(err?.response?.data?.error || "Accept failed");
  }
};

onMounted(async () => {
  await Promise.all([fetchFriends(), fetchRequests(), fetchPresence(), fetchProfile()]);

  signaling.onMessage((msg) => {
    if (msg.type === "ws.connected") {
      signalConnected.value = true;
    }
    if (msg.type === "ws.disconnected") {
      signalConnected.value = false;
    }
    if (msg.type === "presence.update") {
      const payload = msg.payload || {};
      if (payload.userId) {
        presenceMap[payload.userId] = payload.status || "offline";
      }
    }
  });
});

const formatTime = (unixSeconds) => {
  if (!unixSeconds) return "Just now";
  return new Date(unixSeconds * 1000).toLocaleString();
};

const selectUser = (user) => {
  selectedUser.value = user;
  friendQuery.value = user.username;
  searchVisible.value = false;
};

const runSearch = async (value) => {
  if (!value || value.trim().length < 2) {
    searchResults.value = [];
    searchVisible.value = false;
    return;
  }
  searchLoading.value = true;
  searchVisible.value = true;
  try {
    const res = await api.get("/users/search", { params: { query: value.trim() } });
    searchResults.value = res.data.users || [];
  } catch (err) {
    searchResults.value = [];
  } finally {
    searchLoading.value = false;
  }
};

watch(friendQuery, (value) => {
  selectedUser.value = null;
  if (searchTimer) {
    clearTimeout(searchTimer);
  }
  searchTimer = window.setTimeout(() => {
    runSearch(value);
  }, 2000);
});
</script>

<style scoped>
.friends-layout :deep(.ant-card) {
  border-radius: 16px;
  box-shadow: var(--brand-shadow);
}

.panel-card {
  background: #ffffff;
}

.profile-card {
  background: linear-gradient(135deg, rgba(211, 32, 39, 0.14), rgba(255, 255, 255, 0.98));
}

.profile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.profile-id {
  margin-top: 16px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.04);
  display: grid;
  gap: 4px;
}

.profile-label {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.5);
}

.profile-value {
  font-weight: 600;
}

.left-stack {
  display: grid;
  gap: 24px;
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.list-compact :deep(.ant-list-item) {
  padding: 12px 0;
}

.list-compact :deep(.ant-list-item-meta-title) {
  font-weight: 600;
}

.friend-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.friend-id {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.avatar-red {
  background: var(--brand-red);
  color: white;
}

.search-panel {
  margin-top: 12px;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 12px;
  background: #ffffff;
  max-height: 220px;
  overflow: auto;
}

.search-item {
  cursor: pointer;
}

.search-item:hover {
  background: rgba(211, 32, 39, 0.06);
}

.search-name {
  font-weight: 600;
}

.search-id {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

@media (max-width: 991px) {
  .profile-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
