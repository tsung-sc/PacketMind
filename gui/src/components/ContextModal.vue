<template>
  <a-modal
    :open="visible"
    title="上下文"
    :width="modalSize.width"
    :footer="null"
    @cancel="handleClose"
    :body-style="modalBodyStyle"
  >
    <template #modalRender="{ originVNode }">
      <div :style="transformStyle">
        <component :is="originVNode" />
      </div>
    </template>
    <template #title>
      <div ref="modalTitleRef" :class="$style.dragTitle" @mousedown="startDrag">上下文</div>
    </template>
    <div :class="$style.container">
      <div :class="$style.summaryGrid">
        <div :class="$style.summaryCard">
          <div :class="$style.cardTitle">会话</div>
          <div :class="$style.itemGrid">
            <div :class="$style.item"><span>会话</span><span>{{ stats?.session_id || '-' }}</span></div>
            <div :class="$style.item"><span>消息数</span><span>{{ stats?.message_count ?? totalMessages }}</span></div>
            <div :class="$style.item"><span>持久化历史</span><span>{{ stats ? (stats.has_history ? '是' : '否') : '-' }}</span></div>
            <div :class="$style.item"><span>上下文限制</span><span>{{ selectedModel?.max_tokens || stats?.active_max_tokens || '-' }}</span></div>
            <div :class="$style.item"><span>使用率</span><span>{{ usagePercent }}%</span></div>
            <div :class="$style.item"><span>最后活动</span><span>{{ lastTimestamp }}</span></div>
          </div>
        </div>

        <div :class="$style.summaryCard">
          <div :class="$style.cardTitle">统计</div>
          <div :class="$style.itemGrid">
            <div :class="$style.item"><span>模型</span><span>{{ selectedModel?.name || stats?.active_model || '-' }}</span></div>
            <div :class="$style.item"><span>总 token</span><span>{{ totalTokens }}</span></div>
            <div :class="$style.item"><span>输入 token</span><span>{{ inputTokens }}</span></div>
            <div :class="$style.item"><span>输出 token</span><span>{{ outputTokens }}</span></div>
            <div :class="$style.item"><span>推理 token</span><span>{{ reasoningTokens }}</span></div>
            <div :class="$style.item"><span>缓存 (读/写)</span><span>{{ `${cacheReadTokens}/${cacheWriteTokens}` }}</span></div>
          </div>
        </div>
      </div>

      <div :class="$style.breakdownPanel">
        <div :class="$style.cardTitle">上下文拆分</div>
        <div :class="$style.progressTrack">
          <div :class="$style.userSegment" :style="{ width: `${breakdown.user}%` }"></div>
          <div :class="$style.assistantSegment" :style="{ width: `${breakdown.assistant}%` }"></div>
          <div :class="$style.toolSegment" :style="{ width: `${breakdown.tools}%` }"></div>
          <div :class="$style.otherSegment" :style="{ width: `${breakdown.other}%` }"></div>
        </div>
        <div :class="$style.legend">
          <span><i :class="$style.userDot"></i>用户 {{ breakdown.user.toFixed(1) }}%</span>
          <span><i :class="$style.assistantDot"></i>助手 {{ breakdown.assistant.toFixed(1) }}%</span>
          <span><i :class="$style.toolDot"></i>工具调用 {{ breakdown.tools.toFixed(1) }}%</span>
          <span><i :class="$style.otherDot"></i>其他 {{ breakdown.other.toFixed(1) }}%</span>
        </div>
      </div>

      <div :class="$style.viewerShell">
        <div :class="$style.listPanel">
          <div :class="$style.panelHeader">
            <span>原始消息</span>
            <span :class="$style.panelMeta">{{ totalMessages }} 条</span>
          </div>

          <div :class="$style.messageList">
            <button
              v-for="msg in messages"
              :key="msg.id"
              type="button"
              :class="[$style.messageRow, { [$style.selected]: selectedMessage?.id === msg.id }]"
              @click="selectMessage(msg.id)"
            >
              <div :class="$style.messageRowTop">
                <span :class="$style.roleTag">{{ messageHeaderLabel(msg) }}</span>
                <span :class="$style.messageMeta">{{ msg.id }}</span>
              </div>
              <div :class="$style.messagePreview">{{ messagePreview(msg) }}</div>
              <div :class="$style.messageTime">{{ formatTimestamp(msg.timestamp) }}</div>
            </button>
          </div>
        </div>

        <div :class="$style.rawPanel">
          <div :class="$style.panelHeader">
            <div :class="$style.rawHeaderMain">
              <span>原始结构</span>
              <span v-if="selectedMessage" :class="$style.rawHeaderMeta">{{ messageHeaderLabel(selectedMessage) }} &bull; {{ selectedMessage.id }}</span>
            </div>
            <span v-if="selectedMessage" :class="$style.rawHeaderTime">{{ formatTimestamp(selectedMessage.timestamp) }}</span>
          </div>

          <div v-if="selectedMessage" :class="$style.rawViewer">
            <div v-for="(line, index) in selectedMessageLines" :key="`${selectedMessage.id}-${index}`" :class="$style.rawLine">
              <span :class="$style.lineNumber">{{ index + 1 }}</span>
              <span :class="$style.lineCode">{{ line || ' ' }}</span>
            </div>
          </div>

          <div v-else :class="$style.emptyState">
            <div :class="$style.emptyTitle">选择左侧消息</div>
            <div :class="$style.emptyText">右侧会显示该消息的完整原始结构。</div>
          </div>
        </div>
      </div>

      <button
        type="button"
        :class="[$style.resizeHandle, { [$style.resizing]: isResizing }]"
        title="拖拽调整窗口尺寸"
        @mousedown="startResize"
      >
        <span :class="$style.resizeGrip"></span>
      </button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, type CSSProperties, watch, watchEffect } from 'vue'
import { useAgentStore, type Message } from '@/stores/agentStore'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'update:visible', value: boolean): void }>()

const agentStore = useAgentStore()

const selectedMessageID = ref<string | null>(null)
const isResizing = ref(false)
const modalSize = ref({ width: 920, height: 560 })

// --- Drag-to-move state ---
const modalTitleRef = ref<HTMLDivElement | null>(null)
const dragOffsetX = ref(0)
const dragOffsetY = ref(0)
const startedDrag = ref(false)
const preTransformX = ref(0)
const preTransformY = ref(0)

type DragSession = {
  startX: number
  startY: number
}

let dragSession: DragSession | null = null

const transformStyle = computed<CSSProperties>(() => ({
  transform: `translate(${dragOffsetX.value}px, ${dragOffsetY.value}px)`,
}))

const stopDrag = () => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('mousemove', handleDrag)
    window.removeEventListener('mouseup', stopDrag)
  }
  preTransformX.value = dragOffsetX.value
  preTransformY.value = dragOffsetY.value
  dragSession = null
  startedDrag.value = false
  document.body.style.userSelect = ''
  document.body.style.cursor = ''
}

const handleDrag = (event: MouseEvent) => {
  if (!dragSession) return
  const deltaX = event.clientX - dragSession.startX
  const deltaY = event.clientY - dragSession.startY
  const nextX = preTransformX.value + deltaX
  const nextY = preTransformY.value + deltaY
  const maxX = window.innerWidth - 60
  const maxY = window.innerHeight - 40
  dragOffsetX.value = clamp(nextX, -maxX, maxX)
  dragOffsetY.value = clamp(nextY, -maxY, maxY)
}

const startDrag = (event: MouseEvent) => {
  event.preventDefault()
  dragSession = {
    startX: event.clientX,
    startY: event.clientY,
  }
  startedDrag.value = true
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'move'
  window.addEventListener('mousemove', handleDrag)
  window.addEventListener('mouseup', stopDrag)
}

const resetDragOffset = () => {
  dragOffsetX.value = 0
  dragOffsetY.value = 0
  preTransformX.value = 0
  preTransformY.value = 0
}
// --- End drag-to-move state ---

const stats = computed(() => agentStore.sessionContext)
const messages = computed(() => agentStore.messages)
const selectedModel = computed(() => agentStore.getSelectedModel())

const DEFAULT_MODAL_WIDTH = 920
const DEFAULT_MODAL_HEIGHT = 560
const MIN_MODAL_WIDTH = 760
const MIN_MODAL_HEIGHT = 420
const VIEWPORT_MARGIN_X = 80
const VIEWPORT_MARGIN_Y = 140

type ResizeSession = {
  startX: number
  startY: number
  startWidth: number
  startHeight: number
}

let resizeSession: ResizeSession | null = null

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

const getViewportConstraints = () => {
  if (typeof window === 'undefined') {
    return {
      maxWidth: DEFAULT_MODAL_WIDTH,
      maxHeight: DEFAULT_MODAL_HEIGHT,
    }
  }

  return {
    maxWidth: Math.max(MIN_MODAL_WIDTH, window.innerWidth - VIEWPORT_MARGIN_X),
    maxHeight: Math.max(MIN_MODAL_HEIGHT, window.innerHeight - VIEWPORT_MARGIN_Y),
  }
}

const applyModalSize = (width: number, height: number) => {
  const { maxWidth, maxHeight } = getViewportConstraints()

  modalSize.value = {
    width: clamp(Math.round(width), MIN_MODAL_WIDTH, maxWidth),
    height: clamp(Math.round(height), MIN_MODAL_HEIGHT, maxHeight),
  }
}

const modalBodyStyle = computed<CSSProperties>(() => ({
  height: `${modalSize.value.height}px`,
  overflow: 'hidden',
  padding: '12px 14px 14px',
  display: 'flex',
  flexDirection: 'column',
}))

const stopResize = () => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('mousemove', handleResize)
    window.removeEventListener('mouseup', stopResize)
  }

  document.body.style.userSelect = ''
  document.body.style.cursor = ''
  resizeSession = null
  isResizing.value = false
}

const handleResize = (event: MouseEvent) => {
  if (!resizeSession) {
    return
  }

  const nextWidth = resizeSession.startWidth + (event.clientX - resizeSession.startX)
  const nextHeight = resizeSession.startHeight + (event.clientY - resizeSession.startY)
  applyModalSize(nextWidth, nextHeight)
}

const startResize = (event: MouseEvent) => {
  event.preventDefault()
  event.stopPropagation()

  resizeSession = {
    startX: event.clientX,
    startY: event.clientY,
    startWidth: modalSize.value.width,
    startHeight: modalSize.value.height,
  }

  isResizing.value = true
  document.body.style.userSelect = 'none'
  document.body.style.cursor = 'nwse-resize'
  window.addEventListener('mousemove', handleResize)
  window.addEventListener('mouseup', stopResize)
}

onBeforeUnmount(() => {
	stopResize()
	stopDrag()
})

watch(() => props.visible, (visible) => {
	if (!visible) {
		return
	}

	resetDragOffset()
	applyModalSize(modalSize.value.width, modalSize.value.height)
	agentStore.fetchSessionContext()
})

watchEffect(() => {
	if (!props.visible) {
		return
	}

	const currentMessages = messages.value
	if (!currentMessages.length) {
		selectedMessageID.value = null
		return
	}

	if (!selectedMessageID.value || !currentMessages.some((msg) => msg.id === selectedMessageID.value)) {
		selectedMessageID.value = currentMessages[currentMessages.length - 1].id
	}
})

const selectMessage = (id: string) => {
  selectedMessageID.value = id
}

const formatJSON = (val: unknown): string => {
  if (!val) return ''
  if (typeof val === 'string') {
    try {
      return JSON.stringify(JSON.parse(val), null, 2)
    } catch {
      return val
    }
  }
  return JSON.stringify(val, null, 2)
}

const parseJSONSafely = (val: string) => {
  try {
    return JSON.parse(val)
  } catch {
    return val
  }
}

const buildRawMessagePayload = (msg: Message) => {
  const parts: Array<Record<string, unknown>> = []

  if (msg.content) {
    parts.push({
      type: 'text',
      text: msg.content,
      messageID: msg.id,
    })
  }

  if (msg.toolName) {
    parts.push({
      type: 'tool_call',
      name: msg.toolName,
      arguments: msg.arguments ? parseJSONSafely(msg.arguments) : undefined,
    })
  }

  if (msg.errorCategory || msg.errorToolName) {
    parts.push({
      type: 'error',
      category: msg.errorCategory || null,
      toolName: msg.errorToolName || null,
      recovered: msg.errorRecovered ?? null,
      timeout: msg.errorTimeout || null,
    })
  }

  if (msg.requestIds?.length) {
    parts.push({
      type: 'related_requests',
      requestIDs: msg.requestIds,
    })
  }

  if (msg.provenance) {
    parts.push({
      type: 'provenance',
      value: msg.provenance,
    })
  }

  return {
    message: {
      id: msg.id,
      sessionID: stats.value?.session_id || null,
      role: messageHeaderLabel(msg),
      time: {
        created: msg.timestamp,
        display: formatTimestamp(msg.timestamp),
      },
    },
    agent: {
      specialist: msg.specialistName || null,
      depth: msg.depth ?? null,
    },
    model: {
      modelID: selectedModel.value?.id || stats.value?.active_model || null,
      modelName: selectedModel.value?.name || null,
    },
    summary: {
      hasContent: Boolean(msg.content),
      hasToolCall: Boolean(msg.toolName),
      hasError: Boolean(msg.errorCategory || msg.errorToolName),
      requestCount: msg.requestIds?.length || 0,
    },
    parts,
  }
}

const getRawMessageLines = (msg: Message) => formatJSON(buildRawMessagePayload(msg)).split('\n')

const selectedMessage = computed(() => {
  if (!messages.value.length) {
    return null
  }

  return messages.value.find((msg) => msg.id === selectedMessageID.value) ?? messages.value[messages.value.length - 1]
})

const selectedMessageLines = computed(() => selectedMessage.value ? getRawMessageLines(selectedMessage.value) : [])

const totalMessages = computed(() => messages.value.length)
const assistantMessages = computed(() => messages.value.filter((msg) => msg.role === 'assistant').length)
const userMessages = computed(() => messages.value.filter((msg) => msg.role === 'user').length)
const toolMessages = computed(() => messages.value.filter((msg) => ['agent_action', 'agent_observation', 'agent_error', 'agent_decision'].includes(msg.role)).length)

const inputTokens = computed(() => agentStore.tokenUsage?.input_tokens || 0)
const outputTokens = computed(() => agentStore.tokenUsage?.output_tokens || 0)
const cacheWriteTokens = computed(() => agentStore.tokenUsage?.cache_creation_input_tokens || 0)
const cacheReadTokens = computed(() => agentStore.tokenUsage?.cache_read_input_tokens || 0)
const totalTokens = computed(() => agentStore.tokensUsed || stats.value?.estimated_tokens || 0)
const reasoningTokens = computed(() => Math.max(totalTokens.value - inputTokens.value - outputTokens.value - cacheWriteTokens.value - cacheReadTokens.value, 0))
const usagePercent = computed(() => {
  const limit = selectedModel.value?.max_tokens || stats.value?.active_max_tokens || 0
  if (!limit) return 0
  return Math.min(100, Math.round((totalTokens.value / limit) * 100))
})

const breakdown = computed(() => {
  const total = totalMessages.value || 1
  const user = (userMessages.value / total) * 100
  const assistant = (assistantMessages.value / total) * 100
  const tools = (toolMessages.value / total) * 100
  const other = Math.max(0, 100 - user - assistant - tools)
  return { user, assistant, tools, other }
})

const firstTimestamp = computed(() => messages.value.length > 0 ? formatTimestamp(messages.value[0].timestamp) : '-')
const lastTimestamp = computed(() => messages.value.length > 0 ? formatTimestamp(messages.value[messages.value.length - 1].timestamp) : '-')

const formatTimestamp = (timestamp: number) => {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN')
}

const messagePreview = (msg: Message) => {
  if (msg.content?.trim()) {
    return msg.content.replace(/\s+/g, ' ').slice(0, 120)
  }

  if (msg.toolName) {
    return `tool_call · ${msg.toolName}`
  }

  if (msg.errorToolName || msg.errorCategory) {
    return `error · ${msg.errorToolName || msg.errorCategory}`
  }

  if (msg.requestIds?.length) {
    return `related_requests · ${msg.requestIds.length} 条`
  }

  if (msg.provenance) {
    return 'provenance'
  }

  return '无附加内容'
}

const messageLabel = (role: Message['role']) => {
  switch (role) {
    case 'user': return 'user'
    case 'assistant': return 'assistant'
    case 'agent_action': return 'tool'
    case 'agent_observation':
    case 'agent_error':
    case 'agent_decision':
    case 'agent_related':
    case 'agent_provenance':
    case 'agent_thought':
      return 'assistant'
    default:
      return role
  }
}

const messageHeaderLabel = (msg: Message) => {
  switch (msg.role) {
    case 'agent_action':
      return 'tool'
    case 'agent_observation':
      return 'observation'
    case 'agent_error':
      return 'error'
    case 'agent_decision':
      return 'decision'
    case 'agent_thought':
      return 'assistant'
    case 'agent_related':
      return 'related'
    case 'agent_provenance':
      return 'provenance'
    default:
      return msg.role
  }
}

const handleClose = () => {
  selectedMessageID.value = null
  resetDragOffset()
  emit('update:visible', false)
}
</script>

<style module>
.dragTitle {
  cursor: move;
  user-select: none;
}
.container {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
  flex: 1;
  min-height: 0;
  padding-bottom: 10px;
}
.summaryGrid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.summaryCard, .breakdownPanel, .listPanel, .rawPanel {
  border: 1px solid #e5e5e5;
  border-radius: 2px;
  background: #fff;
}
.summaryCard {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px 12px;
  min-width: 0;
}
.cardTitle {
  font-size: 12px;
  font-weight: 700;
  color: #444;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.itemGrid {
  display: grid;
  gap: 7px;
}
.item {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  gap: 10px;
  font-size: 12px;
  color: #333;
  align-items: center;
}
.item span:first-child { color: #777; }
.item span:last-child {
  min-width: 0;
  color: #111;
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}
.breakdownPanel {
  display: flex;
  flex-direction: column;
  gap: 9px;
  padding: 10px 12px;
}
.progressTrack { display: flex; width: 100%; height: 8px; border-radius: 2px; overflow: hidden; background: #f0f0f0; }
.userSegment { background: #3c9a3c; }
.assistantSegment { background: #bf5af2; }
.toolSegment { background: #8c6a1f; }
.otherSegment { background: #8c8c8c; }
.legend { display: flex; flex-wrap: wrap; gap: 12px; font-size: 12px; color: #666; }
.userDot,.assistantDot,.toolDot,.otherDot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 4px; }
.userDot { background: #3c9a3c; }
.assistantDot { background: #bf5af2; }
.toolDot { background: #8c6a1f; }
.otherDot { background: #8c8c8c; }
.viewerShell {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 12px;
  flex: 1;
  min-height: 0;
}
.listPanel, .rawPanel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
.panelHeader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-bottom: 1px solid #ececec;
  background: linear-gradient(180deg, #fafafa 0%, #f3f3f3 100%);
  font-size: 12px;
  font-weight: 600;
  color: #333;
}
.panelMeta { font-size: 11px; color: #777; font-weight: 500; }
.messageList { flex: 1; overflow-y: auto; min-height: 0; background: #fff; }
.messageRow {
  all: unset;
  box-sizing: border-box;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
}
.messageRow:hover { background: #fafafa; }
.messageRow:last-child { border-bottom: none; }
.selected {
  background: #f4f6fa;
  box-shadow: inset 2px 0 0 #1f1f1f;
}
.messageRowTop {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.roleTag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 6px;
  border: 1px solid #d9d9d9;
  border-radius: 2px;
  background: #fafafa;
  font-size: 10px;
  line-height: 1.4;
  text-transform: uppercase;
  color: #555;
  flex-shrink: 0;
}
.messageMeta { min-width: 0; color: #333; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.messagePreview {
  color: #777;
  font-size: 12px;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}
.messageTime { color: #999; white-space: nowrap; font-size: 11px; }
.rawHeaderMain {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.rawHeaderMeta {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  color: #666;
  font-weight: 500;
}
.rawHeaderTime { flex-shrink: 0; color: #999; font-size: 11px; font-weight: 500; }
.rawViewer { flex: 1; min-height: 0; overflow: auto; background: #fbfbfc; }
.rawLine { display: grid; grid-template-columns: 52px minmax(0, 1fr); align-items: start; min-height: 24px; }
.lineNumber { user-select: none; text-align: right; padding: 0 12px 0 0; color: #a6a6a6; font-size: 12px; line-height: 24px; border-right: 1px solid #f0f0f0; background: #fafafa; }
.lineCode { display: block; padding: 0 14px; color: #1f1f1f; font-size: 12px; line-height: 24px; white-space: pre; font-family: 'Consolas', 'Monaco', 'Courier New', monospace; }
.emptyState {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex: 1;
  min-height: 0;
  background: #fafafa;
  color: #777;
  text-align: center;
}
.emptyTitle { font-size: 13px; font-weight: 600; color: #444; }
.emptyText { font-size: 12px; }
.resizeHandle {
  all: unset;
  position: absolute;
  right: 0;
  bottom: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
  cursor: nwse-resize;
  z-index: 5;
}
.resizeGrip {
  width: 12px;
  height: 12px;
  background-image:
    linear-gradient(135deg, transparent 0 44%, #a8a8a8 44% 52%, transparent 52% 100%),
    linear-gradient(135deg, transparent 0 62%, #bdbdbd 62% 70%, transparent 70% 100%),
    linear-gradient(135deg, transparent 0 80%, #d0d0d0 80% 88%, transparent 88% 100%);
  opacity: 0.8;
}
.resizeHandle:hover .resizeGrip,
.resizing .resizeGrip {
  opacity: 1;
}
</style>
