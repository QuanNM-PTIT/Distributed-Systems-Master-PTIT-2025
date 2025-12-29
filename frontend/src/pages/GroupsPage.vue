<template>
  <a-row :gutter="[24, 24]">
    <a-col :xs="24" :lg="8">
      <a-card class="panel-card" :bordered="false">
        <a-typography-title :level="4" class="section-title">Create Group</a-typography-title>
        <a-form layout="vertical" @submit.prevent="createGroup">
          <a-form-item label="Group name">
            <a-input v-model:value="groupName" placeholder="Enter group name" />
          </a-form-item>
          <a-button type="primary" block html-type="submit" :loading="creating">Create</a-button>
        </a-form>
      </a-card>

      <a-card class="panel-card" :bordered="false">
        <a-typography-title :level="4" class="section-title">Groups</a-typography-title>
        <a-skeleton v-if="loadingGroups" active :paragraph="{ rows: 3 }" />
        <a-empty v-else-if="groups.length === 0" description="No groups yet." />
        <a-list v-else :data-source="groups" item-layout="horizontal" class="list-compact">
          <template #renderItem="{ item }">
            <a-list-item class="group-item" @click="selectGroup(item)">
              <a-list-item-meta :title="item.name" :description="item.groupId" />
            </a-list-item>
          </template>
        </a-list>
      </a-card>
    </a-col>

    <a-col :xs="24" :lg="16">
      <a-card class="panel-card" :bordered="false">
        <div class="card-title-row">
          <a-typography-title :level="4" class="section-title">Members</a-typography-title>
          <a-button type="link" :disabled="!selectedGroup" @click="refreshMembers">Refresh</a-button>
        </div>
        <a-empty v-if="!selectedGroup" description="Select a group to view members." />
        <template v-else>
          <a-form layout="vertical" @submit.prevent="inviteMembers">
            <a-form-item label="Invite friends">
              <a-select
                v-model:value="inviteIds"
                mode="multiple"
                placeholder="Select friends"
                :options="friendOptions"
              />
            </a-form-item>
            <a-button type="primary" html-type="submit" :loading="inviting">Invite</a-button>
          </a-form>

          <a-divider />
          <a-skeleton v-if="loadingMembers" active :paragraph="{ rows: 4 }" />
          <a-empty v-else-if="members.length === 0" description="No members yet." />
          <a-list v-else :data-source="members" item-layout="horizontal" class="list-compact">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta :title="item.username || item.userId" :description="item.role" />
              </a-list-item>
            </template>
          </a-list>
        </template>
      </a-card>
    </a-col>
  </a-row>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import api from "../services/api";
import { message } from "ant-design-vue";

const groups = ref([]);
const friends = ref([]);
const members = ref([]);
const selectedGroup = ref(null);
const groupName = ref("");
const inviteIds = ref([]);
const loadingGroups = ref(true);
const loadingMembers = ref(false);
const creating = ref(false);
const inviting = ref(false);

const friendOptions = computed(() =>
  friends.value.map((f) => ({ label: f.username || f.userId, value: f.userId }))
);

const fetchGroups = async () => {
  loadingGroups.value = true;
  try {
    const res = await api.get("/groups/list");
    groups.value = res.data.groups || [];
  } finally {
    loadingGroups.value = false;
  }
};

const fetchFriends = async () => {
  const res = await api.get("/friends/list");
  friends.value = res.data.friends || [];
};

const fetchMembers = async () => {
  if (!selectedGroup.value) return;
  loadingMembers.value = true;
  try {
    const res = await api.get(`/groups/${selectedGroup.value.groupId}/members`);
    members.value = res.data.members || [];
  } finally {
    loadingMembers.value = false;
  }
};

const selectGroup = async (group) => {
  selectedGroup.value = group;
  inviteIds.value = [];
  await fetchMembers();
};

const refreshMembers = async () => {
  await fetchMembers();
};

const createGroup = async () => {
  if (!groupName.value.trim()) return;
  creating.value = true;
  try {
    await api.post("/groups", { name: groupName.value.trim() });
    message.success("Group created");
    groupName.value = "";
    await fetchGroups();
  } catch (err) {
    message.error(err?.response?.data?.error || "Create failed");
  } finally {
    creating.value = false;
  }
};

const inviteMembers = async () => {
  if (!selectedGroup.value || inviteIds.value.length === 0) return;
  inviting.value = true;
  try {
    await Promise.all(
      inviteIds.value.map((userId) =>
        api.post("/groups/invite", { groupId: selectedGroup.value.groupId, userId })
      )
    );
    message.success("Invitations sent");
    inviteIds.value = [];
    await fetchMembers();
  } catch (err) {
    message.error(err?.response?.data?.error || "Invite failed");
  } finally {
    inviting.value = false;
  }
};

onMounted(async () => {
  await Promise.all([fetchGroups(), fetchFriends()]);
});
</script>

<style scoped>
.panel-card {
  background: #ffffff;
  border-radius: 16px;
  box-shadow: var(--brand-shadow);
}

.group-item {
  cursor: pointer;
}

.group-item:hover {
  background: rgba(0, 0, 0, 0.02);
}

.card-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.list-compact :deep(.ant-list-item) {
  padding: 12px 0;
}
</style>
