<template>
  <a-row :gutter="[24, 24]" class="chat-layout">
    <a-col :xs="24" :lg="7">
      <a-card class="panel-card sidebar-card" :bordered="false">
        <a-tabs v-model:activeKey="mode">
          <a-tab-pane key="friends" tab="Friends">
            <a-input v-model:value="filter" placeholder="Search friend" class="chat-search" />
            <a-skeleton v-if="loadingFriends" active :paragraph="{ rows: 4 }" />
            <a-empty v-else-if="filteredFriends.length === 0" description="No friends found." />
            <a-list v-else :data-source="filteredFriends" item-layout="horizontal" class="list-compact">
              <template #renderItem="{ item }">
                <a-list-item class="chat-peer" @click="selectPeer(item)">
                  <a-list-item-meta>
                    <template #avatar>
                      <a-avatar class="avatar-red">{{ item.username?.slice(0, 1)?.toUpperCase() || "U" }}</a-avatar>
                    </template>
                    <template #title>
                      <div class="friend-title">
                        <span>{{ item.username || item.userId }}</span>
                        <span class="friend-id">{{ item.userId }}</span>
                      </div>
                    </template>
                  </a-list-item-meta>
                  <template #actions>
                    <a-space size="small">
                      <a-tag :color="presenceMap[item.userId] === 'online' ? 'green' : 'volcano'">
                        {{ presenceMap[item.userId] || 'offline' }}
                      </a-tag>
                      <a-tag :color="peerConnectionLabel(item.userId).color">
                        {{ peerConnectionLabel(item.userId).text }}
                      </a-tag>
                    </a-space>
                  </template>
                </a-list-item>
              </template>
            </a-list>
          </a-tab-pane>
          <a-tab-pane key="groups" tab="Groups">
            <div class="group-toolbar">
              <a-button type="primary" block @click="openGroupModal">Create Group</a-button>
            </div>
            <a-skeleton v-if="loadingGroups" active :paragraph="{ rows: 3 }" />
            <a-empty v-else-if="groups.length === 0" description="No groups found." />
            <a-list v-else :data-source="groups" item-layout="horizontal" class="list-compact">
              <template #renderItem="{ item }">
                <a-list-item class="chat-peer" @click="selectGroup(item)">
                  <a-list-item-meta :title="item.name" :description="item.groupId" />
                </a-list-item>
              </template>
            </a-list>
          </a-tab-pane>
        </a-tabs>
      </a-card>
    </a-col>

    <a-col :xs="24" :lg="17">
      <a-card class="panel-card chat-card" :bordered="false">
        <div class="chat-header">
          <div>
            <div class="chat-peer-name">{{ headerTitle }}</div>
            <div class="chat-peer-id">{{ headerSubtitle }}</div>
          </div>
          <a-space>
            <a-tag v-if="isActive" :color="connectionState === 'connected' ? 'green' : 'volcano'">
              {{ connectionStateLabel }}
            </a-tag>
            <a-button v-if="mode === 'friends' && activePeer" @click="reconnectPeer">Reconnect</a-button>
          </a-space>
        </div>
        <div v-if="mode === 'groups' && activeGroup" class="group-status">
          <a-space wrap>
            <span v-for="member in activeGroupMembers" :key="member.userId" class="group-member">
              <a-badge :status="presenceMap[member.userId] === 'online' ? 'success' : 'default'" />
              <span class="group-member-name">{{ member.username || member.userId }}</span>
            </span>
          </a-space>
        </div>

        <div class="chat-messages" ref="messagesRef">
          <a-empty v-if="isActive && activeMessages.length === 0" description="No messages yet." />
          <a-empty v-else-if="!isActive" :description="mode === 'groups' ? 'Select a group to start chatting.' : 'Select a friend to start chatting.'" />
          <div v-else class="message-list">
            <div
              v-for="msg in activeMessages"
              :key="msg.msgId"
              :class="['message', msg.from === auth.userId ? 'outgoing' : 'incoming']"
            >
              <div class="message-bubble">
                <div v-if="mode === 'groups'" class="message-sender">{{ msg.senderName || msg.from }}</div>
                <template v-if="msg.kind === 'file'">
                  <div class="file-title">{{ msg.name }}</div>
                  <div class="file-meta">{{ formatFileSize(msg.size) }} · {{ msg.mime || 'file' }}</div>
                  <a-progress :percent="msg.progress || 0" size="small" />
                  <a-button v-if="msg.url" type="link" :href="msg.url" target="_blank">Download</a-button>
                </template>
                <div v-else class="message-text">{{ msg.text }}</div>
                <div class="message-meta">
                  L{{ msg.lamport }} · {{ formatTime(msg.clientTimestamp) }}
                  <span v-if="msg.from === auth.userId" class="message-status">· {{ msg.status }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="chat-input">
          <a-input v-model:value="draft" placeholder="Type a message" @pressEnter="sendMessage" />
          <a-space>
            <input ref="fileInputRef" type="file" class="file-input" @change="onFileChange" />
            <a-button @click="triggerFilePicker" :disabled="!isActive">Send file</a-button>
            <a-button type="primary" @click="sendMessage" :disabled="!isActive">Send</a-button>
          </a-space>
        </div>
      </a-card>
    </a-col>
  </a-row>

  <a-modal v-model:open="groupModalOpen" title="Create Group" @ok="createGroup" :confirm-loading="creatingGroup">
    <a-form layout="vertical">
      <a-form-item label="Group name">
        <a-input v-model:value="groupName" placeholder="Enter group name" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { onBeforeRouteLeave } from "vue-router";
import { useAuthStore } from "../stores/auth";
import api from "../services/api";
import signaling from "../services/signaling";
import webrtc from "../services/webrtcService";
import { message as antMessage } from "ant-design-vue";

const auth = useAuthStore();
const mode = ref("friends");

const friends = ref([]);
const groups = ref([]);
const groupMembers = ref({});
const activeGroup = ref(null);
const activeGroupPeers = ref([]);
const groupModalOpen = ref(false);
const groupName = ref("");
const creatingGroup = ref(false);

const presenceMap = ref({});
const loadingFriends = ref(true);
const loadingGroups = ref(true);
const filter = ref("");
const activePeer = ref(null);
const draft = ref("");
const messages = ref({});
const groupMessages = ref({});
const lamports = ref({});
const groupLamports = ref({});
const seenMsgIds = ref({});
const groupSeenMsgIds = ref({});
const groupAckMap = ref({});
const connectionState = ref("disconnected");
const connectionMap = ref({});
const messagesRef = ref(null);
const fileInputRef = ref(null);
const retryTimers = ref({});
const retryAttempts = ref({});
const heartbeatTimers = ref({});
const lastPong = ref({});
const incomingFiles = ref({});

const filteredFriends = computed(() => {
  const term = filter.value.trim().toLowerCase();
  if (!term) return friends.value;
  return friends.value.filter((item) =>
    `${item.username || ""} ${item.userId}`.toLowerCase().includes(term)
  );
});

const isActive = computed(() => (mode.value === "friends" ? !!activePeer.value : !!activeGroup.value));

const activeMessages = computed(() => {
  if (mode.value === "friends") {
    if (!activePeer.value) return [];
    return messages.value[activePeer.value.userId] || [];
  }
  if (!activeGroup.value) return [];
  return groupMessages.value[activeGroup.value.groupId] || [];
});

const activeGroupMembers = computed(() => {
  if (!activeGroup.value) return [];
  return (groupMembers.value[activeGroup.value.groupId] || []).filter(
    (member) => member.userId !== auth.userId
  );
});

const headerTitle = computed(() => {
  if (mode.value === "friends") {
    return activePeer.value?.username || activePeer.value?.userId || "Select a friend";
  }
  return activeGroup.value?.name || "Select a group";
});

const headerSubtitle = computed(() => {
  if (mode.value === "friends") {
    return activePeer.value?.userId || "";
  }
  return activeGroup.value?.groupId || "";
});

const connectionStateLabel = computed(() => {
  switch (connectionState.value) {
    case "connected":
      return "Connected";
    case "connecting":
      return "Connecting";
    default:
      return "Disconnected";
  }
});

const peerConnectionLabel = (peerId) => {
  const state = connectionMap.value[peerId] || "disconnected";
  switch (state) {
    case "connected":
      return { color: "green", text: "P2P connected" };
    case "connecting":
      return { color: "gold", text: "P2P connecting" };
    default:
      return { color: "volcano", text: "P2P disconnected" };
  }
};

const fetchFriends = async () => {
  loadingFriends.value = true;
  try {
    const res = await api.get("/friends/list");
    friends.value = res.data.friends || [];
  } finally {
    loadingFriends.value = false;
  }
};

const fetchGroups = async () => {
  loadingGroups.value = true;
  try {
    const res = await api.get("/groups/list");
    groups.value = res.data.groups || [];
  } finally {
    loadingGroups.value = false;
  }
};

const openGroupModal = () => {
  groupModalOpen.value = true;
};

const createGroup = async () => {
  if (!groupName.value.trim()) return;
  creatingGroup.value = true;
  try {
    await api.post("/groups", { name: groupName.value.trim() });
    antMessage.success("Group created");
    groupName.value = "";
    groupModalOpen.value = false;
    await fetchGroups();
  } catch (err) {
    antMessage.error(err?.response?.data?.error || "Create failed");
  } finally {
    creatingGroup.value = false;
  }
};

const fetchGroupMembers = async (groupId) => {
  const res = await api.get(`/groups/${groupId}/members`);
  groupMembers.value = { ...groupMembers.value, [groupId]: res.data.members || [] };
  return res.data.members || [];
};

const fetchPresence = async () => {
  const res = await api.get("/presence");
  const map = {};
  (res.data.presence || []).forEach((item) => {
    map[item.userId] = item.status;
  });
  presenceMap.value = map;
};

const setPeerConnection = (peerId, state) => {
  connectionMap.value = { ...connectionMap.value, [peerId]: state };
};

const startHeartbeat = (peerId) => {
  if (heartbeatTimers.value[peerId]) return;
  lastPong.value = { ...lastPong.value, [peerId]: Date.now() };
  webrtc.send(peerId, { type: "chat.ping", ts: Date.now() });
  heartbeatTimers.value[peerId] = window.setInterval(() => {
    const last = lastPong.value[peerId] || 0;
    if (Date.now() - last > 9000) {
      setPeerConnection(peerId, "disconnected");
      if (activePeer.value && activePeer.value.userId === peerId) {
        setConnectionState("disconnected");
      }
    } else {
      webrtc.send(peerId, { type: "chat.ping", ts: Date.now() });
    }
  }, 3000);
};

const stopHeartbeat = (peerId) => {
  if (heartbeatTimers.value[peerId]) {
    clearInterval(heartbeatTimers.value[peerId]);
    heartbeatTimers.value[peerId] = null;
  }
};

const setConnectionState = (state) => {
  connectionState.value = state;
};

const disconnectAllPeers = () => {
  webrtc.disconnectAll();
  Object.keys(heartbeatTimers.value).forEach((peerId) => stopHeartbeat(peerId));
  connectionMap.value = {};
  setConnectionState("disconnected");
};

const handleVisibilityChange = () => {
  if (document.hidden) {
    disconnectAllPeers();
  }
};

const handleBeforeUnload = () => {
  disconnectAllPeers();
};

const selectPeer = async (friend) => {
  disconnectAllPeers();
  activeGroup.value = null;
  activeGroupPeers.value = [];
  activePeer.value = friend;
  setConnectionState("connecting");
  await webrtc.connect(friend.userId);
};

const selectGroup = async (group) => {
  disconnectAllPeers();
  activePeer.value = null;
  activeGroup.value = group;
  setConnectionState("connecting");
  const members = await fetchGroupMembers(group.groupId);
  const peers = members.filter((m) => m.userId !== auth.userId).map((m) => m.userId);
  activeGroupPeers.value = peers;
  await Promise.all(peers.map((peerId) => webrtc.connectGroup(peerId, group.groupId)));
};

const reconnectPeer = async () => {
  if (!activePeer.value) return;
  webrtc.disconnect(activePeer.value.userId);
  stopHeartbeat(activePeer.value.userId);
  setPeerConnection(activePeer.value.userId, "disconnected");
  await webrtc.connect(activePeer.value.userId);
  setConnectionState("connecting");
};

const insertMessage = (peerId, msg) => {
  const list = messages.value[peerId] || [];
  list.push(msg);
  list.sort((a, b) => {
    if (a.lamport !== b.lamport) return a.lamport - b.lamport;
    if (a.clientTimestamp !== b.clientTimestamp) return a.clientTimestamp - b.clientTimestamp;
    return a.msgId.localeCompare(b.msgId);
  });
  messages.value = { ...messages.value, [peerId]: list };
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight;
    }
  });
};

const resolveGroupSender = (groupId, userId) => {
  const members = groupMembers.value[groupId] || [];
  const found = members.find((m) => m.userId === userId);
  return found?.username || userId;
};

const insertGroupMessage = (groupId, msg) => {
  const list = groupMessages.value[groupId] || [];
  list.push({ ...msg, senderName: resolveGroupSender(groupId, msg.from) });
  list.sort((a, b) => {
    if (a.lamport !== b.lamport) return a.lamport - b.lamport;
    if (a.clientTimestamp !== b.clientTimestamp) return a.clientTimestamp - b.clientTimestamp;
    return a.msgId.localeCompare(b.msgId);
  });
  groupMessages.value = { ...groupMessages.value, [groupId]: list };
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight;
    }
  });
};

const markSeen = (peerId, msgId) => {
  const seen = seenMsgIds.value[peerId] || new Set();
  if (seen.has(msgId)) return false;
  seen.add(msgId);
  seenMsgIds.value = { ...seenMsgIds.value, [peerId]: seen };
  return true;
};

const markGroupSeen = (groupId, msgId) => {
  const seen = groupSeenMsgIds.value[groupId] || new Set();
  if (seen.has(msgId)) return false;
  seen.add(msgId);
  groupSeenMsgIds.value = { ...groupSeenMsgIds.value, [groupId]: seen };
  return true;
};

const triggerFilePicker = () => {
  if (fileInputRef.value) fileInputRef.value.click();
};

const arrayBufferToBase64 = (buffer) => {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i += 1) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
};

const base64ToUint8Array = (b64) => {
  const binary = atob(b64);
  const len = binary.length;
  const bytes = new Uint8Array(len);
  for (let i = 0; i < len; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
};

const sha256Hex = async (buffer) => {
  const hash = await crypto.subtle.digest("SHA-256", buffer);
  const bytes = new Uint8Array(hash);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
};

const onFileChange = async (event) => {
  const file = event.target.files?.[0];
  event.target.value = "";
  if (!file || !isActive.value) return;
  if (file.size > 25 * 1024 * 1024) {
    antMessage.error("File too large (max 25MB).");
    return;
  }
  await sendFile(file);
};

const sendFile = async (file) => {
  const fileId = crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const buffer = await file.arrayBuffer();
  const checksum = await sha256Hex(buffer);
  const meta = {
    type: "file.meta",
    fileId,
    name: file.name,
    size: file.size,
    mime: file.type || "application/octet-stream",
    checksum,
    clientTimestamp: Date.now()
  };
  if (mode.value === "groups") {
    meta.groupId = activeGroup.value.groupId;
  }
  const msg = {
    kind: "file",
    msgId: fileId,
    from: auth.userId,
    to: mode.value === "friends" ? activePeer.value?.userId : undefined,
    groupId: mode.value === "groups" ? activeGroup.value.groupId : undefined,
    name: file.name,
    size: file.size,
    mime: meta.mime,
    checksum,
    clientTimestamp: meta.clientTimestamp,
    progress: 0,
    status: "sending"
  };

  if (mode.value === "friends") {
    const peerId = activePeer.value.userId;
    if (connectionState.value !== "connected") {
      antMessage.warning("Peer not connected yet.");
      return;
    }
    webrtc.send(peerId, meta);
    insertMessage(peerId, msg);
    await sendFileChunks([peerId], fileId, buffer, file.size, meta);
  } else {
    activeGroupPeers.value.forEach((peerId) => webrtc.send(peerId, meta));
    insertGroupMessage(activeGroup.value.groupId, msg);
    await sendFileChunks(activeGroupPeers.value, fileId, buffer, file.size, meta);
  }
};

const sendFileChunks = async (peers, fileId, buffer, size, meta) => {
  const chunkSize = 16 * 1024;
  let offset = 0;
  let sentBytes = 0;
  while (offset < buffer.byteLength) {
    const slice = buffer.slice(offset, offset + chunkSize);
    const payload = arrayBufferToBase64(slice);
    const chunk = {
      type: "file.chunk",
      fileId,
      offset,
      size: slice.byteLength,
      data: payload
    };
    if (meta.groupId) chunk.groupId = meta.groupId;
    peers.forEach((peerId) => webrtc.send(peerId, chunk));
    offset += chunkSize;
    sentBytes += slice.byteLength;
    const percent = Math.floor((sentBytes / size) * 100);
    if (meta.groupId) {
      const list = groupMessages.value[meta.groupId] || [];
      const updated = list.map((item) =>
        item.msgId === fileId ? { ...item, progress: percent, status: percent === 100 ? "delivered" : "sending" } : item
      );
      groupMessages.value = { ...groupMessages.value, [meta.groupId]: updated };
    } else {
      const peerId = peers[0];
      const list = messages.value[peerId] || [];
      const updated = list.map((item) =>
        item.msgId === fileId ? { ...item, progress: percent, status: percent === 100 ? "delivered" : "sending" } : item
      );
      messages.value = { ...messages.value, [peerId]: updated };
    }
    if (offset % (chunkSize * 20) === 0) {
      await new Promise((resolve) => setTimeout(resolve, 0));
    }
  }
};

const updateGroupStatus = (groupId, msgId) => {
  const total = activeGroupPeers.value.length;
  const ackSet = groupAckMap.value[groupId]?.[msgId];
  const delivered = ackSet ? ackSet.size : 0;
  const list = groupMessages.value[groupId] || [];
  const updated = list.map((item) =>
    item.msgId === msgId ? { ...item, status: `delivered ${delivered}/${total}` } : item
  );
  groupMessages.value = { ...groupMessages.value, [groupId]: updated };
};

const sendMessage = () => {
  if (!isActive.value || !draft.value.trim()) return;

  if (mode.value === "friends") {
    if (connectionState.value !== "connected") {
      antMessage.warning("Peer not connected yet.");
      return;
    }
    const peerId = activePeer.value.userId;
    const currentLamport = lamports.value[peerId] || 0;
    const nextLamport = currentLamport + 1;
    lamports.value = { ...lamports.value, [peerId]: nextLamport };

    const msg = {
      type: "chat.message",
      msgId: crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`,
      from: auth.userId,
      to: peerId,
      text: draft.value.trim(),
      clientTimestamp: Date.now(),
      lamport: nextLamport,
      status: "sent",
      kind: "text"
    };

    const sent = webrtc.send(peerId, msg);
    if (!sent) {
      antMessage.error("DataChannel not ready. Try reconnect.");
      msg.status = "failed";
      insertMessage(peerId, msg);
      return;
    }

    insertMessage(peerId, msg);
    draft.value = "";
    return;
  }

  const groupId = activeGroup.value.groupId;
  const currentLamport = groupLamports.value[groupId] || 0;
  const nextLamport = currentLamport + 1;
  groupLamports.value = { ...groupLamports.value, [groupId]: nextLamport };

  const msg = {
    type: "group.message",
    msgId: crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`,
    groupId,
    from: auth.userId,
    senderName: resolveGroupSender(groupId, auth.userId),
    text: draft.value.trim(),
    clientTimestamp: Date.now(),
    lamport: nextLamport,
    status: `delivered 0/${activeGroupPeers.value.length}`,
    kind: "text"
  };

  activeGroupPeers.value.forEach((peerId) => {
    webrtc.send(peerId, msg);
  });

  groupAckMap.value = {
    ...groupAckMap.value,
    [groupId]: {
      ...(groupAckMap.value[groupId] || {}),
      [msg.msgId]: new Set()
    }
  };

  insertGroupMessage(groupId, msg);
  draft.value = "";
};

const formatTime = (ms) => new Date(ms).toLocaleTimeString();

const formatFileSize = (size) => {
  if (!size && size !== 0) return "";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

onMounted(async () => {
  document.addEventListener("visibilitychange", handleVisibilityChange);
  window.addEventListener("beforeunload", handleBeforeUnload);
  webrtc.init(signaling);
  signaling.onMessage((msg) => {
    if (msg.type === "chat.busy") {
      const fromId = msg.from || msg.payload?.from;
      if (fromId) {
        setPeerConnection(fromId, "disconnected");
        if (activePeer.value?.userId === fromId) {
          setConnectionState("disconnected");
        }
      }
      antMessage.warning("Peer is busy with another chat.");
      return;
    }
    if (msg.type?.startsWith("signal.") || msg.type?.startsWith("group.signal.")) {
      const fromId = msg.from || msg.payload?.from;
      if (msg.type === "signal.offer" && fromId) {
        const busy =
          activePeer.value &&
          activePeer.value.userId !== fromId &&
          (connectionState.value === "connected" || connectionState.value === "connecting");
        if (busy) {
          signaling.send({ type: "chat.busy", to: fromId, payload: { from: auth.userId } });
          return;
        }
        if (!activePeer.value || activePeer.value.userId !== fromId) {
          activePeer.value = friends.value.find((f) => f.userId === fromId) || { userId: fromId };
          mode.value = "friends";
          setConnectionState("connecting");
        }
      }
      webrtc.handleSignal(msg);
    }
    if (msg.type === "presence.update") {
      const { userId, status } = msg.payload || {};
      if (!userId) return;
      presenceMap.value = { ...presenceMap.value, [userId]: status };
      if (status === "offline") {
        setPeerConnection(userId, "disconnected");
        stopHeartbeat(userId);
        if (activePeer.value?.userId === userId) {
          setConnectionState("disconnected");
        }
        webrtc.disconnect(userId);
      }
    }
  });

  webrtc.on("message", ({ peerId, data }) => {
    if (!data || !data.type) return;
    if (data.type === "chat.ping") {
      webrtc.send(peerId, { type: "chat.pong", ts: Date.now() });
      return;
    }
    if (data.type === "chat.pong") {
      lastPong.value = { ...lastPong.value, [peerId]: Date.now() };
      setPeerConnection(peerId, "connected");
      if (activePeer.value && activePeer.value.userId === peerId) {
        setConnectionState("connected");
      }
      if (activeGroup.value && activeGroupPeers.value.includes(peerId)) {
        setConnectionState("connected");
      }
      return;
    }
    if (data.type === "chat.ack") {
      const list = messages.value[peerId] || [];
      const updated = list.map((item) =>
        item.msgId === data.msgId ? { ...item, status: "delivered" } : item
      );
      messages.value = { ...messages.value, [peerId]: updated };
      return;
    }
    if (data.type === "group.ack") {
      const groupId = data.groupId;
      if (!groupId) return;
      const groupMap = groupAckMap.value[groupId] || {};
      const ackSet = groupMap[data.msgId] || new Set();
      ackSet.add(peerId);
      groupAckMap.value = { ...groupAckMap.value, [groupId]: { ...groupMap, [data.msgId]: ackSet } };
      updateGroupStatus(groupId, data.msgId);
      return;
    }
    if (data.type === "chat.message") {
      if (!markSeen(peerId, data.msgId)) return;
      const current = lamports.value[peerId] || 0;
      const incomingLamport = data.lamport || 0;
      const nextLamport = Math.max(current, incomingLamport) + 1;
      lamports.value = { ...lamports.value, [peerId]: nextLamport };
      insertMessage(peerId, { ...data, status: "delivered", kind: "text" });
      webrtc.send(peerId, { type: "chat.ack", msgId: data.msgId });
      return;
    }
    if (data.type === "group.message" && data.groupId) {
      if (!markGroupSeen(data.groupId, data.msgId)) return;
      const current = groupLamports.value[data.groupId] || 0;
      const incomingLamport = data.lamport || 0;
      const nextLamport = Math.max(current, incomingLamport) + 1;
      groupLamports.value = { ...groupLamports.value, [data.groupId]: nextLamport };
      insertGroupMessage(data.groupId, { ...data, kind: "text" });
      webrtc.send(peerId, { type: "group.ack", groupId: data.groupId, msgId: data.msgId });
      return;
    }
    if (data.type === "file.meta") {
      const key = data.groupId ? `${data.groupId}:${data.fileId}` : data.fileId;
      if (incomingFiles.value[key]) return;
      incomingFiles.value = {
        ...incomingFiles.value,
        [key]: { meta: data, chunks: [], received: 0 }
      };
      const msg = {
        kind: "file",
        msgId: data.fileId,
        from: peerId,
        groupId: data.groupId,
        name: data.name,
        size: data.size,
        mime: data.mime,
        checksum: data.checksum,
        clientTimestamp: data.clientTimestamp,
        progress: 0,
        status: "receiving"
      };
      if (data.groupId) {
        insertGroupMessage(data.groupId, msg);
      } else {
        insertMessage(peerId, msg);
      }
      return;
    }
    if (data.type === "file.chunk") {
      const key = data.groupId ? `${data.groupId}:${data.fileId}` : data.fileId;
      const transfer = incomingFiles.value[key];
      if (!transfer) return;
      const chunkBytes = base64ToUint8Array(data.data);
      transfer.chunks.push(chunkBytes);
      transfer.received += chunkBytes.byteLength;
      const percent = Math.floor((transfer.received / transfer.meta.size) * 100);
      if (data.groupId) {
        const list = groupMessages.value[data.groupId] || [];
        const updated = list.map((item) =>
          item.msgId === data.fileId ? { ...item, progress: percent, status: percent === 100 ? "ready" : "receiving" } : item
        );
        groupMessages.value = { ...groupMessages.value, [data.groupId]: updated };
      } else {
        const list = messages.value[peerId] || [];
        const updated = list.map((item) =>
          item.msgId === data.fileId ? { ...item, progress: percent, status: percent === 100 ? "ready" : "receiving" } : item
        );
        messages.value = { ...messages.value, [peerId]: updated };
      }
      if (transfer.received >= transfer.meta.size) {
        const blob = new Blob(transfer.chunks, { type: transfer.meta.mime });
        const url = URL.createObjectURL(blob);
        if (data.groupId) {
          const list = groupMessages.value[data.groupId] || [];
          const updated = list.map((item) =>
            item.msgId === data.fileId ? { ...item, url, status: "ready", progress: 100 } : item
          );
          groupMessages.value = { ...groupMessages.value, [data.groupId]: updated };
        } else {
          const list = messages.value[peerId] || [];
          const updated = list.map((item) =>
            item.msgId === data.fileId ? { ...item, url, status: "ready", progress: 100 } : item
          );
          messages.value = { ...messages.value, [peerId]: updated };
        }
        delete incomingFiles.value[key];
      }
    }
  });

  webrtc.on("status", ({ peerId, state }) => {
    if (state === "channel-open") {
      startHeartbeat(peerId);
      setPeerConnection(peerId, "connected");
      if (activeGroup.value && activeGroupPeers.value.includes(peerId)) {
        setConnectionState("connected");
      }
    }
    if (state === "channel-closed") {
      stopHeartbeat(peerId);
      setPeerConnection(peerId, "disconnected");
      if (activeGroup.value && activeGroupPeers.value.includes(peerId)) {
        const stillConnected = activeGroupPeers.value.some(
          (id) => connectionMap.value[id] === "connected"
        );
        if (!stillConnected) {
          setConnectionState("disconnected");
        }
      }
    }

    if (!activePeer.value || activePeer.value.userId !== peerId) return;
    if (state === "channel-open") {
      setConnectionState("connected");
      if (retryTimers.value[peerId]) {
        clearTimeout(retryTimers.value[peerId]);
        retryTimers.value[peerId] = null;
      }
      retryAttempts.value[peerId] = 0;
    } else if (state === "disconnected" || state === "failed" || state === "channel-closed") {
      setConnectionState("disconnected");
      const attempts = retryAttempts.value[peerId] || 0;
      const delay = Math.min(2000 * Math.pow(2, attempts), 10000);
      if (retryTimers.value[peerId]) return;
      retryTimers.value[peerId] = setTimeout(async () => {
        retryTimers.value[peerId] = null;
        retryAttempts.value[peerId] = attempts + 1;
        if (activePeer.value && activePeer.value.userId === peerId) {
          setConnectionState("connecting");
          await webrtc.connect(peerId);
        }
      }, delay);
    }
  });

  await Promise.all([fetchFriends(), fetchGroups(), fetchPresence()]);
});

watch(activePeer, (current, previous) => {
  if (previous && (!current || previous.userId !== current.userId)) {
    webrtc.disconnect(previous.userId);
    stopHeartbeat(previous.userId);
    setPeerConnection(previous.userId, "disconnected");
  }
});

watch(mode, (nextMode) => {
  if (nextMode === "friends") {
    activeGroup.value = null;
    activeGroupPeers.value = [];
  } else {
    activePeer.value = null;
  }
  disconnectAllPeers();
});

onBeforeRouteLeave(() => {
  disconnectAllPeers();
});

onBeforeUnmount(() => {
  document.removeEventListener("visibilitychange", handleVisibilityChange);
  window.removeEventListener("beforeunload", handleBeforeUnload);
  disconnectAllPeers();
});
</script>

<style scoped>
.chat-layout :deep(.ant-card) {
  border-radius: 16px;
  box-shadow: var(--brand-shadow);
}

.panel-card {
  background: #ffffff;
}

.sidebar-card {
  min-height: 620px;
}

.chat-search {
  margin-bottom: 12px;
}

.chat-peer {
  cursor: pointer;
}

.chat-peer:hover {
  background: rgba(0, 0, 0, 0.02);
}

.group-toolbar {
  margin-bottom: 12px;
}

.chat-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 620px;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.group-status {
  padding: 12px 0;
  border-bottom: 1px dashed rgba(0, 0, 0, 0.08);
}

.group-member {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.group-member-name {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
}

.message-sender {
  font-size: 12px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.6);
  margin-bottom: 4px;
}

.chat-peer-name {
  font-weight: 600;
  font-size: 16px;
}

.chat-peer-id {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.chat-messages {
  flex: 1;
  overflow: auto;
  padding: 12px 4px;
}

.message-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message {
  display: flex;
}

.message.outgoing {
  justify-content: flex-end;
}

.message-bubble {
  max-width: 70%;
  padding: 10px 14px;
  border-radius: 14px;
  background: rgba(211, 32, 39, 0.12);
}

.message.incoming .message-bubble {
  background: rgba(0, 0, 0, 0.05);
}

.message-text {
  white-space: pre-wrap;
}

.message-meta {
  margin-top: 6px;
  font-size: 11px;
  color: rgba(0, 0, 0, 0.45);
}

.message-status {
  text-transform: capitalize;
}

.chat-input {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
}

.list-compact :deep(.ant-list-item) {
  padding: 12px 0;
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

.file-input {
  display: none;
}

.file-title {
  font-weight: 600;
  margin-bottom: 4px;
}

.file-meta {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.5);
  margin-bottom: 6px;
}
</style>
