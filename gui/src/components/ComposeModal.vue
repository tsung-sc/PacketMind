<template>
  <a-modal
    :open="visible"
    :footer="null"
    :closable="false"
    :width="820"
    wrapClassName="packetmind-compose-modal"
    :bodyStyle="{ padding: '0' }"
    @cancel="handleClose"
  >
    <div :class="$style.shell">
      <div :class="$style.titleBar">
        <div :class="$style.titleGroup">
          <div :class="$style.title">Compose</div>
          <div :class="$style.subtitle">Edit and resend the captured request</div>
        </div>
        <a-button size="small" :class="$style.closeBtn" @click="handleClose">Close</a-button>
      </div>

      <div :class="$style.toolbar">
        <a-select v-model:value="method" size="small" :class="$style.methodSelect">
          <a-select-option v-for="item in methodOptions" :key="item" :value="item">{{ item }}</a-select-option>
        </a-select>
        <a-input v-model:value="url" size="small" :class="$style.urlInput" placeholder="https://example.com/path" />
        <a-button
          type="primary"
          size="small"
          :loading="sending"
          :class="$style.sendBtn"
          @click="handleSend"
        >
          <template #icon><SendOutlined /></template>
          Send
        </a-button>
      </div>

      <div :class="$style.content">
        <div :class="$style.panel">
          <button type="button" :class="$style.sectionHeader" @click="headersExpanded = !headersExpanded">
            <span :class="$style.sectionTitleWrap">
              <component :is="headersExpanded ? DownOutlined : RightOutlined" :class="$style.sectionArrow" />
              <span :class="$style.sectionTitle">Headers</span>
              <span :class="$style.sectionMeta">{{ headerRows.length }} items</span>
            </span>
            <a-button size="small" :class="$style.sectionAction" @click.stop="addHeaderRow">Add Header</a-button>
          </button>

          <div v-if="headersExpanded" :class="$style.sectionBody">
            <div v-if="headerRows.length === 0" :class="$style.emptyState">No headers. Add one to override defaults.</div>
            <div v-else :class="$style.headerList">
              <div v-for="row in headerRows" :key="row.id" :class="$style.headerRow">
                <a-input
                  :value="row.name"
                  size="small"
                  :class="$style.headerName"
                  placeholder="Header name"
                  @update:value="updateHeaderRow(row.id, 'name', $event)"
                />
                <a-input
                  :value="row.value"
                  size="small"
                  :class="$style.headerValue"
                  placeholder="Header value"
                  @update:value="updateHeaderRow(row.id, 'value', $event)"
                />
                <a-button size="small" danger :class="$style.rowDelete" @click="removeHeaderRow(row.id)">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <div :class="$style.panel">
          <button type="button" :class="$style.sectionHeader" @click="bodyExpanded = !bodyExpanded">
            <span :class="$style.sectionTitleWrap">
              <component :is="bodyExpanded ? DownOutlined : RightOutlined" :class="$style.sectionArrow" />
              <span :class="$style.sectionTitle">Body</span>
              <span :class="$style.sectionMeta">{{ body.length }} chars</span>
            </span>
          </button>

          <div v-if="bodyExpanded" :class="$style.sectionBody">
            <div :class="$style.contentHint">Content-Type: {{ contentTypeHint }}</div>
            <a-textarea v-model:value="body" :rows="10" :class="$style.textArea" placeholder="Request body" />
          </div>
        </div>

        <div v-if="sendError" :class="$style.errorBanner">{{ sendError }}</div>

        <div v-if="responseVisible" :class="$style.panel">
          <div :class="$style.responseHeader">
            <div :class="$style.sectionTitleWrap">
              <span :class="$style.sectionTitle">Response</span>
              <span :class="$style.responseStat">Status {{ responseStatus }}</span>
              <span :class="$style.responseStat">{{ responseDuration }}</span>
            </div>
            <a-button size="small" :disabled="!responseBody" @click="copyResponse">Copy Response</a-button>
          </div>
          <div :class="$style.sectionBody">
            <a-textarea :value="responseBody" :rows="12" readonly :class="$style.textArea" />
          </div>
        </div>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import { DeleteOutlined, DownOutlined, RightOutlined, SendOutlined } from '@ant-design/icons-vue';
import { requestApi } from '@/api/wails';
import type { Request } from '@/types';
import { copyToClipboard } from '@/utils/wails';

interface Props {
  visible: boolean;
  request: Request | null;
}

interface HeaderRow {
  id: string;
  name: string;
  value: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

const method = ref('GET');
const url = ref('');
const body = ref('');
const headersExpanded = ref(true);
const bodyExpanded = ref(true);
const sending = ref(false);
const sendError = ref('');
const responseData = ref<any>(null);
const headerRows = ref<HeaderRow[]>([]);
const headersMap = ref<Map<string, string>>(new Map());

const buildRowId = () => `header-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;

const syncHeadersMap = () => {
  const next = new Map<string, string>();
  headerRows.value.forEach((row) => {
    const name = row.name.trim();
    if (!name) return;
    next.set(name, row.value);
  });
  headersMap.value = next;
};

const resetState = () => {
  method.value = 'GET';
  url.value = '';
  body.value = '';
  headerRows.value = [];
  headersMap.value = new Map();
  headersExpanded.value = true;
  bodyExpanded.value = true;
  sending.value = false;
  sendError.value = '';
  responseData.value = null;
};

const fillFromRequest = (request: Request | null) => {
  resetState();
  if (!request) return;

  method.value = (request.method || 'GET').toUpperCase();
  url.value = request.url || '';
  body.value = request.body || '';
  headerRows.value = Object.entries(request.headers || {}).map(([name, values]) => ({
    id: buildRowId(),
    name,
    value: Array.isArray(values) && values.length > 0 ? String(values[0] ?? '') : '',
  }));
  syncHeadersMap();
};

watch(() => [props.visible, props.request] as const, ([visible, request]) => {
  if (visible) {
    fillFromRequest(request);
    return;
  }
  resetState();
}, { immediate: true });

const contentTypeHint = computed(() => props.request?.content_type || 'No Content-Type header');

const responseVisible = computed(() => Boolean(responseData.value));
const responseStatus = computed(() => {
  const value = responseData.value;
  if (!value) return '-';
  return value.status_code || value.status || value.code || '-';
});
const responseDuration = computed(() => {
  const value = responseData.value;
  if (!value) return '-';
  const duration = value.duration ?? value.elapsed_ms ?? value.elapsed;
  return typeof duration === 'number' ? `${duration} ms` : '-';
});
const responseBody = computed(() => {
  const value = responseData.value;
  if (!value) return '';
  if (typeof value.resp_body === 'string') return value.resp_body;
  if (typeof value.body === 'string') return value.body;
  if (typeof value.response_body === 'string') return value.response_body;
  return value.message || '';
});

const updateHeaderRow = (id: string, field: 'name' | 'value', value: string) => {
  const row = headerRows.value.find((item) => item.id === id);
  if (!row) return;
  row[field] = value;
  syncHeadersMap();
};

const addHeaderRow = () => {
  headerRows.value.push({ id: buildRowId(), name: '', value: '' });
  syncHeadersMap();
};

const removeHeaderRow = (id: string) => {
  headerRows.value = headerRows.value.filter((row) => row.id !== id);
  syncHeadersMap();
};

const buildHeadersPayload = () => {
  syncHeadersMap();
  const result: Record<string, string[]> = {};
  headersMap.value.forEach((value, key) => {
    result[key] = [value];
  });
  return result;
};

const handleClose = () => {
  emit('update:visible', false);
};

const handleSend = async () => {
  if (!url.value.trim()) {
    sendError.value = 'URL is required';
    return;
  }

  sending.value = true;
  sendError.value = '';
  responseData.value = null;

  try {
    const response = await requestApi.compose({
      method: method.value,
      url: url.value.trim(),
      headers: buildHeadersPayload(),
      body: body.value,
    });

    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to send composed request');
    }

    responseData.value = response.data || {};
    message.success('Request sent');
    emit('update:visible', false);
  } catch (error: any) {
    sendError.value = error?.message || 'Failed to send composed request';
    message.error(sendError.value);
  } finally {
    sending.value = false;
  }
};

const copyResponse = async () => {
  if (!responseBody.value) return;
  const ok = await copyToClipboard(responseBody.value);
  if (ok) message.success('Response copied');
  else message.error('Failed to copy response');
};
</script>

<style module>
.shell {
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
  color: #1f1f1f;
  min-height: 680px;
  max-height: 78vh;
}

.titleBar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: linear-gradient(180deg, #efefef 0%, #dedede 100%);
  border-bottom: 1px solid #adadad;
}

.titleGroup {
  min-width: 0;
}

.title {
  font-size: 13px;
  font-weight: 600;
  color: #1c1c1c;
}

.subtitle {
  margin-top: 2px;
  font-size: 11px;
  color: #666;
}

.closeBtn {
  border-radius: 2px !important;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f0f0f0;
  border-bottom: 1px solid #c8c8c8;
}

.methodSelect {
  width: 108px;
  flex: 0 0 auto;
}

.urlInput {
  flex: 1 1 auto;
  min-width: 0;
}

.sendBtn {
  flex: 0 0 auto;
  border-radius: 2px !important;
  background: linear-gradient(180deg, #52c41a 0%, #389e0d 100%) !important;
  border-color: #2f7d0c !important;
}

.sendBtn:hover,
.sendBtn:focus {
  background: linear-gradient(180deg, #5fd028 0%, #3ea611 100%) !important;
  border-color: #2f7d0c !important;
}

.content {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.panel {
  border: 1px solid #bebebe;
  background: #fcfcfc;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.85);
}

.sectionHeader,
.responseHeader {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 10px;
  background: linear-gradient(180deg, #f7f7f7 0%, #ececec 100%);
  border: 0;
  border-bottom: 1px solid #d6d6d6;
  text-align: left;
}

.sectionHeader {
  cursor: pointer;
}

.sectionTitleWrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.sectionArrow {
  font-size: 11px;
  color: #5d5d5d;
}

.sectionTitle {
  font-size: 12px;
  font-weight: 600;
  color: #212121;
}

.sectionMeta,
.responseStat {
  font-size: 11px;
  color: #6e6e6e;
}

.sectionAction {
  border-radius: 2px !important;
}

.sectionBody {
  padding: 10px;
}

.headerList {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.headerRow {
  display: grid;
  grid-template-columns: minmax(150px, 0.34fr) minmax(0, 1fr) 34px;
  gap: 6px;
  align-items: center;
}

.headerName,
.headerValue {
  min-width: 0;
}

.rowDelete {
  border-radius: 2px !important;
}

.contentHint {
  margin-bottom: 8px;
  font-size: 11px;
  color: #666;
}

.textArea :global(.ant-input) {
  font-family: Consolas, "SFMono-Regular", Menlo, Monaco, monospace;
  font-size: 12px;
  line-height: 1.5;
  border-radius: 2px !important;
  background: #fff;
}

.emptyState {
  padding: 8px;
  border: 1px dashed #cfcfcf;
  background: #fafafa;
  font-size: 11px;
  color: #777;
}

.errorBanner {
  border: 1px solid #e0a9a9;
  background: #fff2f0;
  color: #a33a3a;
  padding: 8px 10px;
  font-size: 12px;
}
</style>
