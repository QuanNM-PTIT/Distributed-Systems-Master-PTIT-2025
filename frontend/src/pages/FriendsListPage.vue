<template>
  <a-card class="panel-card" :bordered="false">
    <div class="card-title-row">
      <a-typography-title :level="4" class="section-title">Danh sách bạn bè</a-typography-title>
      <a-button type="link" @click="state.fetchFriends">Refresh</a-button>
    </div>
    <a-skeleton v-if="state.loadingFriends" active :paragraph="{ rows: 4 }" />
    <a-empty v-else-if="state.friends.length === 0" description="No friends yet.">
      <template #image>
        <div class="empty-illustration">🤝</div>
      </template>
      <a-button type="primary" ghost @click="state.fetchFriends">Find friends</a-button>
    </a-empty>
    <a-list v-else :data-source="state.friends" item-layout="horizontal">
      <template #renderItem="{ item }">
        <a-list-item>
          <a-list-item-meta>
            <template #avatar>
              <a-avatar class="avatar-red">{{ item.username?.slice(0, 1)?.toUpperCase() || "U" }}</a-avatar>
            </template>
            <template #title>
              <div class="friend-title">
                <span>{{ item.username || "Unknown" }}</span>
                <span class="friend-id">{{ item.userId }}</span>
              </div>
            </template>
          </a-list-item-meta>
          <template #actions>
            <a-tag :color="state.presenceMap[item.userId] === 'online' ? 'green' : 'volcano'">
              {{ state.presenceMap[item.userId] || 'offline' }}
            </a-tag>
          </template>
        </a-list-item>
      </template>
    </a-list>
  </a-card>
</template>

<script setup>
import { inject } from "vue";

const state = inject("friendsState");
</script>

<style scoped>
.panel-card {
  background: #ffffff;
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
</style>
