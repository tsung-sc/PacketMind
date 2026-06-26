<template>
  <div :class="$style.container">
    <div :class="$style.filterBar">
      <div :class="$style.filterColumn">
        <a-input-search
          ref="searchInputRef"
          v-model:value="searchKeyword"
          placeholder="Search host/path"
          allowClear
          :class="$style.searchInput"
          @search="onSearch"
        />
        <div :class="$style.actionRow">
          <span :class="$style.totalText">Total: {{ store.total }}</span>
          <a-button size="small" :class="$style.actionBtn" @click="expandAll">Expand All</a-button>
          <a-button size="small" :class="$style.actionBtn" @click="collapseAll">Collapse All</a-button>
        </div>
      </div>
    </div>

    <a-alert
      v-if="store.error"
      message="Error"
      :description="store.error"
      type="error"
      closable
      style="margin: 8px 16px"
    />

    <div :class="$style.treeContainer">
      <div v-if="store.loading && store.requests.length === 0" :class="$style.loading">
        <a-spin size="large" />
      </div>
      <a-empty
        v-else-if="store.requests.length === 0"
        description="No requests captured yet."
        style="margin-top: 50px"
      />
      <div v-else :class="$style.treeRoot">
        <div v-for="node in treeNodes" :key="node.key">
          <RequestTreeNode
            :node="node"
            :expandedKeys="expandedKeys"
            :recentNodeKeys="store.recentNodeKeys"
            :selected-id="store.selectedId"
            :on-contextmenu="showContextMenu"
            @toggle="toggleNode"
            @select="store.selectRequestById"
          />
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="request-context-menu"
        :style="contextMenuStyle"
        @click.stop
      >
        <template v-if="contextMenu.node?.request">
          <div class="request-menu-item" @click="handleCopyUrl">
            <CopyOutlined /> Copy URL
          </div>
          <div class="request-menu-item" @click="handleCopyCurl">
            <CopyOutlined /> Copy cURL
          </div>
          <div
            :class="['request-menu-item', !hasResponseBody ? 'request-menu-disabled' : '']"
            @click="handleCopyResponseBody"
          >
            <CopyOutlined /> Copy Response Body
          </div>
          <div class="request-menu-divider" />
          <div
            :class="['request-menu-item', !hasResponseBody ? 'request-menu-disabled' : '']"
            @click="handleSaveResponse"
          >
            <SaveOutlined /> Save Response...
          </div>
          <div class="request-menu-divider" />
          <div class="request-menu-item" @click="handleRepeatRequest">
            <ReloadOutlined /> Repeat
          </div>
          <div class="request-menu-item" @click="handleComposeRequest">
            <EditOutlined /> Compose
          </div>
          <div class="request-menu-divider" />
          <div class="request-menu-item" @click="handleAnalyzeRequest">
            <RobotOutlined /> Agent Analyze
          </div>
          <div class="request-menu-divider" />
          <div class="request-menu-item request-menu-item-danger" @click="handleDeleteRequest">
            <DeleteOutlined /> Delete
          </div>
        </template>
        <template v-else-if="contextMenu.node">
          <div class="request-menu-item" @click="handleCopyHost">
            <CopyOutlined /> Copy Host
          </div>
          <div class="request-menu-divider" />
          <div class="request-menu-item" @click="handleExpandAllFromMenu">
            <ExpandOutlined /> Expand All
          </div>
          <div class="request-menu-item" @click="handleCollapseAllFromMenu">
            <ShrinkOutlined /> Collapse All
          </div>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, reactive, ref, useCssModule, watch, type PropType } from 'vue';
import dayjs from 'dayjs';
import { message } from 'ant-design-vue';
import {
  RightOutlined,
  DownOutlined,
  LoadingOutlined,
  FolderFilled,
  FileTextOutlined,
  ApiOutlined,
  LockOutlined,
  GlobalOutlined,
  ThunderboltOutlined,
  CodeOutlined,
  SwapOutlined,
  Html5Outlined,
  PictureOutlined,
  BgColorsOutlined,
  FontSizeOutlined,
  ProfileOutlined,
  CloseCircleFilled,
  CopyOutlined,
  SaveOutlined,
  ReloadOutlined,
  EditOutlined,
  RobotOutlined,
  ExpandOutlined,
  ShrinkOutlined,
} from '@ant-design/icons-vue';
import { requestApi } from '@/api/wails';
import { isRequestPending, useRequestStore } from '@/stores/requestStore';
import type { Request } from '@/types';
import { copyToClipboard } from '@/utils/wails';

type TreeNodeKind = 'host' | 'folder' | 'request';

interface TreeNode {
  key: string;
  kind: TreeNodeKind;
  label: string;
  host?: string;
  request?: Request;
  children?: TreeNode[];
  meta?: string;
  iconType?: string;
  statusCode?: number;
  firstSeenAt?: string;
}

const emit = defineEmits<{
  (e: 'analyzeRequest', requestId: string): void;
  (e: 'composeRequest', requestId: string): void;
  (e: 'expandAll'): void;
  (e: 'collapseAll'): void;
}>();

const store = useRequestStore();
const expandedKeys = reactive(new Set<string>());
const styles = useCssModule();
const searchKeyword = ref(store.filters.search || '');
const searchInputRef = ref<any>(null);
const contextMenu = ref<{ visible: boolean; x: number; y: number; node: TreeNode | null }>({
  visible: false,
  x: 0,
  y: 0,
  node: null,
});

const responseFilenameExtensionMap: Array<[RegExp, string]> = [
  [/json/i, '.json'],
  [/html/i, '.html'],
  [/xml/i, '.xml'],
  [/javascript|ecmascript/i, '.js'],
  [/css/i, '.css'],
  [/svg/i, '.svg'],
  [/plain|text\//i, '.txt'],
  [/png/i, '.png'],
  [/jpe?g/i, '.jpg'],
  [/gif/i, '.gif'],
  [/webp/i, '.webp'],
  [/pdf/i, '.pdf'],
];

const textDecoder = new TextDecoder();
const textEncoder = new TextEncoder();

const isTextLikeContentType = (contentType?: string) => {
  const value = (contentType || '').toLowerCase();
  return value.startsWith('text/')
    || value.includes('json')
    || value.includes('xml')
    || value.includes('javascript')
    || value.includes('ecmascript')
    || value.includes('svg')
    || value.includes('x-www-form-urlencoded');
};

const decodeBase64Bytes = (value?: string) => {
  const normalized = (value || '').replace(/\s+/g, '');
  if (!normalized || normalized.length % 4 !== 0 || !/^[A-Za-z0-9+/]+={0,2}$/.test(normalized)) {
    return null;
  }

  try {
    const binary = window.atob(normalized);
    return Uint8Array.from(binary, (char) => char.charCodeAt(0));
  } catch {
    return null;
  }
};

const isMostlyTextBytes = (bytes: Uint8Array) => {
  if (!bytes.length) return true;

  let printable = 0;
  for (const byte of bytes) {
    if (byte === 9 || byte === 10 || byte === 13 || (byte >= 32 && byte <= 126) || byte >= 160) {
      printable += 1;
    }
  }

  return printable / bytes.length > 0.82;
};

const getResponseBodyBytes = (request: Request) => {
  if (!request.resp_body) {
    return new Uint8Array();
  }

  const decodedBytes = decodeBase64Bytes(request.resp_body);
  if (!decodedBytes) {
    return textEncoder.encode(request.resp_body);
  }

  if (isTextLikeContentType(request.resp_content_type) && !isMostlyTextBytes(decodedBytes)) {
    return textEncoder.encode(request.resp_body);
  }

  return decodedBytes;
};

const getResponseBodyText = (request: Request) => {
  if (!request.resp_body) {
    return '';
  }

  const decodedBytes = decodeBase64Bytes(request.resp_body);
  if (isTextLikeContentType(request.resp_content_type) && decodedBytes && isMostlyTextBytes(decodedBytes)) {
    return textDecoder.decode(decodedBytes);
  }

  if (isTextLikeContentType(request.resp_content_type)) {
    return request.resp_body;
  }

  if (decodedBytes && !isTextLikeContentType(request.resp_content_type)) {
    return null;
  }

  return request.resp_body;
};

const inferResponseExtension = (contentType?: string) => {
  const normalized = (contentType || '').split(';')[0].trim();
  const matched = responseFilenameExtensionMap.find(([pattern]) => pattern.test(normalized));
  return matched?.[1] || (normalized ? '.bin' : '.txt');
};

const inferResponseFilename = (request: Request) => {
  const fallbackExtension = inferResponseExtension(request.resp_content_type);

  try {
    const pathname = new URL(request.url).pathname || '';
    const segments = pathname.split('/').filter(Boolean);
    const lastSegment = decodeURIComponent(segments[segments.length - 1] || '').trim();

    if (!lastSegment) {
      return `response${fallbackExtension}`;
    }

    if (/\.[a-z0-9]+$/i.test(lastSegment)) {
      return lastSegment;
    }

    return `${lastSegment}${fallbackExtension}`;
  } catch {
    return `response${fallbackExtension}`;
  }
};

const triggerDownload = (blob: Blob, filename: string) => {
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = objectUrl;
  anchor.download = filename;
  anchor.style.display = 'none';
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0);
};

const contextMenuRequest = computed(() => contextMenu.value.node?.request || null);
const hasResponseBody = computed(() => Boolean(contextMenuRequest.value?.resp_body));

const contextMenuStyle = computed(() => {
  const requestMenu = Boolean(contextMenu.value.node?.request);
  const menuWidth = requestMenu ? 188 : 164;
  const menuHeight = requestMenu ? 236 : 106;
  let x = contextMenu.value.x;
  let y = contextMenu.value.y;

  if (x + menuWidth > window.innerWidth) {
    x = window.innerWidth - menuWidth - 10;
  }
  if (y + menuHeight > window.innerHeight) {
    y = window.innerHeight - menuHeight - 10;
  }

  return {
    left: `${Math.max(8, x)}px`,
    top: `${Math.max(8, y)}px`,
  };
});

watch(searchKeyword, (value) => {
  store.setFilters({ search: value });
});

onMounted(async () => {
  await store.fetchRequests(1);
  // 默认展开第一层 host 节点
  collectExpandableKeys(treeNodes.value, expandedKeys);
  window.addEventListener('packetmind:request-list-command', handleExternalCommand as EventListener);
  document.addEventListener('click', handleGlobalDismiss);
  document.addEventListener('contextmenu', handleGlobalDismiss);
  document.addEventListener('keydown', handleKeydown);
  window.addEventListener('scroll', handleGlobalDismiss, true);
});

onUnmounted(() => {
  window.removeEventListener('packetmind:request-list-command', handleExternalCommand as EventListener);
  document.removeEventListener('click', handleGlobalDismiss);
  document.removeEventListener('contextmenu', handleGlobalDismiss);
  document.removeEventListener('keydown', handleKeydown);
  window.removeEventListener('scroll', handleGlobalDismiss, true);
});

const onSearch = (value: string) => {
  searchKeyword.value = value;
};

const requestSort = (a: Request, b: Request) => {
  const aTime = a.created_at || '';
  const bTime = b.created_at || '';
  return bTime.localeCompare(aTime);
};

const splitPathSegments = (req: Request) => {
  const rawPath = (req.path || '/').replace(/^\/+|\/+$/g, '');
  const segments = rawPath ? rawPath.split('/').filter(Boolean) : [];
  const query = req.query_string ? `?${req.query_string}` : '';

  if (!segments.length) {
    return [query || '/'];
  }

  if (query) {
    segments[segments.length - 1] = segments[segments.length - 1] + query;
  }

  return segments;
};

const getRequestIconType = (req: Request) => {
  if (isRequestPending(req)) return 'pending';
  if (req.error || !req.status_code || req.status_code === 0) return 'error';

  const contentType = `${req.content_type || ''} ${req.resp_content_type || ''}`.toLowerCase();
  const url = req.url.toLowerCase();
  const path = (req.path || '').toLowerCase();

  if (req.is_websocket || req.scheme === 'wss' || req.scheme === 'ws' || url.startsWith('ws:') || url.startsWith('wss:')) return 'websocket';
  if (req.scheme === 'socks5') return 'socket';

  if (contentType.includes('html') || path.endsWith('.html')) return 'html';
  if (contentType.includes('javascript') || path.endsWith('.js')) return 'js';
  if (contentType.includes('css') || path.endsWith('.css')) return 'css';
  if (contentType.includes('json') || path.endsWith('.json')) return 'json';
  if (contentType.includes('image') || path.match(/\.(png|jpe?g|gif|svg|webp|ico)$/)) return 'image';
  if (contentType.includes('font') || path.match(/\.(woff2?|ttf|eot|otf)$/)) return 'font';
  if (contentType.includes('protobuf') || contentType.includes('grpc') || contentType.includes('x-protobuf')) return 'protobuf';

  if (req.scheme === 'https') return 'https';
  if (req.scheme === 'http') return 'http';
  return 'file';
};

const iconMeta = (req: Request) => {
  const time = req.duration ? `${req.duration}ms` : '-';
  const code = req.status_code || '-';
  return `${code} · ${time}`;
};

const treeNodeSort = (a: TreeNode, b: TreeNode) => {
  const aBranch = a.kind !== 'request';
  const bBranch = b.kind !== 'request';
  if (aBranch !== bBranch) {
    return aBranch ? -1 : 1;
  }

  const aTime = a.firstSeenAt || '';
  const bTime = b.firstSeenAt || '';
  if (aTime !== bTime) {
    return aTime.localeCompare(bTime);
  }

  return a.label.localeCompare(b.label);
};

const compressSingleChildFolders = (nodes: TreeNode[]): TreeNode[] => {
  return nodes.map((node) => {
    if (!node.children?.length || node.kind !== 'folder') return node;

    let current = { ...node, children: compressSingleChildFolders(node.children) };

    while (current.children && current.children.length === 1) {
      const child = current.children[0];
      if (child.kind === 'folder') {
        current = {
          ...current,
          key: child.key,
          label: `${current.label}/${child.label}`,
          children: child.children,
        };
      } else if (child.kind === 'request') {
        return {
          ...child,
          label: `${current.label}/${child.label}`,
        };
      } else {
        break;
      }
    }

    return current;
  });
};

const buildHostTree = (host: string, requests: Request[]): TreeNode => {
  const root: TreeNode = {
    key: `host:${host}`,
    kind: 'host',
    label: host,
    host,
    meta: `${requests.length}`,
    iconType: requests.some((req) => req.scheme === 'https') ? 'https' : 'http',
    children: [],
  };

  requests.sort(requestSort);

  for (const req of requests) {
    const segments = splitPathSegments(req);
    let children = root.children!;
    let parentKey = root.key;

    segments.forEach((segment, index) => {
      const isLeaf = index === segments.length - 1;
      const segmentKey = `${parentKey}/${segment}`;

      if (isLeaf) {
        children.push({
          key: `${segmentKey}#${req.id}`,
          kind: 'request',
          label: segment,
          host,
          request: req,
          meta: iconMeta(req),
          iconType: getRequestIconType(req),
          statusCode: req.status_code,
          firstSeenAt: req.created_at,
        });
        children.sort(treeNodeSort);
        return;
      }

      let folder = children.find((child) => child.kind === 'folder' && child.label === segment);
      if (!folder) {
        folder = {
          key: segmentKey,
          kind: 'folder',
          label: segment,
          host,
          children: [],
          iconType: 'folder',
          firstSeenAt: req.created_at,
        };
        children.push(folder);
        children.sort(treeNodeSort);
      }

      if (!folder.firstSeenAt || ((req.created_at || '') && (req.created_at || '') < folder.firstSeenAt)) {
        folder.firstSeenAt = req.created_at;
      }

      children = folder.children!;
      parentKey = folder.key;
    });
  }

  root.children = compressSingleChildFolders(root.children || []).sort(treeNodeSort);
  return root;
};

const treeNodes = computed<TreeNode[]>(() => {
  const groups = new Map<string, Request[]>();

  for (const req of store.requests) {
    const host = req.host || 'unknown';
    if (!groups.has(host)) groups.set(host, []);
    groups.get(host)!.push(req);
  }

  return Array.from(groups.entries())
    .map(([host, requests]) => buildHostTree(host, requests))
    .sort(treeNodeSort);
});

const collectExpandableKeys = (nodes: TreeNode[], target: Set<string>) => {
  for (const node of nodes) {
    if (node.children?.length) {
      target.add(node.key);
      collectExpandableKeys(node.children, target);
    }
  }
};

const toggleNode = (key: string) => {
  if (expandedKeys.has(key)) expandedKeys.delete(key);
  else expandedKeys.add(key);
};

const expandAll = () => {
  expandedKeys.clear();
  collectExpandableKeys(treeNodes.value, expandedKeys);
};

const collapseAll = () => {
  expandedKeys.clear();
};

const showContextMenu = (event: MouseEvent, node: TreeNode) => {
  event.stopPropagation();
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    node,
  };
};

const hideContextMenu = () => {
  contextMenu.value.visible = false;
};

const handleGlobalDismiss = () => {
  if (contextMenu.value.visible) {
    hideContextMenu();
  }
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && contextMenu.value.visible) {
    hideContextMenu();
  }
};

const handleExternalCommand = (event: Event) => {
  const customEvent = event as CustomEvent<{ command?: string }>;
  if (customEvent.detail?.command === 'expand-all') {
    expandAll();
    return;
  }
  if (customEvent.detail?.command === 'collapse-all') {
    collapseAll();
    return;
  }
  if (customEvent.detail?.command === 'focus-search') {
    searchInputRef.value?.focus();
    return;
  }
};

const handleCopyHost = async () => {
  const label = contextMenu.value.node?.host || contextMenu.value.node?.label || '';
  hideContextMenu();
  if (!label) return;

  const ok = await copyToClipboard(label);
  if (ok) message.success('Host copied');
  else message.error('Failed to copy host');
};

const handleCopyUrl = async () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;

  const ok = await copyToClipboard(request.url);
  if (ok) message.success('URL copied');
  else message.error('Failed to copy URL');
};

const handleCopyCurl = async () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;

  try {
    const curl = await requestApi.export(request.session_id, request.id, 'curl');
    const ok = await copyToClipboard(curl);
    if (!ok) {
      throw new Error('clipboard');
    }
    message.success('cURL copied');
  } catch {
    message.error('Failed to copy cURL');
  }
};

const handleCopyResponseBody = async () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request?.resp_body) return;

  const responseText = getResponseBodyText(request);
  if (responseText === null) {
    message.warning('Binary response body cannot be copied as text');
    return;
  }

  const ok = await copyToClipboard(responseText);
  if (ok) message.success('Response body copied');
  else message.error('Failed to copy response body');
};

const handleSaveResponse = () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request?.resp_body) return;

  try {
    const bytes = getResponseBodyBytes(request);
    const blob = new Blob([bytes], { type: request.resp_content_type || 'application/octet-stream' });
    triggerDownload(blob, inferResponseFilename(request));
    message.success('Response saved');
  } catch {
    message.error('Failed to save response');
  }
};

const handleRepeatRequest = async () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;

  try {
    await requestApi.replay(request.session_id, request.id);
    message.success('Replaying request...');
  } catch {
    message.error('Failed to repeat request');
  }
};

const handleComposeRequest = () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;
  emit('composeRequest', request.id);
};

const handleAnalyzeRequest = () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;
  emit('analyzeRequest', request.id);
};

const handleDeleteRequest = async () => {
  const request = contextMenuRequest.value;
  hideContextMenu();
  if (!request) return;

  try {
    const response = await requestApi.delete(request.session_id, request.id);
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to delete request');
    }

    if (store.selectedId === request.id) {
      store.selectRequest(null);
    }
    await store.fetchRequests(1);
    message.success('Request deleted');
  } catch {
    message.error('Failed to delete request');
  }
};

const handleExpandAllFromMenu = () => {
  hideContextMenu();
  expandAll();
  emit('expandAll');
};

const handleCollapseAllFromMenu = () => {
  hideContextMenu();
  collapseAll();
  emit('collapseAll');
};

const protocolIconComponent = (iconType?: string) => {
  switch (iconType) {
    case 'error':
      return CloseCircleFilled;
    case 'pending':
      return LoadingOutlined;
    case 'folder':
      return FolderFilled;
    case 'https':
      return LockOutlined;
    case 'http':
      return GlobalOutlined;
    case 'websocket':
      return SwapOutlined;
    case 'protobuf':
      return CodeOutlined;
    case 'socket':
      return ThunderboltOutlined;
    case 'host':
      return ApiOutlined;
    case 'html':
      return Html5Outlined;
    case 'js':
      return CodeOutlined;
    case 'css':
      return BgColorsOutlined;
    case 'json':
      return ProfileOutlined;
    case 'image':
      return PictureOutlined;
    case 'font':
      return FontSizeOutlined;
    default:
      return FileTextOutlined;
  }
};

const protocolIconClass = (iconType?: string) => {
  switch (iconType) {
    case 'error':
      return 'iconError';
    case 'pending':
      return 'iconPending';
    case 'https':
      return 'iconHttps';
    case 'http':
      return 'iconHttp';
    case 'websocket':
      return 'iconWebsocket';
    case 'protobuf':
      return 'iconProtobuf';
    case 'socket':
      return 'iconSocket';
    case 'folder':
      return 'iconFolder';
    case 'html':
      return 'iconHtml';
    case 'js':
      return 'iconJs';
    case 'css':
      return 'iconCss';
    case 'json':
      return 'iconJson';
    case 'image':
      return 'iconImage';
    case 'font':
      return 'iconFont';
    default:
      return 'iconDefault';
  }
};

const RequestTreeNode = defineComponent({
  name: 'RequestTreeNode',
  props: {
    node: { type: Object as () => TreeNode, required: true },
    expandedKeys: { type: Object as () => Set<string>, required: true },
    recentNodeKeys: { type: Object as () => Set<string>, required: true },
    selectedId: { type: String, default: null },
    onContextmenu: Function as PropType<(event: MouseEvent, node: TreeNode) => void>,
  },
  emits: ['toggle', 'select'],
  setup(props, { emit }) {
    const isExpanded = computed(() => props.expandedKeys.has(props.node.key));
    const hasChildren = computed(() => Boolean(props.node.children?.length));
    const isSelected = computed(() => props.node.request?.id === props.selectedId);
    const isRecent = computed(() => props.recentNodeKeys.has(props.node.key));
    const requestTime = computed(() => props.node.request?.created_at ? dayjs(props.node.request.created_at).format('HH:mm:ss') : '');
    const isPending = computed(() => props.node.request ? isRequestPending(props.node.request) : false);
    const statusClass = computed(() => {
      if (isPending.value) return styles.statusPending;
      const status = props.node.statusCode;
      if (status === 0 || status === undefined) return styles.statusFailed;
      if (status >= 500) return styles.statusError;
      if (status >= 400) return styles.statusWarn;
      if (status >= 300) return styles.statusRedirect;
      if (status >= 200) return styles.statusSuccess;
      return styles.statusNeutral;
    });

    const onClick = () => {
      if (props.node.request) {
        emit('select', props.node.request.id);
      } else if (hasChildren.value) {
        emit('toggle', props.node.key);
      }
    };

    return () => h('div', { class: styles.treeNode }, [
      h('div', {
        class: [
          styles.treeRow,
          hasChildren.value ? styles.treeBranch : '',
          isSelected.value ? styles.treeRowSelected : '',
          isRecent.value ? styles.treeRowRecent : '',
          isPending.value ? styles.treeRowPending : '',
          props.node.request ? styles.treeLeaf : '',
          props.node.request && !isPending.value && (props.node.request.error || !props.node.statusCode || props.node.statusCode === 0) ? styles.treeRowFailed : '',
        ],
        onClick,
        onContextmenu: (event: MouseEvent) => {
          event.preventDefault();
          event.stopPropagation();
          props.onContextmenu?.(event, props.node);
        },
      }, [
        h('span', { class: styles.expandSlot }, [
          hasChildren.value
            ? h(isExpanded.value ? DownOutlined : RightOutlined, { class: styles.expandGlyph })
            : null,
        ]),
        h(protocolIconComponent(props.node.kind === 'host' ? 'host' : props.node.iconType), {
          class: [styles.protocolIcon, styles[protocolIconClass(props.node.kind === 'host' ? 'host' : props.node.iconType)]],
        }),
        h('span', { class: styles.treeLabel, title: props.node.label }, props.node.label),
        props.node.request
          ? h('span', { class: styles.requestMeta }, [
              h('span', { class: styles.methodText }, props.node.request.method),
              isPending.value
                ? h('span', { class: [styles.metaText, statusClass.value] }, [
                    h('span', { class: styles.pendingSpinner }),
                    h('span', 'Pending'),
                  ])
                : h('span', { class: [styles.metaText, statusClass.value] }, props.node.meta || ''),
              h('span', { class: styles.metaText }, requestTime.value),
            ])
          : h('span', { class: styles.branchMeta }, props.node.kind === 'host' ? `${props.node.meta} requests` : ''),
      ]),
      hasChildren.value && isExpanded.value
        ? h('div', { class: styles.treeChildren }, props.node.children!.map((child) =>
            h(RequestTreeNode, {
              node: child,
              expandedKeys: props.expandedKeys,
              recentNodeKeys: props.recentNodeKeys,
              selectedId: props.selectedId,
              onContextmenu: props.onContextmenu,
              onToggle: (key: string) => emit('toggle', key),
              onSelect: (id: string) => emit('select', id),
            }),
          ))
        : null,
    ]);
  },
});
</script>

<style module>
.container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: #efefef;
  color: #1f1f1f;
}

.filterBar {
  padding: 3px 6px;
  background: linear-gradient(180deg, #ececec 0%, #e1e1e1 100%);
  border-bottom: 1px solid #ababab;
  flex-shrink: 0;
}

.filterColumn {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.searchInput {
  width: 100%;
  min-width: 0;
}

.searchInput :global(.ant-input) {
  font-size: 11px !important;
  height: 20px !important;
}

.searchInput :global(.ant-input-suffix) {
  font-size: 11px !important;
}

.searchInput :global(.ant-input-group-addon) {
  display: none;
}

.actionRow {
  display: flex;
  align-items: center;
  gap: 3px;
  flex-wrap: nowrap;
  min-width: 0;
  overflow: hidden;
}

.totalText {
  color: #6a6a6a;
  font-size: 10px;
  white-space: nowrap;
  flex: 0 0 auto;
  margin-right: 2px;
}

.actionBtn {
  flex: 0 0 auto;
  font-size: 9px !important;
  padding: 0 5px !important;
  height: 18px !important;
  line-height: 18px !important;
  border-radius: 2px !important;
}

.treeContainer {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.loading {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.treeRoot {
  padding: 4px 0 10px;
}

.treeNode {
  min-width: 0;
}

.treeChildren {
  margin-left: 18px;
}

.treeRow {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 24px;
  padding: 1px 8px 1px 4px;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.55s ease;
}

.treeRow:hover {
  background: #e2ecf8;
}

.treeRowSelected {
  background: #c7d9ef;
}

.treeRowRecent {
  animation: requestFlash 1.35s ease-out;
}

@keyframes requestFlash {
  0% {
    background: linear-gradient(90deg, #fff8d6 0%, #ffefb1 72%, #ffe08a 100%);
  }
  35% {
    background: linear-gradient(90deg, #fff5ce 0%, #ffebb0 72%, #ffd879 100%);
  }
  100% {
    background: transparent;
  }
}

.treeBranch {
  color: #202020;
}

.treeLeaf {
  color: #1c1c1c;
}

.treeRowFailed {
  color: #8c8c8c;
}

.treeRowFailed .treeLabel {
  color: #8c8c8c;
}

.treeRowFailed .methodText {
  color: #8c8c8c;
}

.treeRowPending {
  color: #5a8ab5;
}

.treeRowPending .treeLabel {
  color: #5a8ab5;
}

.treeRowPending .methodText {
  color: #5a8ab5;
}

.expandSlot {
  width: 12px;
  display: inline-flex;
  justify-content: center;
  color: #6a6a6a;
  font-size: 10px;
  flex-shrink: 0;
}

.expandGlyph {
  font-size: 10px;
}

.protocolIcon {
  font-size: 14px;
  flex-shrink: 0;
}

.iconFolder {
  color: #7cb9e6;
}

.iconError {
  color: #d32f2f;
}

.iconPending {
  color: #5a8ab5;
  animation: requestPendingPulse 1.25s ease-in-out infinite;
}

.iconHttps {
  color: #3b88d6;
}

.iconHttp {
  color: #69a7e8;
}

.iconWebsocket {
  color: #3fa36b;
}

.iconProtobuf {
  color: #8b67d6;
}

.iconSocket {
  color: #4f88ff;
}

.iconHtml {
  color: #e34f26;
}

.iconJs {
  color: #f7df1e;
}

.iconCss {
  color: #1572b6;
}

.iconJson {
  color: #8b8b8b;
}

.iconImage {
  color: #4caf50;
}

.iconFont {
  color: #ff9800;
}

.iconDefault {
  color: #8d8d8d;
}

.treeLabel {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.requestMeta,
.branchMeta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  color: #767676;
  font-size: 11px;
  white-space: nowrap;
}

.methodText {
  color: #4f4f4f;
  min-width: 48px;
  text-align: right;
}

.metaText {
  color: #767676;
}

.pendingSpinner {
  width: 10px;
  height: 10px;
  border: 1.5px solid rgba(90, 138, 181, 0.28);
  border-top-color: #5a8ab5;
  border-radius: 50%;
  display: inline-block;
  margin-right: 5px;
  vertical-align: -1px;
  animation: requestPendingSpin 0.8s linear infinite;
}

.statusPending {
  color: #5a8ab5;
}

.statusSuccess {
  color: #2f8f46;
}

.statusRedirect {
  color: #a67d19;
}

.statusWarn {
  color: #b96a20;
}

.statusError {
  color: #b04444;
}

.statusFailed {
  color: #d32f2f;
  font-style: italic;
}

.statusNeutral {
  color: #767676;
}

@keyframes requestPendingSpin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes requestPendingPulse {
  0%,
  100% {
    opacity: 0.45;
  }
  50% {
    opacity: 1;
  }
}
</style>

<style>
.request-context-menu {
  position: fixed;
  z-index: 10000;
  background: #f2f2f2;
  border: 1px solid #a0a0a0;
  box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);
  padding: 2px 0;
  min-width: 180px;
  font-size: 12px;
}

.request-menu-item {
  padding: 4px 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1f1f1f;
}

.request-menu-item:hover {
  background: #0066cc;
  color: #ffffff;
}

.request-menu-item:hover .anticon {
  color: #ffffff;
}

.request-menu-item-danger {
  color: #1f1f1f;
}

.request-menu-item-danger:hover {
  background: #d13438;
  color: #ffffff;
}

.request-menu-divider {
  height: 1px;
  background: #c0c0c0;
  margin: 2px 0;
}

.request-menu-disabled {
  color: #9c9c9c;
  cursor: not-allowed;
}

.request-menu-disabled:hover {
  background: transparent;
  color: #9c9c9c;
}

.request-menu-disabled:hover .anticon {
  color: #9c9c9c;
}
</style>
