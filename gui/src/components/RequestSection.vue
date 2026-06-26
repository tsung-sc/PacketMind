<template>
  <div :class="$style.container">
    <div :class="$style.summaryBar">
      <span :class="$style.summaryText">{{ requestLine }}</span>
    </div>

    <div :class="$style.contentArea">
      <div :class="$style.viewerArea">
        <HeadersView
          v-if="activeView === 'headers'"
          :headers="headers"
          :requestId="requestId"
          :location="'Request Headers'"
          :leadLine="requestLine"
          :isNarrow="isNarrow"
        />

        <div v-else-if="activeView === 'query'" :class="[$style.viewerPad, isNarrow ? $style.narrowDetail : '']">
          <div v-if="queryRows.length" :class="$style.kvRows">
            <div v-for="row in queryRows" :key="row.key" :class="$style.kvRow">
              <div :class="$style.kvKey">{{ row.key }}</div>
              <div
                :class="$style.kvValue"
                @contextmenu.stop.prevent="openContextMenu($event, row.key, row.value, 'Request Query String')"
              >
                {{ row.value }}
              </div>
            </div>
          </div>
          <a-empty v-else description="No query parameters" />
        </div>

        <div v-else-if="activeView === 'cookies'" :class="[$style.viewerPad, isNarrow ? $style.narrowDetail : '']">
          <div v-if="cookieRows.length" :class="$style.kvRows">
            <div v-for="row in cookieRows" :key="row.key" :class="$style.kvRow">
              <div :class="$style.kvKey">{{ row.key }}</div>
              <div
                :class="$style.kvValue"
                @contextmenu.stop.prevent="openContextMenu($event, row.key, row.value, 'Request Cookies')"
              >
                {{ row.value }}
              </div>
            </div>
          </div>
          <a-empty v-else description="No cookies" />
        </div>

        <div v-else-if="activeView === 'text'" :class="$style.viewerPad">
          <pre
            :class="$style.bodyContent"
            @contextmenu.stop.prevent="openContextMenu($event, 'Request Body', formattedBody, 'Request Body Text')"
          >{{ formattedBody }}</pre>
        </div>

        <div v-else-if="activeView === 'json_text'" :class="$style.viewerPad">
          <pre
            :class="[$style.bodyContent, $style.jsonContent]"
            @contextmenu.stop.prevent="openContextMenu($event, 'Request JSON', formattedJSON, 'Request JSON Text')"
          ><span v-html="highlightedJSON"></span></pre>
        </div>

        <div v-else-if="activeView === 'tree'" :class="[$style.viewerPad, $style.treePad]">
          <div v-if="bodyTree" :class="$style.treeRoot">
            <BodyTreeNodeView :node="bodyTree.root" :depth="0" />
          </div>
          <a-empty v-else description="Body tree unavailable" />
        </div>

        <div v-else-if="activeView === 'hex'" :class="$style.viewerPad">
          <pre
            :class="[$style.bodyContent, $style.hexContent]"
            @contextmenu.stop.prevent="openContextMenu($event, 'Request Body Hex', hexBody, 'Request Body Hex')"
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
              <tr v-if="!requestWebSocketFrames.length">
                <td colspan="4" :class="$style.websocketEmpty">No request-side WebSocket messages</td>
              </tr>
              <tr v-for="frame in requestWebSocketFrames" :key="frame.id">
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
            @contextmenu.stop.prevent="openContextMenu($event, 'Request Raw', rawRequest, 'Request Raw')"
          >{{ rawRequest }}</pre>
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
import { EMPTY_BODY_PLACEHOLDER, formatBodyForTextAsync, isJSONContentType, parseStructuredBodyAsync, StructuredBodyTree } from './bodyTreeUtils';

interface Props {
  requestId: string;
  method: string;
  url: string;
  path: string;
  queryString: string;
  httpVersion: string;
  headers: Record<string, string[]> | null;
  body: string | null;
  contentType: string | null;
  cookies: Record<string, string> | null;
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

type ViewKey = (typeof baseTabs)[number]['key'] | 'tree' | 'json_text';
type ExtendedViewKey = ViewKey | 'query' | 'cookies' | 'text' | 'hex' | 'raw' | 'websocket';
const activeView = ref<ExtendedViewKey>('headers');

const requestLine = computed(() => {
  const target = props.path || '/';
  const query = props.queryString ? `?${props.queryString}` : '';
  return `${props.method} ${target}${query}`;
});

const formattedBody = ref<string>(EMPTY_BODY_PLACEHOLDER);
const formattedJSON = ref<string>(EMPTY_BODY_PLACEHOLDER);
const bodyTree = ref<StructuredBodyTree | null>(null);

const hasQueryString = computed(() => !!props.queryString);
const hasCookies = computed(() => !!props.cookies && Object.keys(props.cookies).length > 0);
const hasBody = computed(() => !!props.body && props.body.trim().length > 0);
const hasJSONText = ref(false);

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

const requestWebSocketFrames = computed(() => (props.websocketFrames || []).filter((frame) => frame.direction === 'sent'));
const hasWebSocketFrames = computed(() => requestWebSocketFrames.value.length > 0);

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
    const [fmtBody, tree] = await Promise.all([
      formatBodyForTextAsync(props.body, props.contentType, ce),
      parseStructuredBodyAsync(props.body, props.contentType, ce)
    ]);
    formattedBody.value = fmtBody;
    formattedJSON.value = fmtBody;
    bodyTree.value = tree;
    hasJSONText.value = isJSONContentType(props.contentType) && !!props.body;
  },
  { immediate: true, deep: true }
);

const tabs = computed(() => {
  const list: Array<{ key: ViewKey; label: string }> = [...baseTabs];
  if (hasQueryString.value) {
    list.push({ key: 'query', label: 'Query String' });
  }
  if (hasCookies.value) {
    list.push({ key: 'cookies', label: 'Cookies' });
  }
  if (hasBody.value) {
    list.push({ key: 'text', label: 'Text' });
    if (hasJSONText.value) {
      list.push({ key: 'json_text', label: 'JSON Text' });
    }
    if (bodyTree.value) {
      list.push({ key: 'tree', label: 'Tree' });
    }
    list.push({ key: 'hex', label: 'Hex' });
  }
  if (hasWebSocketFrames.value) {
    list.push({ key: 'websocket', label: 'WebSocket' });
  }
  list.push({ key: 'raw', label: 'Raw' });
  return list;
});

watch([bodyTree, hasJSONText, tabs], ([treeValue, jsonValue, availableTabs]) => {
  const allowed = new Set(availableTabs.map((tab) => tab.key));
  if (!treeValue && activeView.value === 'tree') {
    activeView.value = hasBody.value ? 'text' : 'raw';
    return;
  }
  if (!jsonValue && activeView.value === 'json_text') {
    activeView.value = hasBody.value ? 'text' : 'raw';
    return;
  }
  if (!allowed.has(activeView.value)) {
    activeView.value = allowed.has('headers') ? 'headers' : availableTabs[0]?.key || 'raw';
  }
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

const queryRows = computed(() => {
  if (!props.queryString) return [];
  return props.queryString
    .split('&')
    .filter(Boolean)
    .map((entry, index) => {
      const [key, ...rest] = entry.split('=');
      return {
        key: decodeURIComponent(key || `param_${index}`),
        value: decodeURIComponent(rest.join('=') || ''),
      };
    });
});

const cookieRows = computed(() => {
  if (!props.cookies) return [];
  return Object.entries(props.cookies).map(([key, value]) => ({ key, value }));
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

const rawRequest = computed(() => {
  const headerLines = Object.entries(props.headers || {})
    .flatMap(([name, values]) => (values || []).map((value) => `${name}: ${value}`));
  return [requestLine.value, ...headerLines, '', formattedBody.value].join('\n').trim();
});

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
  location: 'Request',
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

.kvRows {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.kvRow {
  display: grid;
  grid-template-columns: minmax(96px, 180px) minmax(0, 1fr);
  gap: 16px;
  padding: 5px 0;
}

.kvKey {
  text-align: right;
  font-size: 12px;
  font-weight: 600;
  color: #303030;
}

.kvValue {
  font-size: 12px;
  line-height: 1.45;
  color: #202020;
  word-break: break-all;
  cursor: context-menu;
  border-radius: 3px;
  padding: 0 2px;
}

.kvValue:hover {
  background: #e8f1fb;
}

.narrowDetail .kvRow {
  grid-template-columns: 1fr;
  gap: 2px 0;
}

.narrowDetail .kvKey {
  text-align: left;
}

.narrowDetail .kvValue {
  padding-left: 12px;
  word-break: break-word;
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
  border-right: 1px solid transparent;
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
