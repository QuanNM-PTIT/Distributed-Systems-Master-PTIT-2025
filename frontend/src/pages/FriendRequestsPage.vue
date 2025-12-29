<template>
  <a-card class="panel-card" :bordered="false">
    <div class="card-title-row">
    <a-typography-title :level="4" class="section-title">Friend Requests</a-typography-title>
      <a-button type="link" @click="state.fetchRequests">Refresh</a-button>
    </div>
    <a-skeleton v-if="state.loadingRequests" active :paragraph="{ rows: 3 }" />
    <a-empty v-else-if="state.requests.length === 0" description="No pending requests." />
    <a-list v-else :data-source="state.requests" item-layout="horizontal">
      <template #renderItem="{ item }">
        <a-list-item>
          <a-list-item-meta :title="item.fromUserId" :description="state.formatTime(item.createdAt)" />
          <template #actions>
            <a-button type="primary" @click="state.acceptRequest(item.fromUserId)">Accept</a-button>
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
</style>
