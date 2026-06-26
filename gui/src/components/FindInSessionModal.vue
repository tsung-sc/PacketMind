<template>
  <a-modal
    :open="visible"
    :footer="null"
    :closable="false"
    :width="600"
    wrapClassName="packetmind-find-modal"
    :bodyStyle="{ padding: '0' }"
    @cancel="handleClose"
  >
    <div :class="$style.shell">
      <div :class="$style.titleBar">
        <div :class="$style.titleGroup">
          <div :class="$style.title">Find in Session</div>
        </div>
        <a-button size="small" :class="$style.closeBtn" @click="handleClose">Close</a-button>
      </div>

      <div :class="$style.content">
        <div :class="$style.searchRow">
          <label :class="$style.searchLabel">Find:</label>
          <div :class="$style.searchInputWrapper">
            <a-input
              v-model:value="searchText"
              size="small"
              :class="$style.searchInput"
              @keydown.enter="handleFind"
              ref="searchInputRef"
            />
            <div :class="$style.searchHint">Using glob syntax, e.g. *test* or *.jpg</div>
          </div>
        </div>

        <div :class="$style.optionsRow">
          <a-checkbox v-model:checked="regex">Regular expression</a-checkbox>
          <a-checkbox v-model:checked="caseSensitive">Case sensitive</a-checkbox>
          <a-checkbox v-model:checked="wholeWords">Whole words</a-checkbox>
        </div>

        <div :class="$style.fieldset">
          <div :class="$style.legend">Scope</div>
          <div :class="$style.fieldsetBody">
            <div :class="$style.scopeColumn">
              <a-radio-group v-model:value="searchScope" :class="$style.radioGroup">
                <a-radio value="session" :class="$style.radioItem">Session</a-radio>
                <a-radio value="selected" :class="$style.radioItem">Selected</a-radio>
                <a-radio value="path" :class="$style.radioItem">
                  <span style="display: inline-block; width: 36px">Path</span>
                  <a-input
                    v-model:value="searchPath"
                    size="small"
                    :disabled="searchScope !== 'path'"
                    :class="$style.pathInput"
                  />
                </a-radio>
              </a-radio-group>
            </div>
            <div :class="$style.includeColumn">
              <div :class="$style.checkboxGroup">
                <a-checkbox v-model:checked="includeReqUrl" :class="$style.checkItem">Request URL</a-checkbox>
                <a-checkbox v-model:checked="includeReqHeader" :class="$style.checkItem">Request Header</a-checkbox>
                <a-checkbox v-model:checked="includeReqBody" :class="$style.checkItem">Request Body</a-checkbox>
                <a-checkbox v-model:checked="includeRespHeader" :class="$style.checkItem">Response Header</a-checkbox>
                <a-checkbox v-model:checked="includeRespBody" :class="$style.checkItem">Response Body</a-checkbox>
              </div>
            </div>
          </div>
        </div>

        <div :class="$style.resultsArea">
          <div v-if="matches.length === 0 && !isSearching" :class="$style.emptyState">
            {{ resultsText || 'No searches performed yet.' }}
          </div>
          <div v-else-if="isSearching" :class="$style.emptyState">
            Searching...
          </div>
          <a-tree
            v-else
            :tree-data="treeData"
            v-model:expanded-keys="expandedKeys"
            v-model:selected-keys="selectedKeys"
            show-icon
            :class="$style.tree"
            @select="onNodeSelect"
          >
            <template #icon="{ dataRef }">
              <GlobalOutlined v-if="dataRef.type === 'host'" :class="$style.treeIconHost" />
              <FolderOutlined v-else-if="dataRef.type === 'path'" :class="$style.treeIconFolder" />
              <FileTextOutlined v-else :class="$style.treeIconFile" />
            </template>
            <template #title="{ dataRef }">
              <span v-if="dataRef.type === 'host' || dataRef.type === 'path'" :class="$style.nodeTitle">
                {{ dataRef.title }} <span :class="$style.matchCount">({{ dataRef.matchCount }} matches)</span>
              </span>
              <div v-else :class="$style.matchNode">
                <span :class="[$style.methodTag, $style['method' + (dataRef.match.method || 'GET')]]">{{ dataRef.match.method || 'REQ' }}</span>
                <div :class="$style.matchContent">
                  <div :class="$style.matchField">{{ dataRef.match.field }}{{ dataRef.match.field_key ? ' (' + dataRef.match.field_key + ')' : '' }}</div>
                  <div :class="$style.matchPreview">{{ dataRef.title }}</div>
                </div>
              </div>
            </template>
          </a-tree>
        </div>
      </div>

      <div :class="$style.footer">
        <span :class="$style.resultsCount">Displaying {{ resultsCount }} results</span>
        <div :class="$style.actions">
          <a-button size="small" :class="$style.actionBtn" @click="handlePrev" :disabled="matches.length === 0">Previous</a-button>
          <a-button size="small" :class="$style.actionBtn" @click="handleNext" :disabled="matches.length === 0">Next</a-button>
          <a-button size="small" @click="handleClose" :class="$style.actionBtn">Cancel</a-button>
          <a-button size="small" type="primary" @click="handleFind" :class="$style.findBtn" :loading="isSearching">Find</a-button>
        </div>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, useCssModule } from 'vue';
import { GlobalOutlined, FolderOutlined, FileTextOutlined } from '@ant-design/icons-vue';
import { requestApi } from '@/api/wails';
import { useSessionStore } from '@/stores/sessionStore';
import { useRequestStore } from '@/stores/requestStore';

interface Props {
  visible: boolean;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const searchInputRef = ref<any>(null);
const sessionStore = useSessionStore();
const requestStore = useRequestStore();

// State
const searchText = ref('');
const regex = ref(false);
const caseSensitive = ref(false);
const wholeWords = ref(false);
const searchScope = ref('session');
const searchPath = ref('');

const includeReqUrl = ref(true);
const includeReqHeader = ref(true);
const includeReqBody = ref(true);
const includeRespHeader = ref(true);
const includeRespBody = ref(true);

const resultsText = ref('');
const resultsCount = ref(0);
const matches = ref<any[]>([]);
const currentMatchIndex = ref(-1);

const treeData = ref<any[]>([]);
const expandedKeys = ref<string[]>([]);
const selectedKeys = ref<string[]>([]);

const isSearching = ref(false);

// Focus on open
watch(() => props.visible, (val) => {
  if (val) {
    nextTick(() => {
      searchInputRef.value?.focus();
    });
  } else {
    // Reset state on close if desired, or keep to remember last search
  }
});

const handleClose = () => {
  emit('update:visible', false);
};

const handleFind = async () => {
  if (!searchText.value.trim()) {
    resultsText.value = 'Please enter a search term.';
    return;
  }

  isSearching.value = true;
  resultsText.value = 'Searching...';
  resultsCount.value = 0;
  matches.value = [];
  currentMatchIndex.value = -1;

  try {
    const sessionId = sessionStore.activeSession?.id || 'default';
    const opts = {
      session_id: sessionId,
      query: searchText.value,
      is_regex: regex.value,
      is_case_sensitive: caseSensitive.value,
      is_whole_word: wholeWords.value,
      include_req_url: includeReqUrl.value,
      include_req_header: includeReqHeader.value,
      include_req_body: includeReqBody.value,
      include_resp_header: includeRespHeader.value,
      include_resp_body: includeRespBody.value,
      include_notes: true,
      include_error: true,
    };
    
    // Need to use the raw window.go.bindings since the generator might not have exported it to requestApi properly yet, or we use requestApi if it exists
    let res;
    const w = window as any;
    if (w.go?.bindings?.RequestAPI?.FindInSession) {
      res = await w.go.bindings.RequestAPI.FindInSession(opts);
    } else if ((requestApi as any).FindInSession) {
      res = await (requestApi as any).FindInSession(opts);
    } else {
      throw new Error('FindInSession API not found');
    }

    if (res.code === 0) {
      let data = res.data || [];
      // Filter by scope
      if (searchScope.value === 'selected' && requestStore.selectedRequest) {
        data = data.filter((m: any) => m.request_id === requestStore.selectedRequest?.id);
      } else if (searchScope.value === 'path' && searchPath.value) {
        data = data.filter((m: any) => m.path.includes(searchPath.value) || m.url.includes(searchPath.value));
      }
      
      matches.value = data;
      resultsCount.value = data.length;
      
      if (data.length === 0) {
        resultsText.value = 'No matches found.';
        treeData.value = [];
        expandedKeys.value = [];
        selectedKeys.value = [];
      } else {
        buildTreeData(data);
      }
    } else {
      resultsText.value = `Error: ${res.message}`;
    }
  } catch (error: any) {
    resultsText.value = `Error: ${error.message}`;
  } finally {
    isSearching.value = false;
  }
};

const handleNext = () => {
  if (matches.value.length === 0) return;
  currentMatchIndex.value = (currentMatchIndex.value + 1) % matches.value.length;
  updateSelection();
};

const handlePrev = () => {
  if (matches.value.length === 0) return;
  currentMatchIndex.value = (currentMatchIndex.value - 1 + matches.value.length) % matches.value.length;
  updateSelection();
};

const onNodeSelect = (keys: any[], info: any) => {
  if (keys.length > 0 && info.node.type === 'match') {
    currentMatchIndex.value = info.node.matchIndex;
    selectedKeys.value = keys;
    // Highlight request in background if we want, currently just updating index
    requestStore.selectedRequestId = info.node.match.request_id;
  } else {
    // Keep the previous selection if clicking on a folder
    selectedKeys.value = [`match-${currentMatchIndex.value}`];
  }
};

const updateSelection = () => {
  if (matches.value.length === 0 || currentMatchIndex.value < 0) {
    selectedKeys.value = [];
    return;
  }
  selectedKeys.value = [`match-${currentMatchIndex.value}`];
  
  // Also select the request in the request store
  const match = matches.value[currentMatchIndex.value];
  if (match) {
    requestStore.selectedRequestId = match.request_id;
  }
  scrollToSelected();
};

const style = useCssModule();

const scrollToSelected = () => {
  nextTick(() => {
    const el = document.querySelector(`.${style.resultsArea} .ant-tree-treenode-selected`);
    if (el) {
      el.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  });
};

const buildTreeData = (data: any[]) => {
  const hostsMap = new Map<string, any>();
  const allKeys = new Set<string>();

  data.forEach((m, idx) => {
    let urlObj;
    let host = 'unknown';
    let path = m.url || '/';
    
    try {
      urlObj = new URL(m.url);
      host = urlObj.host || 'unknown';
      path = urlObj.pathname || '/';
    } catch {
      // Keep fallbacks
    }

    if (!hostsMap.has(host)) {
      hostsMap.set(host, {
        key: `host-${host}`,
        title: host,
        type: 'host',
        matchCount: 0,
        childrenMap: new Map<string, any>()
      });
      allKeys.add(`host-${host}`);
    }

    const hostNode = hostsMap.get(host);
    hostNode.matchCount++;

    if (!hostNode.childrenMap.has(path)) {
      hostNode.childrenMap.set(path, {
        key: `path-${host}-${path}`,
        title: path,
        type: 'path',
        matchCount: 0,
        childrenList: []
      });
      allKeys.add(`path-${host}-${path}`);
    }

    const pathNode = hostNode.childrenMap.get(path);
    pathNode.matchCount++;
    
    pathNode.childrenList.push({
      key: `match-${idx}`,
      title: m.preview.trim() || '(No preview available)',
      type: 'match',
      match: m,
      matchIndex: idx,
      isLeaf: true
    });
  });

  const finalTree = Array.from(hostsMap.values()).map(h => {
    return {
      key: h.key,
      title: h.title,
      type: h.type,
      matchCount: h.matchCount,
      children: Array.from(h.childrenMap.values()).map((p: any) => {
        return {
          key: p.key,
          title: p.title,
          type: p.type,
          matchCount: p.matchCount,
          children: p.childrenList
        };
      })
    };
  });

  treeData.value = finalTree;
  expandedKeys.value = Array.from(allKeys); // Expand all by default
  
  currentMatchIndex.value = 0;
  updateSelection();
};
</script>

<style module>
.shell {
  display: flex;
  flex-direction: column;
  background: #ececec;
  color: #1f1f1f;
  min-height: 520px;
  border-radius: 4px;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.titleBar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: linear-gradient(180deg, #f5f5f5 0%, #e0e0e0 100%);
  border-bottom: 1px solid #c0c0c0;
}

.titleGroup {
  min-width: 0;
}

.title {
  font-size: 13px;
  font-weight: 600;
  color: #1c1c1c;
}

.closeBtn {
  border-radius: 2px !important;
}

.content {
  flex: 1;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.searchRow {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.searchLabel {
  font-size: 12px;
  font-weight: 500;
  padding-top: 4px;
  width: 40px;
  text-align: right;
  color: #333;
}

.searchInputWrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.searchInput {
  border-radius: 2px !important;
}

.searchHint {
  font-size: 11px;
  color: #777;
}

.optionsRow {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-left: 48px; /* Align with input */
  margin-bottom: 4px;
}

.optionsRow :global(.ant-checkbox-wrapper) {
  font-size: 12px;
  color: #333;
}

.fieldset {
  margin-top: 6px;
  border: 1px solid #c0c0c0;
  border-radius: 2px;
  padding: 12px;
  position: relative;
  background: #f4f4f4;
}

.legend {
  position: absolute;
  top: -8px;
  left: 8px;
  background: #ececec;
  padding: 0 4px;
  font-size: 11px;
  font-weight: 600;
  color: #444;
}

.fieldsetBody {
  display: flex;
  gap: 32px;
}

.scopeColumn {
  flex: 1;
}

.includeColumn {
  flex: 1;
}

.radioGroup {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radioItem {
  font-size: 12px;
  color: #333;
  display: flex;
  align-items: center;
}

.pathInput {
  margin-left: 8px;
  width: 140px;
  border-radius: 2px !important;
}

.checkboxGroup {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.checkItem {
  font-size: 12px;
  color: #333;
  margin-left: 0 !important; /* Ant design sets margin-left on sibling checkboxes */
}

.resultsArea {
  margin-top: 8px;
  flex: 1;
  display: flex;
  background: #fff;
  border: 1px solid #c0c0c0;
  border-radius: 2px;
  overflow: auto;
}

.emptyState {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #888;
  font-size: 12px;
}

.tree {
  flex: 1;
  font-size: 12px;
  padding: 4px;
}

.tree :global(.ant-tree-node-content-wrapper) {
  display: flex;
  align-items: center;
  flex: 1;
  overflow: hidden;
  line-height: 24px;
}

.tree :global(.ant-tree-title) {
  flex: 1;
  overflow: hidden;
}

.treeIconHost, .treeIconFolder, .treeIconFile {
  font-size: 14px;
  margin-right: 4px;
}

.treeIconHost { color: #1890ff; }
.treeIconFolder { color: #faad14; }
.treeIconFile { color: #52c41a; }

.nodeTitle {
  font-weight: 500;
  color: #333;
}

.matchCount {
  color: #888;
  font-weight: normal;
  font-size: 11px;
  margin-left: 4px;
}

.matchNode {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
}

.methodTag {
  font-size: 10px;
  font-weight: 600;
  padding: 0 4px;
  border-radius: 2px;
  color: #fff;
  min-width: 40px;
  text-align: center;
  margin-top: 2px;
}

.methodGET { background-color: #007bb6; }
.methodPOST { background-color: #00a884; }
.methodPUT { background-color: #ff9900; }
.methodDELETE { background-color: #e53935; }
.methodPATCH { background-color: #8e44ad; }
.methodOPTIONS { background-color: #ff5722; }
.methodREQ { background-color: #607d8b; }

.matchContent {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.matchField {
  font-size: 11px;
  color: #666;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.matchPreview {
  font-family: Consolas, Monaco, "Courier New", monospace;
  font-size: 11px;
  color: #222;
  background: #f5f5f5;
  padding: 2px 4px;
  border-radius: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
  max-width: 100%;
}

.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: #e4e4e4;
  border-top: 1px solid #c0c0c0;
}

.resultsCount {
  font-size: 12px;
  color: #555;
}

.actions {
  display: flex;
  gap: 8px;
}

.actionBtn, .findBtn {
  border-radius: 2px !important;
  min-width: 72px;
}

.findBtn {
  background: linear-gradient(180deg, #1890ff 0%, #096dd9 100%) !important;
  border-color: #096dd9 !important;
}

.findBtn:hover, .findBtn:focus {
  background: linear-gradient(180deg, #40a9ff 0%, #1890ff 100%) !important;
  border-color: #1890ff !important;
}
</style>
