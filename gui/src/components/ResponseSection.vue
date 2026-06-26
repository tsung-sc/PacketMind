<template>
  <div :class="$style.container">
    <div :class="$style.summaryBar">
      <span :class="$style.summaryText">{{ statusLine }}</span>
    </div>

    <div :class="$style.contentArea">
      <div :class="$style.viewerArea">
        <HeadersView
          v-if="activeView === 'headers'"
          :headers="headers"
          :requestId="requestId"
          :location="'Response Headers'"
          :leadLine="statusLine"
          :isNarrow="isNarrow"
        />

        <div v-else-if="activeView === 'text'" :class="$style.viewerPad">
          <pre
            :class="$style.bodyContent"
            @contextmenu.stop.prevent="openContextMenu($event, 'Response Body', formattedBody, 'Response Body Text')"
          >{{ formattedBody }}</pre>
        </div>

        <div v-else-if="activeView === 'json_text'" :class="$style.viewerPad">
          <pre
            :class="[$style.bodyContent, $style.jsonContent]"
            @contextmenu.stop.prevent="openContextMenu($event, 'Response JSON', formattedJSON, 'Response JSON Text')"
          ><span v-html="highlightedJSON"></span></pre>
        </div>

        <div v-else-if="activeView === 'html'" :class="[$style.viewerPad, $style.iframePad]">
          <iframe
            v-if="htmlContent"
            :srcdoc="htmlContent"
            sandbox=""
            :class="$style.previewIframe"
          ></iframe>
          <a-empty v-else description="HTML preview unavailable" />
        </div>

        <div v-else-if="activeView === 'tree'" :class="[$style.viewerPad, $style.treePad]">
          <div v-if="bodyTree" :class="$style.treeRoot">
            <BodyTreeNodeView :node="bodyTree.root" :depth="0" />
          </div>
          <a-empty v-else description="Body tree unavailable" />
        </div>

        <div v-else-if="activeView === 'image'" :class="[$style.viewerPad, $style.imagePad]">
          <div v-if="imageSrc" :class="$style.imageWrap">
            <img :src="imageSrc" alt="Response preview" :class="$style.previewImage" />
          </div>
          <a-empty v-else description="Image preview unavailable" />
        </div>

        <div v-else-if="activeView === 'hex'" :class="$style.viewerPad">
          <pre
            :class="[$style.bodyContent, $style.hexContent]"
            @contextmenu.stop.prevent="openContextMenu($event, 'Response Body Hex', hexBody, 'Response Body Hex')"
          >{{ hexBody }}</pre>
        </div>

        <div v-else-if="activeView === 'websocket'" :class="[$style.viewerPad, $style.websocketPad]">
          <table :class="$style.websocketTable">
            <thead>
              <tr>
                <th>Type</th>
                <th>Start</th>
                <th>End</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!responseWebSocketFrames.length">
                <td colspan="4" :class="$style.websocketEmpty">No response-side WebSocket messages</td>
              </tr>
              <tr v-for="frame in responseWebSocketFrames" :key="frame.id">
                <td>{{ websocketTypeLabel(frame) }}</td>
                <td>{{ formatFrameTime(frame.created_at) }}</td>
                <td>{{ formatFrameTime(frame.created_at) }}</td>
                <td :class="$style.websocketMessage">{{ frame.payload || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else :class="$style.viewerPad">
          <pre
            :class="$style.bodyContent"
            @contextmenu.stop.prevent="openContextMenu($event, 'Response Raw', rawResponse, 'Response Raw')"
          >{{ rawResponse }}</pre>
        </div>
      </div>

      <div :class="$style.tabBar">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          :class="[$style.tab, { [$style.tabActive]: activeView === tab.key }]"
          @click="activeView = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <ContextMenu
      :visible="contextMenu.visible"
      :position="contextMenu.position"
      :fieldName="contextMenu.fieldName"
      :fieldValue="contextMenu.fieldValue"
      :location="contextMenu.location"
      :requestId="requestId"
      @close="closeContextMenu"
      @analyze="handleAnalyze"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import BodyTreeNodeView from './BodyTreeNodeView.vue';
import HeadersView from './HeadersView.vue';
import ContextMenu from './ContextMenu.vue';
import { useAgentStore } from '@/stores/agentStore';
import { EMPTY_BODY_PLACEHOLDER, buildImageDataURLAsync, decodeBodyAsync, formatBodyForTextAsync, isImageContentType, isJSONContentType, isHTMLContentType, parseStructuredBodyAsync, StructuredBodyTree } from './bodyTreeUtils';

interface Props {
  requestId: string;
  statusCode: number;
  statusReason: string;
  httpVersion: string;
  headers: Record<string, string[]> | null;
  body: string | null;
  contentType: string | null;
  websocketFrames?: Array<{
    id: string;
    direction: string;
    opcode: number;
    frame_type: string;
    payload: string;
    payload_size: number;
    created_at: string;
    fin: boolean;
    masked: boolean;
  }> | null;
  isNarrow?: boolean;
}

const props = defineProps<Props>();
const agentStore = useAgentStore();

const baseTabs = [
  { key: 'headers', label: 'Headers' },
] as const;

type ViewKey = (typeof baseTabs)[number]['key'] | 'tree' | 'image' | 'json_text' | 'html';
type ExtendedViewKey = ViewKey | 'text' | 'hex' | 'raw' | 'websocket';
const activeView = ref<ExtendedViewKey>('headers');

const statusLine = computed(() => `${props.httpVersion || 'HTTP/1.1'} ${props.statusCode} ${props.statusReason || ''}`.trim());

const formattedBody = ref<string>(EMPTY_BODY_PLACEHOLDER);
const formattedJSON = ref<string>(EMPTY_BODY_PLACEHOLDER);
const htmlContent = ref<string | null>(null);
const bodyTree = ref<StructuredBodyTree | null>(null);
const hasImagePreview = ref(false);
const imageSrc = ref<string | null>(null);
const hasJSONText = ref(false);
const hasHTMLPreview = ref(false);
const hasBody = computed(() => !!props.body && props.body.trim().length > 0);
const responseWebSocketFrames = computed(() => (props.websocketFrames || []).filter((frame) => frame.direction === 'received'));
const hasWebSocketFrames = computed(() => responseWebSocketFrames.value.length > 0);

const escapeHtml = (s: string): string =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

const highlightedJSON = computed(() => {
  const raw = formattedJSON.value;
  if (!raw || raw === EMPTY_BODY_PLACEHOLDER) return escapeHtml(raw);
  
  const escaped = escapeHtml(raw);
  const highlighted = escaped.replace(
    /("(?:\\.|[^"\\])*")\s*:/g,
    '<span class="json-key">$1</span>:'
  ).replace(
    /:\s*("(?:\\.|[^"\\])*")/g,
    ': <span class="json-str">$1</span>'
  ).replace(
    /:\s*(-?\d+\.?\d*(?:[eE][+-]?\d+)?)/g,
    ': <span class="json-num">$1</span>'
  ).replace(
    /:\s*(true|false)\b/g,
    ': <span class="json-bool">$1</span>'
  ).replace(
    /:\s*(null)\b/g,
    ': <span class="json-null">$1</span>'
  );

  return highlighted.split('\n').map(line => `<div class="json-line">${line || ' '}</div>`).join('');
});

const getContentEncoding = (headers: Record<string, string[]> | null): string | null => {
  if (!headers) return null;
  const key = Object.keys(headers).find(k => k.toLowerCase() === 'content-encoding');
  if (!key) return null;
  const vals = headers[key];
  return vals && vals.length > 0 ? vals[0] : null;
};

watch(
  () => [props.body, props.contentType, props.headers],
  async () => {
    const ce = getContentEncoding(props.headers);
    const [rawDecoded, fmtJSON, tree, img] = await Promise.all([
      decodeBodyAsync(props.body, props.contentType, ce),
      formatBodyForTextAsync(props.body, props.contentType, ce),
      parseStructuredBodyAsync(props.body, props.contentType, ce),
      buildImageDataURLAsync(props.body, props.contentType, ce)
    ]);
    formattedBody.value = rawDecoded;
    formattedJSON.value = fmtJSON;
    htmlContent.value = rawDecoded === EMPTY_BODY_PLACEHOLDER ? null : rawDecoded;
    bodyTree.value = tree;
    imageSrc.value = img;
    hasImagePreview.value = isImageContentType(props.contentType) && !!props.body;
    hasJSONText.value = isJSONContentType(props.contentType) && !!props.body;
    hasHTMLPreview.value = isHTMLContentType(props.contentType) && !!props.body;
  },
  { immediate: true, deep: true }
);

const tabs = computed(() => {
  const list: Array<{ key: ViewKey; label: string }> = [...baseTabs];
  if (!hasBody.value) {
    if (hasWebSocketFrames.value) {
      list.push({ key: 'websocket', label: 'WebSocket' });
    }
    list.push({ key: 'raw', label: 'Raw' });
    return list;
  }
  list.push({ key: 'text', label: 'Text' });
  if (hasJSONText.value) {
    list.push({ key: 'json_text', label: 'JSON Text' });
  }
  if (bodyTree.value) {
    list.push({ key: 'tree', label: 'Tree' });
  }
  if (hasHTMLPreview.value) {
    list.push({ key: 'html', label: 'HTML' });
  }
  if (hasImagePreview.value) {
    list.push({ key: 'image', label: 'Image' });
  }
  list.push({ key: 'hex', label: 'Hex' });
  if (hasWebSocketFrames.value) {
    list.push({ key: 'websocket', label: 'WebSocket' });
  }
  list.push({ key: 'raw', label: 'Raw' });
  return list;
});

watch([bodyTree, hasImagePreview, hasJSONText, hasHTMLPreview, tabs], () => {
  const currentTab = activeView.value;
  const allowed = new Set(tabs.value.map((tab) => tab.key));
  const fallback = hasBody.value ? 'text' : 'headers';
  if (!bodyTree.value && currentTab === 'tree') activeView.value = fallback;
  if (!hasImagePreview.value && currentTab === 'image') activeView.value = fallback;
  if (!hasJSONText.value && currentTab === 'json_text') activeView.value = fallback;
  if (!hasHTMLPreview.value && currentTab === 'html') activeView.value = fallback;
  if (!allowed.has(activeView.value)) activeView.value = fallback;
});

const toHex = (input: string) => {
  if (!input || input === EMPTY_BODY_PLACEHOLDER) return EMPTY_BODY_PLACEHOLDER;
  return Array.from(input)
    .map((char, index) => {
      const hex = char.charCodeAt(0).toString(16).padStart(2, '0');
      return `${hex}${(index + 1) % 16 === 0 ? '\n' : ' '}`;
    })
    .join('')
    .trim();
};

const hexBody = computed(() => toHex(formattedBody.value));

const rawResponse = computed(() => {
  const headerLines = Object.entries(props.headers || {})
    .flatMap(([name, values]) => (values || []).map((value) => `${name}: ${value}`));
  return [statusLine.value, ...headerLines, '', formattedBody.value].join('\n').trim();
});

const websocketTypeLabel = (frame: NonNullable<Props['websocketFrames']>[number]) => {
  return frame.frame_type === 'text' ? 'Text' : frame.frame_type || 'Frame';
};

const formatFrameTime = (value: string) => {
  if (!value) return '-';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return '-';
  return parsed.toLocaleTimeString('zh-CN', { hour12: false });
};

interface ContextMenuState {
  visible: boolean;
  position: { x: number; y: number };
  fieldName: string;
  fieldValue: string;
  location: string;
}

const contextMenu = ref<ContextMenuState>({
  visible: false,
  position: { x: 0, y: 0 },
  fieldName: '',
  fieldValue: '',
  location: 'Response',
});

const openContextMenu = (event: MouseEvent, fieldName: string, fieldValue: string, location: string) => {
  contextMenu.value = {
    visible: true,
    position: { x: event.clientX, y: event.clientY },
    fieldName,
    fieldValue: fieldValue.substring(0, 5000),
    location,
  };
};

const closeContextMenu = () => {
  contextMenu.value.visible = false;
};

const handleAnalyze = (data: { fieldName: string; fieldValue: string; location: string; requestId: string }) => {
  agentStore.analyzeParameter({
    requestId: data.requestId,
    fieldName: data.fieldName,
    fieldValue: data.fieldValue,
    location: data.location,
  });
};
</script>

<style module>
.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #ededed;
}

.summaryBar {
  min-height: 32px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  border-bottom: 1px solid #b7b7b7;
  background: #e6e6e6;
}

.summaryText {
  font-size: 12px;
  color: #202020;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.contentArea {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.viewerArea {
  flex: 1;
  min-height: 0;
  background: #ededed;
}

.viewerPad {
  height: 100%;
  overflow: auto;
  padding: 10px 14px 14px;
}

.treePad {
  padding: 8px 10px 12px;
}

.websocketPad {
  padding: 0;
  background: #f6f6f6;
}

.websocketTable {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.websocketTable th,
.websocketTable td {
  border-bottom: 1px solid #c9c9c9;
  padding: 6px 8px;
  text-align: left;
  vertical-align: top;
}

.websocketTable thead th {
  background: #ececec;
  font-weight: 500;
}

.websocketMessage {
  white-space: pre-wrap;
  word-break: break-word;
}

.websocketEmpty {
  color: #7a7a7a;
}

.treeRoot {
  min-height: 100%;
}

.iframePad {
  padding: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.previewIframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

.imagePad {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f2f2f2;
}

.imageWrap {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
}

.previewImage {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  box-shadow: 0 0 0 1px #c8c8c8;
  background: #fff;
}

.bodyContent {
  margin: 0;
  min-height: 100%;
  font-size: 12px;
  line-height: 1.45;
  color: #202020;
  white-space: pre-wrap;
  word-break: break-all;
  background: transparent;
  cursor: context-menu;
}

.hexContent {
  font-family: Consolas, Monaco, monospace;
}

.jsonContent {
  font-family: Consolas, Monaco, monospace;
  background: #ffffff;
  border: 1px solid #d9d9d9;
  padding: 8px 0;
  border-radius: 2px;
  white-space: pre;
  word-break: normal;
  overflow: auto;
  counter-reset: json-line;
}

.jsonContent :global(.json-line) {
  display: block;
  padding: 0 12px;
  line-height: 1.4;
}

.jsonContent :global(.json-line:hover) {
  background: #f5f5f5;
}

.jsonContent :global(.json-line::before) {
  counter-increment: json-line;
  content: counter(json-line);
  display: inline-block;
  width: 40px;
  text-align: right;
  margin-right: 16px;
  color: #999;
  user-select: none;
  border-right: 1px solid #eee;
  padding-right: 8px;
}

.jsonContent :global(.json-key) {
  color: #000000;
  font-weight: bold;
}

.jsonContent :global(.json-str) {
  color: #c41a16;
}

.jsonContent :global(.json-num) {
  color: #1c00ce;
}

.jsonContent :global(.json-bool) {
  color: #aa0d91;
  font-weight: bold;
}

.jsonContent :global(.json-null) {
  color: #aa0d91;
  font-weight: bold;
}

.tabBar {
  height: 31px;
  display: flex;
  align-items: stretch;
  gap: 0;
  border-top: 1px solid #b5b5b5;
  background: #e1e1e1;
  flex-shrink: 0;
}

.tab {
  border: none;
  background: transparent;
  padding: 0 14px;
  font-size: 12px;
  color: #171717;
  cursor: pointer;
  position: relative;
}

.tab:hover {
  background: #ededed;
}

.tabActive {
  background: #efefef;
}

.tabActive::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: #75a9df;
}
</style>
