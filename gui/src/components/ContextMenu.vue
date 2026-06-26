<template>
  <Teleport to="body">
    <div
      v-if="visible"
      :class="$style.contextMenu"
      :style="menuStyle"
      @click.stop
      @contextmenu.stop.prevent
    >
      <div :class="$style.menuItem" @click.stop="handleAnalyze">
        <span :class="$style.icon">🤖</span>
        <span>Agent Analyze</span>
      </div>
      <div :class="$style.menuItem" @click.stop="handleCopy">
        <span :class="$style.icon">📋</span>
        <span>Copy Value</span>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { copyToClipboard, getWindowSize } from '../utils/wails';

interface ContextMenuProps {
  visible: boolean;
  position: { x: number; y: number };
  fieldName: string;
  fieldValue: string;
  location: string;
  requestId: string;
}

const props = defineProps<ContextMenuProps>();
const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'analyze', data: { fieldName: string; fieldValue: string; location: string; requestId: string }): void;
}>();

const windowDimensions = ref({ width: 1024, height: 768 });

// Update window size when menu becomes visible
watch(() => props.visible, async (isVisible) => {
  if (isVisible) {
    windowDimensions.value = await getWindowSize();
  }
});

const menuStyle = computed(() => {
  // Adjust position to keep menu within viewport
  let x = props.position.x;
  let y = props.position.y;
  
  // Keep menu within viewport bounds
  const menuWidth = 160;
  const menuHeight = 80;
  
  if (x + menuWidth > windowDimensions.value.width) {
    x = windowDimensions.value.width - menuWidth - 10;
  }
  if (y + menuHeight > windowDimensions.value.height) {
    y = windowDimensions.value.height - menuHeight - 10;
  }
  
  return {
    left: `${x}px`,
    top: `${y}px`,
  };
});

const handleAnalyze = () => {
  emit('analyze', {
    fieldName: props.fieldName,
    fieldValue: props.fieldValue,
    location: props.location,
    requestId: props.requestId,
  });
  emit('close');
};

const handleCopy = async () => {
  await copyToClipboard(props.fieldValue);
  emit('close');
};

const handleClickOutside = (e: MouseEvent) => {
  if (!props.visible) return;
  emit('close');
};

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible) {
    emit('close');
  }
};

onMounted(() => {
  // Use setTimeout to prevent immediate close on the same event loop
  setTimeout(() => {
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('contextmenu', handleClickOutside);
    document.addEventListener('keydown', handleKeydown);
  }, 0);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  document.removeEventListener('contextmenu', handleClickOutside);
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<style module>
.contextMenu {
  position: fixed;
  z-index: 10000;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 6px 16px 0 rgba(0, 0, 0, 0.08), 0 3px 6px -4px rgba(0, 0, 0, 0.12), 0 9px 28px 8px rgba(0, 0, 0, 0.05);
  border: 1px solid #f0f0f0;
  padding: 4px 0;
  min-width: 150px;
}

.menuItem {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 14px;
  user-select: none;
}

.menuItem:hover {
  background-color: #e6f7ff;
}

.icon {
  font-size: 14px;
}
</style>
