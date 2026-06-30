<template>
  <div :class="[$style.container, orientation === 'horizontal' ? $style.horizontal : '']">
    <div :class="$style.header">
      <span :class="$style.title">Sessions</span>
      <div :class="$style.headerActions">
        <a-button size="small" @click="handleCreate">
          <template #icon><PlusOutlined /></template>
        </a-button>
      </div>
    </div>

    <div v-if="sessionStore.loading" :class="$style.loading">
      <a-spin size="small" />
    </div>

    <div v-else :class="$style.list">
      <div
        v-for="session in sessionStore.sessions"
        :key="session.id"
        :class="[
          $style.sessionItem,
          {
            [$style.selected]: session.id === sessionStore.selectedSessionId,
            [$style.activated]: session.is_active,
          },
        ]"
        @click="handleSelect(session.id)"
        @contextmenu.prevent="showContextMenu($event, session)"
      >
        <div :class="$style.sessionIcon">
          <FolderOutlined v-if="session.id !== sessionStore.selectedSessionId" />
          <FolderOpenFilled v-else />
          <span v-if="session.is_active" :class="$style.recordingDot" aria-hidden="true" />
        </div>
        <div :class="$style.sessionInfo">
          <div :class="$style.sessionNameRow">
            <span :class="$style.sessionName">{{ session.name }}</span>
            <span v-if="session.id === sessionStore.selectedSessionId" :class="$style.viewBadge">Viewing</span>
            <span v-if="session.is_active" :class="$style.activeBadge">Activated</span>
          </div>
          <div :class="$style.sessionMeta">
            {{ formatTime(session.updated_at) }}
          </div>
        </div>
      </div>

      <div v-if="sessionStore.sessions.length === 0" :class="$style.empty">
        No sessions yet
      </div>
    </div>

    <!-- Context Menu -->
    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="session-context-menu"
        :style="contextMenuStyle"
        @click.stop
      >
        <div
          class="session-menu-item"
          :class="{ 'session-menu-item-disabled': contextMenu.session?.is_active }"
          @click="handleActivate"
        >
          <CheckCircleOutlined /> Activate Session
        </div>
        <div class="session-menu-divider" />
        <div class="session-menu-item" @click="handleRename">
          <EditOutlined /> Rename
        </div>
        <div class="session-menu-divider" />
        <div class="session-menu-item session-menu-item-danger" @click="handleDelete">
          <DeleteOutlined /> Delete
        </div>
      </div>
    </Teleport>

    <!-- Rename Modal -->
    <a-modal
      v-model:open="renameModalVisible"
      title="Rename Session"
      @ok="submitRename"
      @cancel="cancelRename"
    >
      <a-input
        v-model:value="renameValue"
        placeholder="Session name"
        @pressEnter="submitRename"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { message } from 'ant-design-vue';
import {
  PlusOutlined,
  FolderOutlined,
  FolderOpenFilled,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons-vue';
import { useSessionStore } from '@/stores/sessionStore';
import { useAgentStore } from '@/stores/agentStore';
import type { Session } from '@/types';
import dayjs from 'dayjs';

const props = withDefaults(defineProps<{
  orientation?: 'vertical' | 'horizontal';
}>(), {
  orientation: 'vertical',
});

const orientation = props.orientation;

const emit = defineEmits<{
  (e: 'sessionChange', sessionId: string): void;
}>();

const sessionStore = useSessionStore();
const agentStore = useAgentStore();

const contextMenu = ref<{
  visible: boolean;
  x: number;
  y: number;
  session: Session | null;
}>({
  visible: false,
  x: 0,
  y: 0,
  session: null,
});

const renameModalVisible = ref(false);
const renameValue = ref('');

const contextMenuStyle = computed(() => {
  const menuWidth = 140;
  const menuHeight = 120;
  let x = contextMenu.value.x;
  let y = contextMenu.value.y;

  if (x + menuWidth > window.innerWidth) {
    x = window.innerWidth - menuWidth - 10;
  }
  if (y + menuHeight > window.innerHeight) {
    y = window.innerHeight - menuHeight - 10;
  }

  return {
    left: `${x}px`,
    top: `${y}px`,
  };
});

const formatTime = (time: string) => {
  if (!time) return '';
  const d = dayjs(time);
  const now = dayjs();
  if (d.isSame(now, 'day')) {
    return d.format('HH:mm');
  }
  if (d.isSame(now, 'year')) {
    return d.format('MM-DD HH:mm');
  }
  return d.format('YYYY-MM-DD');
};

const showContextMenu = (event: MouseEvent, session: Session) => {
  event.stopPropagation(); // Prevent document contextmenu listener from closing menu immediately
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    session,
  };
};

const hideContextMenu = () => {
  contextMenu.value.visible = false;
};

const handleCreate = async () => {
  try {
    const name = `Session ${sessionStore.sessions.length + 1}`;
    const session = await sessionStore.createSession(name);
    if (session) {
      await sessionStore.activateSession(session.id);
      emit('sessionChange', session.id);
      message.success('Session created');
    }
  } catch (error) {
    message.error('Failed to create session');
  }
};

const handleSelect = async (sessionId: string) => {
  sessionStore.selectSession(sessionId);
  emit('sessionChange', sessionId);
};

const handleActivate = async () => {
  const session = contextMenu.value.session;
  if (!session || session.is_active) {
    hideContextMenu();
    return;
  }
  try {
    await sessionStore.activateSession(session.id);
    emit('sessionChange', session.id);
  } catch (error) {
    message.error('Failed to activate session');
  }
  hideContextMenu();
};

const handleRename = () => {
  if (!contextMenu.value.session) return;
  renameValue.value = contextMenu.value.session.name;
  renameModalVisible.value = true;
  hideContextMenu();
};

const submitRename = async () => {
  if (!contextMenu.value.session || !renameValue.value.trim()) return;
  try {
    await sessionStore.renameSession(contextMenu.value.session.id, renameValue.value.trim());
    message.success('Session renamed');
  } catch (error) {
    message.error('Failed to rename session');
  }
  renameModalVisible.value = false;
  contextMenu.value.session = null;
};

const cancelRename = () => {
  renameModalVisible.value = false;
  contextMenu.value.session = null;
};

const handleDelete = async () => {
  if (!contextMenu.value.session) return;
  const session = contextMenu.value.session;

  if (sessionStore.sessions.length === 1) {
    message.warning('Cannot delete the last session');
    hideContextMenu();
    return;
  }

  try {
    await sessionStore.deleteSession(session.id);
    agentStore.invalidateSessionCache(session.id);
    await sessionStore.fetchSessions();
    const active = sessionStore.activeSession;
    if (active) {
      emit('sessionChange', active.id);
    }
    message.success('Session deleted');
  } catch (error) {
    message.error('Failed to delete session');
  }
  hideContextMenu();
};

const handleClickOutside = () => {
  if (contextMenu.value.visible) {
    hideContextMenu();
  }
};

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && contextMenu.value.visible) {
    hideContextMenu();
  }
};

onMounted(() => {
  sessionStore.fetchSessions();
  document.addEventListener('click', handleClickOutside);
  document.addEventListener('contextmenu', handleClickOutside);
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  document.removeEventListener('contextmenu', handleClickOutside);
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<style module>
.container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  background-color: #e8e8e8;
  border-left: 1px solid #9f9f9f;
  color: #1f1f1f;
}

.horizontal {
  flex-direction: row;
  align-items: stretch;
  height: 100%;
  border-left: 0;
}

.horizontal .header {
  width: 128px;
  padding: 4px 8px;
  border-right: 1px solid #a9a9a9;
  border-bottom: 0;
}

.horizontal .list {
  display: flex;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  align-items: stretch;
}

.horizontal .sessionItem {
  width: 180px;
  min-width: 150px;
  max-width: 220px;
  border-right: 1px solid #d5d5d5;
  border-bottom: 0;
  padding: 4px 9px;
}

.horizontal .sessionItem::before {
  left: 0;
  right: 0;
  top: auto;
  bottom: 0;
  width: auto;
  height: 3px;
}

.horizontal .sessionMeta {
  display: none;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  border-bottom: 1px solid #a9a9a9;
  background: linear-gradient(180deg, #ececec 0%, #dfdfdf 100%);
  flex-shrink: 0;
}

.title {
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 0.02em;
}

.headerActions {
  display: flex;
  gap: 4px;
}

.loading {
  display: flex;
  justify-content: center;
  padding: 20px;
}

.list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.sessionItem {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  cursor: pointer;
  transition: background-color 0.15s;
  border-bottom: 1px solid #d5d5d5;
}

.sessionItem::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: transparent;
}

.sessionItem:hover {
  background-color: #dce8f5;
}

.selected {
  background-color: #c7d9ef;
}

.selected::before {
  background: #1677ff;
}

.activated {
  box-shadow: inset 0 0 0 1px rgba(82, 196, 26, 0.45);
  background-image: linear-gradient(180deg, rgba(246, 255, 237, 0.88) 0%, rgba(240, 255, 228, 0.72) 100%);
}

.selected.activated {
  background-image: linear-gradient(180deg, rgba(230, 244, 255, 0.98) 0%, rgba(214, 240, 255, 0.88) 60%, rgba(241, 255, 230, 0.92) 100%);
  box-shadow: inset 0 0 0 1px rgba(34, 197, 94, 0.35);
}

.sessionIcon {
  position: relative;
  font-size: 14px;
  color: #7cb9e6;
  flex-shrink: 0;
}

.selected .sessionIcon {
  color: #3b88d6;
}

.recordingDot {
  position: absolute;
  right: -3px;
  bottom: -1px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #52c41a;
  border: 1px solid #ffffff;
  box-shadow: 0 0 0 1px rgba(47, 125, 12, 0.22);
}

.sessionInfo {
  flex: 1;
  min-width: 0;
}

.sessionNameRow {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.sessionName {
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.selected .sessionName {
  color: #0f3d75;
}

.viewBadge {
  flex-shrink: 0;
  font-size: 9px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 2px;
  background: #1677ff;
  color: #fff;
  line-height: 1.4;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.activeBadge {
  flex-shrink: 0;
  font-size: 9px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 2px;
  background: #389e0d;
  color: #fff;
  line-height: 1.4;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.sessionMeta {
  font-size: 10px;
  color: #777;
  margin-top: 2px;
}

.empty {
  padding: 20px;
  text-align: center;
  color: #888;
  font-size: 12px;
}
</style>

<style>
/* 全局样式 - 用于 Teleport 的右键菜单 */
.session-context-menu {
  position: fixed;
  z-index: 10000;
  background-color: #f2f2f2;
  border-radius: 0;
  box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);
  border: 1px solid #a0a0a0;
  padding: 2px 0;
  min-width: 140px;
}

.session-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 16px;
  cursor: pointer;
  font-size: 12px;
  color: #1f1f1f;
}

.session-menu-item:hover {
  background-color: #0066cc;
  color: #ffffff;
}

.session-menu-item:hover .anticon {
  color: #ffffff;
}

.session-menu-item.session-menu-item-disabled {
  color: #aaa;
  cursor: default;
  pointer-events: none;
}

.session-menu-item.session-menu-item-danger {
  color: #1f1f1f;
}

.session-menu-item.session-menu-item-danger:hover {
  background-color: #d13438;
  color: #ffffff;
}

.session-menu-divider {
  height: 1px;
  background-color: #c0c0c0;
  margin: 2px 0;
}
</style>
