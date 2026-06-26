<template>
  <div :class="[$style.layout, isMaximised ? $style.layoutMaximised : '']">
    <header :class="$style.header" @dblclick="toggleMaximise">
      <div :class="$style.dragRegion" data-application-menu-root="true">
      <div :class="$style.logo">
        <ApiOutlined /> PacketMind
      </div>
      <ApplicationMenu
        :proxyRunning="proxyRunning"
        :hasSelectedRequest="hasSelectedRequest"
        :hasActiveSession="hasActiveSession"
        :isMaximised="isMaximised"
        :externalProxyEnabled="appSettings.proxy.external_proxy.enabled"
        :throttlingEnabled="appSettings.proxy.throttling.enabled"
        :sslProxyingEnabled="appSettings.proxy.ssl_proxying.enabled"
        :accessControlEnabled="appSettings.proxy.access_control.enabled"
        :breakpointsEnabled="appSettings.proxy.breakpoints.enabled"
        :noCachingEnabled="appSettings.tools.no_caching"
        :blockCookiesEnabled="appSettings.tools.block_cookies"
        :hasProxySettings="true"
        :recentSessions="recentSessions"
        :hostFilterOptions="hostFilterOptions"
        :structureViewState="appSettings.window.structure_view"
        :statusBarState="statusBarVisible"
        :autoSaveState="appSettings.tools.auto_save_enabled"
        @action="handleMenuAction"
      />
      <div :class="$style.headerRight">
        <a-badge 
          :class="$style.noDrag"
          :status="proxyRunning ? 'success' : 'default'" 
          :text="proxyRunning ? 'Proxy Running' : 'Proxy Stopped'"
        />
        <a-button
          :class="[$style.noDrag, $style.proxyToggleBtn, proxyRunning ? $style.proxyStopBtn : $style.proxyStartBtn]"
          type="text"
          size="small"
          @click="toggleProxy"
        >
          {{ proxyRunning ? 'Stop' : 'Start' }} Proxy
        </a-button>
        <a-button
          :class="$style.noDrag"
          size="small"
          @click="clearRequests"
        >
          <template #icon><ClearOutlined /></template>
          Clear
        </a-button>
      </div>
      </div>
      <div v-if="showCustomWindowControls" :class="[$style.windowControls, isMaximised ? $style.windowControlsMaximised : '']">
        <button :class="[$style.windowButton, $style.noDrag]" type="button" @click="minimiseWindow" aria-label="Minimise window">
          <span :class="$style.windowIconBox">
            <span :class="$style.windowGlyphMinimise"></span>
          </span>
        </button>
        <button :class="[$style.windowButton, $style.noDrag]" type="button" @click="toggleMaximise" :aria-label="isMaximised ? 'Restore window' : 'Maximise window'">
          <span :class="$style.windowIconBox">
            <span v-if="!isMaximised" :class="$style.windowGlyphSquare"></span>
            <span v-else :class="$style.windowGlyphRestore">
              <span :class="$style.restoreBack"></span>
              <span :class="$style.restoreFront"></span>
            </span>
          </span>
        </button>
        <button :class="[$style.windowButton, $style.closeButton, $style.noDrag]" type="button" @click="closeWindow" aria-label="Close window">
          <span :class="$style.windowIconBox">
            <span :class="$style.windowGlyphClose"></span>
          </span>
        </button>
      </div>
    </header>
    
    <div :class="$style.mainLayout">
      <!-- Far Left Panel: Session List -->
      <div 
        :class="$style.sessionPanel"
        :style="{ width: sessionWidth + 'px' }"
      >
        <SessionList @sessionChange="handleSessionChange" />
      </div>
      
      <!-- Resize Handle 1 (Session) -->
      <div 
        :class="$style.resizeHandle"
        @mousedown="startResize('session', $event)"
      />
      
      <!-- Left Panel: Request List -->
      <div 
        :class="$style.leftPanel"
        :style="{ width: leftWidth + 'px' }"
      >
        <RequestList @composeRequest="handleComposeRequest" />
      </div>
      
      <!-- Resize Handle 2 (Left) -->
      <div 
        :class="$style.resizeHandle"
        @mousedown="startResize('left', $event)"
      />
      
      <!-- Center Panel: Request Detail -->
      <div :class="$style.centerPanel">
        <RequestDetail />
      </div>
      
      <!-- Resize Handle 3 (Right) -->
      <div 
        :class="$style.resizeHandle"
        @mousedown="startResize('right', $event)"
      />
      
      <!-- Right Panel: Agent Panel -->
      <div 
        :class="$style.rightPanel"
        :style="{ width: rightWidth + 'px' }"
      >
        <AgentPanel />
      </div>
    </div>

    <div v-if="statusBarVisible" :class="$style.statusBar">
      <span v-if="sessionStore.activeSession">Session: {{ sessionStore.activeSession.name }}</span>
      <span v-else>No Active Session</span>
      <span>|</span>
      <span>Requests: {{ requestStore.total }}</span>
      <span>|</span>
      <span>Proxy: {{ proxyRunning ? 'Running on ' + appSettings.proxy.listener.port : 'Stopped' }}</span>
    </div>

    <div v-if="showCustomWindowControls && !isMaximised" :class="$style.resizeGrip" aria-hidden="true"></div>
    <AppSettingsModal
      v-model:visible="settingsModalVisible"
      :mode="settingsModalMode"
      :settings="appSettings"
      @save="handleSettingsSave"
    />

    <ComposeModal
      v-model:visible="composeModalVisible"
      :request="composeRequestData"
    />

    <FindInSessionModal
      v-model:visible="findInSessionModalVisible"
    />

    <a-modal
      :open="localIPModalVisible"
      title="Local IP Addresses"
      :footer="null"
      :width="600"
      @cancel="localIPModalVisible = false"
    >
      <div :class="$style.localIPIntro">
        The available local network interfaces and their IP addresses are listed below. Excluded from this list are any loopback or link local addresses.
      </div>
      <div :class="$style.localIPPanel">
        <div :class="$style.localIPHeaderRow">
          <div :class="$style.localIPInterfaceCol">Network interface</div>
          <div :class="$style.localIPAddressCol">IP address</div>
        </div>
        <div v-if="localIPAddresses.length === 0" :class="$style.localIPEmpty">No active local addresses found.</div>
        <div v-else :class="$style.localIPBody">
          <div v-for="item in localIPAddresses" :key="`${item.interface_name}-${item.ip_address}`" :class="$style.localIPRow">
            <div :class="$style.localIPInterfaceCol">{{ item.interface_name }}</div>
            <div :class="$style.localIPAddressCol">{{ item.ip_address }}</div>
          </div>
        </div>
      </div>
    </a-modal>

    <UpdateModal
      :open="updateModalVisible"
      @close="updateModalVisible = false"
    />

    <a-modal
      :open="aboutModalVisible"
      :footer="null"
      :width="760"
      @cancel="aboutModalVisible = false"
      :bodyStyle="{ padding: '0' }"
      wrapClassName="packetmind-about-modal"
    >
      <div :class="$style.aboutHero">
        <div :class="$style.aboutOrb"></div>
        <div :class="$style.aboutLogoMark">
          <ApiOutlined />
        </div>
        <div :class="$style.aboutTitle">PacketMind</div>
        <div :class="$style.aboutSubtitle">Agent-Powered Proxy & Traffic Analysis</div>
      </div>
      <div :class="$style.aboutMetaBar">
        <span>v {{ appVersion }}</span>
        <span>Desktop Edition</span>
          <span>Built for request inspection and Agent-assisted tracing</span>
      </div>
      <div :class="$style.aboutBody">
        <div :class="$style.aboutSection">
          <div :class="$style.aboutSectionTitle">PacketMind</div>
          <p :class="$style.aboutParagraph">
             PacketMind combines desktop proxy capture with Agent-assisted analysis for HTTP, HTTPS, and SOCKS5 traffic.
          </p>
          <p :class="$style.aboutParagraph">
            Use it to inspect requests, trace values across sessions, analyze payloads, and understand complex network flows faster.
          </p>
        </div>
        <div :class="$style.aboutGrid">
          <div :class="$style.aboutCard">
            <div :class="$style.aboutCardLabel">Core Stack</div>
            <div :class="$style.aboutCardValue">Go · Wails · Vue 3 · TypeScript</div>
          </div>
          <div :class="$style.aboutCard">
            <div :class="$style.aboutCardLabel">Capabilities</div>
             <div :class="$style.aboutCardValue">HTTP / HTTPS / SOCKS5 Proxy · Agent</div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, h } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  ApiOutlined,
  ClearOutlined,
} from '@ant-design/icons-vue';
import ApplicationMenu from '@/components/ApplicationMenu.vue';
import RequestList from '@/components/RequestList.vue';
import RequestDetail from '@/components/RequestDetail.vue';
import AgentPanel from '@/components/AgentPanel.vue';
import SessionList from '@/components/SessionList.vue';
import AppSettingsModal from '@/components/AppSettingsModal.vue';
import ComposeModal from '@/components/ComposeModal.vue';
import FindInSessionModal from '@/components/FindInSessionModal.vue';
import UpdateModal from '@/components/UpdateModal.vue';
import { useWailsEvents } from '@/composables/useWailsEvents';
import { configApi, proxyApi, requestApi, sessionApi, updaterApi } from '@/api/wails';
import { copyToClipboard, isWailsRuntime } from '@/utils/wails';
import { useRequestStore } from '@/stores/requestStore';
import { useSessionStore } from '@/stores/sessionStore';
import { useAgentStore } from '@/stores/agentStore';
import type { AppSettings, Request } from '@/types';

const requestStore = useRequestStore();
const sessionStore = useSessionStore();
const agentStore = useAgentStore();
const proxyRunning = ref(false);
const isMaximised = ref(false);
const showCustomWindowControls = !/Mac|iPod|iPhone|iPad/.test(navigator.platform);
const hasSelectedRequest = computed(() => Boolean(requestStore.selectedRequest));
const hasActiveSession = computed(() => Boolean(sessionStore.activeSession));
const recentSessions = computed(() => {
  return sessionStore.sessions.slice(0, 5).map(s => ({
    id: s.id,
    name: s.name,
  }));
});
const hostFilterOptions = computed(() => {
  const hosts = new Set(requestStore.requests.map(r => r.host).filter(Boolean));
  return Array.from(hosts).map(h => ({
    host: h as string,
    checked: requestStore.filters.host === h,
  })).sort((a, b) => a.host.localeCompare(b.host));
});
const statusBarVisible = ref(true);
const appSettings = ref<AppSettings>({
  proxy: {
    listener: {
      http_enabled: true,
      port: 8888,
      https_enabled: true,
      mitm_enabled: true,
      socks5_enabled: true,
      auto_start_on_boot: false,
    },
    recording: {
      enabled: true,
      max_capture_body_size_mb: 5,
    },
    ssl_proxying: {
      enabled: false,
      include_hosts: [],
      exclude_hosts: [],
    },
    access_control: {
      enabled: false,
      allowed_clients: [],
    },
    external_proxy: {
      enabled: false,
      scheme: 'http',
      host: '',
      port: 0,
      username: '',
      password: '',
      password_configured: false,
      bypass_hosts: ['localhost', '127.0.0.1', '::1'],
    },
    throttling: {
      enabled: false,
      latency_ms: 0,
      downstream_kbps: 0,
      upstream_kbps: 0,
    },
    breakpoints: {
      enabled: false,
      request_matchers: [],
      response_matchers: [],
    },
    reverse_proxy: {
      enabled: false,
      rules: [],
    },
    port_forwarding: {
      enabled: false,
      rules: [],
    },
    web_interface: {
      enabled: false,
      port: 8889,
    },
  },
  tools: {
    no_caching: false,
    block_cookies: false,
    map_remote_enabled: false,
    map_local_enabled: false,
    rewrite_enabled: false,
    block_list_enabled: false,
    dns_spoofing: false,
    mirror_enabled: false,
    auto_save_enabled: false,
    client_process: false,
  },
  window: {
    structure_view: true,
    use_dark_theme: false,
  },
});
const settingsModalVisible = ref(false);
const settingsModalMode = ref<'proxy' | 'recording' | 'ssl' | 'access-control' | 'external-proxy' | 'throttling' | 'breakpoints' | 'reverse-proxy' | 'port-forwarding' | 'web-interface' | 'preferences'>('proxy');
const localIPModalVisible = ref(false);
const localIPAddresses = ref<Array<{ interface_name: string; ip_address: string }>>([]);
const aboutModalVisible = ref(false);
const composeModalVisible = ref(false);
const findInSessionModalVisible = ref(false);
const composeRequestData = ref<Request | null>(null);
const updateModalVisible = ref(false);
const appVersion = ref('1.0.0');

const loadSettings = async () => {
  try {
    const response = await configApi.getSettings();
    if (response.code === 0 && response.data) {
      appSettings.value = response.data;
    }
  } catch (error) {
    console.error('Failed to load desktop settings:', error);
  }
};

const persistSettings = async (next: AppSettings, successMessage?: string) => {
  const previous = appSettings.value;
  appSettings.value = next;
  try {
    const response = await configApi.updateSettings(next);
    if (response.code !== 0 || !response.data) {
      throw new Error(response.message || 'Failed to update desktop settings');
    }
    appSettings.value = response.data;
    if (successMessage) {
      message.success(successMessage);
    }
    return response.data;
  } catch (error) {
    appSettings.value = previous;
    throw error;
  }
};

const loadWindowRuntime = async () => {
  if (!isWailsRuntime()) {
    return null;
  }
  return await import('../wailsjs/runtime/runtime');
};

// Panel widths
const MIN_LEFT_WIDTH = 200;
const MAX_LEFT_WIDTH = 600;
const MIN_RIGHT_WIDTH = 300;
const MAX_RIGHT_WIDTH = 600;
const MIN_SESSION_WIDTH = 140;
const MAX_SESSION_WIDTH = 280;

const leftWidth = ref(400);
const rightWidth = ref(400);
const sessionWidth = ref(180);

// Load saved widths from localStorage
onMounted(() => {
  const savedLeft = localStorage.getItem('panel_left_width');
  const savedRight = localStorage.getItem('panel_right_width');
  const savedSession = localStorage.getItem('panel_session_width');
  
  if (savedLeft) {
    const width = parseInt(savedLeft, 10);
    if (width >= MIN_LEFT_WIDTH && width <= MAX_LEFT_WIDTH) {
      leftWidth.value = width;
    }
  }
  
  if (savedRight) {
    const width = parseInt(savedRight, 10);
    if (width >= MIN_RIGHT_WIDTH && width <= MAX_RIGHT_WIDTH) {
      rightWidth.value = width;
    }
  }

  if (savedSession) {
    const width = parseInt(savedSession, 10);
    if (width >= MIN_SESSION_WIDTH && width <= MAX_SESSION_WIDTH) {
      sessionWidth.value = width;
    }
  }
  
  checkProxyStatus();
  loadSettings();
  initializeActiveSession();
  syncWindowState();
  
  updaterApi.getVersion().then(res => {
    if (res.code === 0 && res.data) {
      appVersion.value = res.data.version;
    }
  });
});

// Resize logic
type ResizeType = 'left' | 'right' | 'session';
let currentResize: ResizeType | null = null;
let startX = 0;
let startWidth = 0;

const startResize = (type: ResizeType, event: MouseEvent) => {
  currentResize = type;
  startX = event.clientX;
  startWidth = type === 'left' ? leftWidth.value : type === 'right' ? rightWidth.value : sessionWidth.value;
  
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', stopResize);
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
};

const onMouseMove = (event: MouseEvent) => {
  if (!currentResize) return;
  
  const deltaX = event.clientX - startX;
  
  if (currentResize === 'left') {
    // Left panel: dragging right increases width
    const newWidth = startWidth + deltaX;
    leftWidth.value = Math.min(MAX_LEFT_WIDTH, Math.max(MIN_LEFT_WIDTH, newWidth));
  } else if (currentResize === 'right') {
    // Right panel: dragging left increases width
    const newWidth = startWidth - deltaX;
    rightWidth.value = Math.min(MAX_RIGHT_WIDTH, Math.max(MIN_RIGHT_WIDTH, newWidth));
  } else {
    // Session panel (leftmost): dragging right increases width
    const newWidth = startWidth + deltaX;
    sessionWidth.value = Math.min(MAX_SESSION_WIDTH, Math.max(MIN_SESSION_WIDTH, newWidth));
  }
};

const stopResize = () => {
  if (currentResize === 'left') {
    localStorage.setItem('panel_left_width', leftWidth.value.toString());
  } else if (currentResize === 'right') {
    localStorage.setItem('panel_right_width', rightWidth.value.toString());
  } else if (currentResize === 'session') {
    localStorage.setItem('panel_session_width', sessionWidth.value.toString());
  }
  
  currentResize = null;
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mouseup', stopResize);
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
};

onUnmounted(() => {
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mouseup', stopResize);
});

const syncWindowState = async () => {
  try {
    const runtime = await loadWindowRuntime();
    isMaximised.value = runtime ? await runtime.WindowIsMaximised() : false;
  } catch (error) {
    isMaximised.value = false;
  }
};

const minimiseWindow = async () => {
  const runtime = await loadWindowRuntime();
  runtime?.WindowMinimise();
};

const toggleMaximise = async () => {
  const runtime = await loadWindowRuntime();
  runtime?.WindowToggleMaximise();
  window.setTimeout(() => {
    void syncWindowState();
  }, 50);
};

const closeWindow = async () => {
  const runtime = await loadWindowRuntime();
  runtime?.Quit();
};

const { subscribeToSession } = useWailsEvents();

const initializeActiveSession = async () => {
  await sessionStore.fetchSessions();
  if (sessionStore.sessions.length === 0) {
    const created = await sessionStore.createSession('Session 1');
    if (created) {
      await sessionStore.activateSession(created.id);
      await sessionStore.fetchSessions();
    }
  }

  if (!sessionStore.activeSession && sessionStore.sessions.length > 0) {
    await sessionStore.activateSession(sessionStore.sessions[0].id);
  }

  const active = sessionStore.activeSession;
  if (!active) {
    return;
  }
  sessionStore.selectSession(active.id);
  requestStore.filters = { ...requestStore.filters, session_id: active.id };
  await requestStore.fetchRequests(1);
  subscribeToSession(active.id);
  await agentStore.bindSession(active.id);
};

const handleSessionChange = async (sessionId: string) => {
  sessionStore.selectSession(sessionId);
  requestStore.clearRequests();
  requestStore.filters = { ...requestStore.filters, session_id: sessionId };
  await requestStore.fetchRequests(1);
  subscribeToSession(sessionId);
  await agentStore.switchSession(sessionId);
};

const checkProxyStatus = async () => {
  try {
    const response = await proxyApi.status();
    if (response.code === 0) {
      proxyRunning.value = response.data?.running || false;
    }
  } catch (error) {
    console.error('Failed to check proxy status:', error);
  }
};

const toggleProxy = async () => {
  try {
    if (proxyRunning.value) {
      await proxyApi.stop();
      proxyRunning.value = false;
      message.success('Proxy stopped');
    } else {
      await proxyApi.start();
      proxyRunning.value = true;
      message.success(`Proxy started on port ${appSettings.value.proxy.listener.port}`);
    }
  } catch (error) {
    message.error('Failed to toggle proxy');
  }
};

const clearRequests = async () => {
  try {
    await requestApi.clear(sessionStore.activeSession?.id);
    requestStore.clearRequests();
    if (sessionStore.activeSession?.id) {
      await requestStore.fetchRequests(1);
    }
    message.success('Requests cleared');
  } catch (error) {
    message.error('Failed to clear requests');
  }
};

const toggleRecording = async () => {
  try {
    await persistSettings({
      ...appSettings.value,
      proxy: {
        ...appSettings.value.proxy,
        recording: {
          ...appSettings.value.proxy.recording,
          enabled: !appSettings.value.proxy.recording.enabled,
        },
      },
    }, appSettings.value.proxy.recording.enabled ? 'Recording disabled' : 'Recording enabled');
  } catch (error) {
    message.error('Failed to update recording setting');
  }
};

const toggleDesktopBoolean = async (
  updater: (current: AppSettings) => AppSettings,
  successMessage: string,
  errorMessage: string,
) => {
  try {
    await persistSettings(updater(appSettings.value), successMessage);
  } catch (error) {
    message.error(errorMessage);
  }
};

const createSessionFromMenu = async () => {
  try {
    const name = `Session ${sessionStore.sessions.length + 1}`;
    const session = await sessionStore.createSession(name);
    if (!session) {
      message.error('Failed to create session');
      return;
    }
    await sessionStore.activateSession(session.id);
    await handleSessionChange(session.id);
    message.success('Session created');
  } catch (error) {
    message.error('Failed to create session');
  }
};

const refreshRequestsFromMenu = async () => {
  try {
    await requestStore.fetchRequests(1);
    message.success('Requests refreshed');
  } catch (error) {
    message.error('Failed to refresh requests');
  }
};

const dispatchRequestListCommand = (command: 'expand-all' | 'collapse-all' | 'focus-search') => {
  window.dispatchEvent(new CustomEvent('packetmind:request-list-command', { detail: { command } }));
};

const exportSelectedRequestToClipboard = async (format: 'curl' | 'har') => {
  const selectedRequest = requestStore.selectedRequest;
  if (!selectedRequest) {
    message.warning('Please select a request first');
    return;
  }

  try {
    const exported = await requestApi.export(selectedRequest.session_id, selectedRequest.id, format);
    const copied = await copyToClipboard(exported);
    if (!copied) {
      message.error('Failed to copy export result');
      return;
    }
    message.success(format === 'curl' ? 'cURL copied to clipboard' : 'HAR copied to clipboard');
  } catch (error) {
    message.error('Failed to export selected request');
  }
};

const replaySelectedRequest = async () => {
  const selectedRequest = requestStore.selectedRequest;
  if (!selectedRequest) {
    message.warning('Please select a request first');
    return;
  }

  try {
    const response = await requestApi.replay(selectedRequest.session_id, selectedRequest.id);
    if (response.code !== 0 || !response.data) {
      message.error(response.message || 'Failed to replay request');
      return;
    }
    await requestStore.fetchRequests(1);
    requestStore.selectRequest(response.data.id);
    message.success('Request replayed');
  } catch (error) {
    message.error('Failed to replay request');
  }
};

const deleteSelectedRequest = async () => {
  const selectedRequest = requestStore.selectedRequest;
  if (!selectedRequest) {
    message.warning('Please select a request first');
    return;
  }

  try {
    const response = await requestApi.delete(selectedRequest.session_id, selectedRequest.id);
    if (response.code !== 0) {
      message.error(response.message || 'Failed to delete request');
      return;
    }
    requestStore.selectRequest(null);
    await requestStore.fetchRequests(1);
    message.success('Request deleted');
  } catch (error) {
    message.error('Failed to delete request');
  }
};

const handleComposeRequest = (id: string) => {
  const request = requestStore.requests.find((item) => item.id === id) || null;
  composeRequestData.value = request;
  composeModalVisible.value = true;
};

const showAbout = () => {
  aboutModalVisible.value = true;
};

const downloadBase64File = (filename: string, contentBase64: string, mimeType = 'application/octet-stream') => {
  const link = document.createElement('a');
  link.href = `data:${mimeType};base64,${contentBase64}`;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

const openImportDialog = () => new Promise<File | null>((resolve) => {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.json';
  input.onchange = () => resolve(input.files?.[0] || null);
  input.click();
});

const readFileAsBase64 = (file: File) => new Promise<string>((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = () => {
    const result = String(reader.result || '');
    const base64 = result.includes(',') ? result.split(',')[1] : '';
    resolve(base64);
  };
  reader.onerror = () => reject(reader.error);
  reader.readAsDataURL(file);
});

const exportAllSessions = async () => {
  try {
    const response = await sessionApi.exportAll();
    if (response.code !== 0 || !response.data) {
      message.error(response.message || 'Failed to export sessions');
      return;
    }
    downloadBase64File(response.data.filename, response.data.content, 'application/json');
    message.success('Session export downloaded');
  } catch (error) {
    message.error('Failed to export sessions');
  }
};

const importSessions = async () => {
  try {
    const file = await openImportDialog();
    if (!file) {
      return;
    }
    const contentBase64 = await readFileAsBase64(file);
    const response = await sessionApi.importAll(contentBase64);
    if (response.code !== 0) {
      message.error(response.message || 'Failed to import sessions');
      return;
    }
    await sessionStore.fetchSessions();
    await initializeActiveSession();
    message.success('Sessions imported');
  } catch (error) {
    message.error('Failed to import sessions');
  }
};

const downloadRootCertificate = async () => {
  try {
    const response = await configApi.downloadCert();
    if (response.code !== 0 || !response.data) {
      message.error(response.message || 'Failed to download root certificate');
      return;
    }
    downloadBase64File(response.data.filename, response.data.content, 'application/x-x509-ca-cert');
    message.success('Root certificate downloaded');
  } catch (error) {
    message.error('Failed to download root certificate');
  }
};

const showLocalIPAddress = async () => {
  try {
    const response = await configApi.getLocalIPAddresses();
    if (response.code !== 0 || !response.data) {
      message.error(response.message || 'Failed to read local addresses');
      return;
    }
    localIPAddresses.value = response.data;
    localIPModalVisible.value = true;
  } catch (error) {
    message.error('Failed to read local addresses');
  }
};

const openSettingsModal = (mode: typeof settingsModalMode.value) => {
  settingsModalMode.value = mode;
  settingsModalVisible.value = true;
};

const handleSettingsSave = async (next: AppSettings) => {
  try {
    await persistSettings(next, `${settingsModalMode.value} settings saved`);
  } catch (error) {
    message.error('Failed to save settings');
  }
};

const handleMenuAction = async (action: string) => {
  if (action.startsWith('file:open-recent:')) {
    const id = action.split(':')[2];
    await sessionStore.activateSession(id);
    await handleSessionChange(id);
    return;
  }
  if (action.startsWith('view:toggle-host:')) {
    const host = action.substring('view:toggle-host:'.length);
    requestStore.setFilters({ host: requestStore.filters.host === host ? undefined : host });
    return;
  }

  switch (action) {
    case 'file:new-session':
      await createSessionFromMenu();
      return;
    case 'file:export-curl':
    case 'edit:copy-curl':
      await exportSelectedRequestToClipboard('curl');
      return;
    case 'edit:cut':
      document.execCommand('cut');
      return;
    case 'edit:copy':
      document.execCommand('copy');
      return;
    case 'edit:paste':
      document.execCommand('paste');
      return;
    case 'edit:select-all':
      document.execCommand('selectAll');
      return;
    case 'edit:delete':
      await deleteSelectedRequest();
      return;
    case 'file:export-har':
      await exportSelectedRequestToClipboard('har');
      return;
    case 'file:clear-session':
      await clearRequests();
      return;
    case 'file:clear-recent':
      message.info('Recent sessions list cannot be cleared (showing top 5).');
      return;
    case 'file:close':
    case 'file:close-all':
      requestStore.clearRequests();
      message.info('Session view closed');
      return;
    case 'file:export-session':
    case 'file:save-session':
    case 'file:save-session-as':
      await exportAllSessions();
      return;
    case 'file:import-session':
    case 'file:open-session':
      await importSessions();
      return;
    case 'file:exit':
      await closeWindow();
      return;
    case 'view:refresh-requests':
      await refreshRequestsFromMenu();
      return;
    case 'view:structure':
      await toggleDesktopBoolean(
        (current) => ({ ...current, window: { ...current.window, structure_view: true } }),
        'Structure view activated',
        'Failed to switch view',
      );
      return;
    case 'view:sequence':
      await toggleDesktopBoolean(
        (current) => ({ ...current, window: { ...current.window, structure_view: false } }),
        'Sequence view activated',
        'Failed to switch view',
      );
      return;
    case 'view:clear-focus':
      requestStore.setFilters({ host: undefined });
      return;
    case 'view:chart':
      Modal.info({
        title: 'Session Chart Summary',
        content: h('div', [
          h('p', `Active Session: ${sessionStore.activeSession?.name || 'None'}`),
          h('p', `Total Requests: ${requestStore.total}`),
          h('p', `Focused Host: ${requestStore.filters.host || 'All'}`)
        ])
      });
      return;
    case 'view:toggle-status-bar':
      statusBarVisible.value = !statusBarVisible.value;
      return;
    case 'edit:find':
      findInSessionModalVisible.value = true;
      return;
    case 'view:expand-all':
      dispatchRequestListCommand('expand-all');
      return;
    case 'view:collapse-all':
      dispatchRequestListCommand('collapse-all');
      return;
    case 'view:fullscreen': {
      const runtime = await loadWindowRuntime();
      if (runtime) {
        const fullscreen = await runtime.WindowIsFullscreen();
        if (fullscreen) {
          runtime.WindowUnfullscreen();
        } else {
          runtime.WindowFullscreen();
        }
      }
      return;
    }
    case 'proxy:toggle-recording':
      await toggleRecording();
      return;
    case 'proxy:toggle-throttling':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          proxy: {
            ...current.proxy,
            throttling: {
              ...current.proxy.throttling,
              enabled: !current.proxy.throttling.enabled,
            },
          },
        }),
        appSettings.value.proxy.throttling.enabled ? 'Throttling disabled' : 'Throttling enabled',
        'Failed to update throttling setting',
      );
      return;
    case 'proxy:toggle-ssl-proxying':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          proxy: {
            ...current.proxy,
            ssl_proxying: {
              ...current.proxy.ssl_proxying,
              enabled: !current.proxy.ssl_proxying.enabled,
            },
          },
        }),
        appSettings.value.proxy.ssl_proxying.enabled ? 'SSL proxying disabled' : 'SSL proxying enabled',
        'Failed to update SSL proxying setting',
      );
      return;
    case 'proxy:toggle-breakpoints':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          proxy: {
            ...current.proxy,
            breakpoints: {
              ...current.proxy.breakpoints,
              enabled: !current.proxy.breakpoints.enabled,
            },
          },
        }),
        appSettings.value.proxy.breakpoints.enabled ? 'Breakpoints disabled' : 'Breakpoints enabled',
        'Failed to update breakpoint setting',
      );
      return;
    case 'tools:toggle-no-caching':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          tools: {
            ...current.tools,
            no_caching: !current.tools.no_caching,
          },
        }),
        appSettings.value.tools.no_caching ? 'No Caching disabled' : 'No Caching enabled',
        'Failed to update No Caching setting',
      );
      return;
    case 'tools:toggle-block-cookies':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          tools: {
            ...current.tools,
            block_cookies: !current.tools.block_cookies,
          },
        }),
        appSettings.value.tools.block_cookies ? 'Block Cookies disabled' : 'Block Cookies enabled',
        'Failed to update Block Cookies setting',
      );
      return;
    case 'tools:toggle-auto-save':
      await toggleDesktopBoolean(
        (current) => ({
          ...current,
          tools: {
            ...current.tools,
            auto_save_enabled: !current.tools.auto_save_enabled,
          },
        }),
        appSettings.value.tools.auto_save_enabled ? 'Auto Save disabled' : 'Auto Save enabled',
        'Failed to update Auto Save setting',
      );
      return;
    case 'tools:auto-save-settings':
      openSettingsModal('preferences');
      return;
    case 'tools:repeat-request':
      await replaySelectedRequest();
      return;
    case 'tools:repeat-advanced':
      await replaySelectedRequest();
      message.info('Repeat advanced dispatched');
      return;
    case 'tools:edit-request':
      if (requestStore.selectedRequest) {
        Modal.info({
          title: 'Edit Request',
          content: `Editing request: ${requestStore.selectedRequest.method} ${requestStore.selectedRequest.url}`,
        });
      }
      return;
    case 'tools:validate':
      if (requestStore.selectedRequest) {
        Modal.success({
          title: 'Validation Result',
          content: 'Request syntax and headers are well-formed.',
        });
      }
      return;
    case 'proxy:settings':
      openSettingsModal('proxy');
      return;
    case 'proxy:ssl-settings':
      openSettingsModal('ssl');
      return;
    case 'proxy:access-control':
      openSettingsModal('access-control');
      return;
    case 'proxy:web-interface':
      openSettingsModal('web-interface');
      return;
    case 'proxy:external-proxy':
      openSettingsModal('external-proxy');
      return;
    case 'proxy:throttling-settings':
      openSettingsModal('throttling');
      return;
    case 'proxy:reverse-proxies':
      openSettingsModal('reverse-proxy');
      return;
    case 'proxy:port-forwarding':
      openSettingsModal('port-forwarding');
      return;
    case 'proxy:breakpoint-settings':
      openSettingsModal('breakpoints');
      return;
    case 'proxy:sys-enable':
      message.info('System proxy toggling is currently controlled via external OS settings');
      return;
    case 'proxy:sys-settings':
      message.info('Please configure system proxy manually in OS settings');
      return;
    case 'proxy:recording-settings':
      openSettingsModal('recording');
      return;
    case 'edit:preferences':
      openSettingsModal('preferences');
      return;
    case 'window:minimize':
      await minimiseWindow();
      return;
    case 'window:toggle-maximize':
      await toggleMaximise();
      return;
    case 'window:close':
      await closeWindow();
      return;
    case 'window:switch-session-1':
      if (sessionStore.sessions.length > 0) {
        const id = sessionStore.sessions[0].id;
        await sessionStore.activateSession(id);
        await handleSessionChange(id);
      }
      return;
    case 'help:about':
      agentStore.loadModels();
      showAbout();
      return;
    case 'help:install-cert':
      await downloadRootCertificate();
      return;
    case 'help:local-ip':
      await showLocalIPAddress();
      return;
    case 'help:check-updates':
      updateModalVisible.value = true;
      return;
    default:
      return;
  }
};
</script>

<style module>
.layout {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background-color: #d9d9d9;
  color: #1f1f1f;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  box-shadow: inset 0 0 0 1px #b8b8b8;
}

.layoutMaximised {
  border: none;
}

.header {
  display: flex;
  align-items: center;
  background: linear-gradient(180deg, #f4f4f4 0%, #e0e0e0 55%, #d6d6d6 100%);
  padding-left: 8px;
  padding-right: 4px;
  border-bottom: 1px solid #9f9f9f;
  height: 34px;
  flex-shrink: 0;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82), inset 0 -1px 0 rgba(0, 0, 0, 0.04);
  --wails-draggable: drag;
  user-select: none;
}

.dragRegion {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  flex: 1;
  min-width: 0;
  padding-right: 8px;
}

.logo {
  font-size: 12px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #333;
  letter-spacing: 0.01em;
  min-width: 0;
}

.headerRight {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  margin-left: auto;
}

.proxyToggleBtn {
  color: #fff !important;
  border-width: 1px !important;
  border-style: solid !important;
  padding: 0 15px !important;
  font-weight: 600;
}

.proxyStartBtn {
  background: linear-gradient(180deg, #52c41a 0%, #389e0d 100%) !important;
  border-color: #2f7d0c !important;
}

.proxyStartBtn:hover,
.proxyStartBtn:focus {
  background: linear-gradient(180deg, #5fd028 0%, #3ea611 100%) !important;
  border-color: #2f7d0c !important;
  color: #fff !important;
}

.proxyStartBtn:active {
  background: linear-gradient(180deg, #389e0d 0%, #2f7d0c 100%) !important;
}

.proxyStopBtn {
  background: linear-gradient(180deg, #ff4d4f 0%, #cf1322 100%) !important;
  border-color: #a8071a !important;
}

.proxyStopBtn:hover,
.proxyStopBtn:focus {
  background: linear-gradient(180deg, #ff6666 0%, #d91b2c 100%) !important;
  border-color: #a8071a !important;
  color: #fff !important;
}

.proxyStopBtn:active {
  background: linear-gradient(180deg, #cf1322 0%, #a8071a 100%) !important;
}

.noDrag {
  --wails-draggable: no-drag;
}

.windowControls {
  display: flex;
  align-items: stretch;
  height: 100%;
  margin-left: auto;
  --wails-draggable: no-drag;
}

.windowControlsMaximised {
  margin-right: 0;
}

.windowButton {
  width: 46px;
  height: 100%;
  border: 0;
  padding: 0;
  margin: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  color: #333;
  cursor: default;
  border-radius: 0;
  transition: background-color 0.1s ease;
}

.windowButton:hover {
  background: rgba(0, 0, 0, 0.08);
}

.windowButton:active {
  background: rgba(0, 0, 0, 0.15);
}

.closeButton:hover {
  background: #e81123;
  color: #ffffff;
}

.closeButton:active {
  background: #c50f1f;
}

.windowIconBox {
  position: relative;
  width: 10px;
  height: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: currentColor;
}

.windowGlyphMinimise {
  position: absolute;
  left: 0;
  right: 0;
  top: 5px;
  height: 1px;
  background: currentColor;
}

.windowGlyphSquare {
  position: absolute;
  width: 10px;
  height: 10px;
  border: 1px solid currentColor;
  box-sizing: border-box;
  top: 0;
  left: 0;
}

.windowGlyphRestore {
  position: relative;
  width: 10px;
  height: 10px;
}

.restoreBack,
.restoreFront {
  position: absolute;
  box-sizing: border-box;
  border: 1px solid currentColor;
  background: transparent;
}

.restoreBack {
  width: 8px;
  height: 8px;
  top: -1px;
  left: 2px;
}

.restoreFront {
  width: 8px;
  height: 8px;
  left: -1px;
  top: 2px;
  background: #e6e6e6;
}

.closeButton:hover .restoreFront,
.closeButton:hover .restoreBack {
  background: transparent;
}

.windowGlyphClose {
  position: absolute;
  inset: 0;
}

.windowGlyphClose::before,
.windowGlyphClose::after {
  content: '';
  position: absolute;
  left: 4px;
  top: -1px;
  width: 1px;
  height: 12px;
  background: currentColor;
  transform-origin: center;
}

.windowGlyphClose::before {
  transform: rotate(45deg);
}

.windowGlyphClose::after {
  transform: rotate(-45deg);
}

.resizeGrip {
  position: fixed;
  right: 1px;
  bottom: 1px;
  width: 14px;
  height: 14px;
  pointer-events: none;
  opacity: 0.6;
  background-image:
    linear-gradient(135deg, transparent 0 48%, rgba(100, 100, 100, 0.8) 48% 52%, transparent 52%),
    linear-gradient(135deg, transparent 0 62%, rgba(100, 100, 100, 0.7) 62% 66%, transparent 66%),
    linear-gradient(135deg, transparent 0 76%, rgba(100, 100, 100, 0.6) 76% 80%, transparent 80%);
}

.mainLayout {
  display: flex;
  flex: 1;
  overflow: hidden;
  min-height: 0;
  background-color: #d9d9d9;
}

.statusBar {
  height: 22px;
  background: #f0f0f0;
  border-top: 1px solid #ababab;
  display: flex;
  align-items: center;
  padding: 0 8px;
  font-size: 11px;
  color: #555;
  gap: 8px;
  flex-shrink: 0;
}

.leftPanel, .sessionPanel, .rightPanel {
  flex-shrink: 0;
  height: 100%;
  overflow: hidden;
  background: #f8f8f8;
  border-right: 1px solid #ababab;
}

.centerPanel {
  flex: 1 1 0;
  min-width: 200px;
  height: 100%;
  overflow: hidden;
  background: #fbfbfb;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #ababab;
  border-left: 1px solid #fdfdfd;
}

.resizeHandle {
  width: 4px;
  height: 100%;
  background-color: #d3d3d3;
  cursor: col-resize;
  flex-shrink: 0;
  transition: background-color 0.1s;
  position: relative;
  border-left: 1px solid #efefef;
  border-right: 1px solid #a8a8a8;
  z-index: 10;
}

.resizeHandle:hover, .resizeHandle:active {
  background-color: #c5c5c5;
  border-right-color: #8f8f8f;
}

.resizeHandle::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 1px;
  height: 16px;
  background-color: #9a9a9a;
  box-shadow: 2px 0 0 #9a9a9a;
}

.localIPIntro {
  margin-bottom: 12px;
  font-size: 13px;
  line-height: 1.5;
  color: #333;
}

.localIPPanel {
  border: 1px solid #9cb9d9;
  background: #fff;
  min-height: 268px;
}

.localIPHeaderRow,
.localIPRow {
  display: grid;
  grid-template-columns: 1.6fr 1fr;
}

.localIPHeaderRow {
  background: linear-gradient(180deg, #fbfbfb 0%, #efefef 100%);
  border-bottom: 1px solid #d4d4d4;
  font-size: 12px;
  font-weight: 600;
  color: #222;
}

.localIPRow {
  font-size: 13px;
  color: #222;
}

.localIPInterfaceCol,
.localIPAddressCol {
  padding: 7px 10px;
}

.localIPInterfaceCol {
  border-right: 1px solid #e3e3e3;
}

.localIPBody {
  min-height: 228px;
}

.localIPEmpty {
  padding: 16px 10px;
  font-size: 13px;
  color: #666;
}

.aboutHero {
  position: relative;
  overflow: hidden;
  padding: 40px 24px 28px;
  background: radial-gradient(circle at 20% 20%, rgba(255,255,255,0.18), transparent 30%),
    radial-gradient(circle at 80% 30%, rgba(255,255,255,0.14), transparent 24%),
    linear-gradient(135deg, #36c3cd 0%, #129db9 55%, #0f8aa8 100%);
  color: #fff;
  text-align: center;
}

.aboutOrb {
  position: absolute;
  inset: -40px;
  background:
    radial-gradient(circle at 50% 50%, rgba(255,255,255,0.14), transparent 38%),
    radial-gradient(circle at 50% 50%, transparent 48%, rgba(255,255,255,0.12) 49%, transparent 50%),
    radial-gradient(circle at 50% 50%, transparent 58%, rgba(255,255,255,0.1) 59%, transparent 60%);
  pointer-events: none;
}

.aboutLogoMark {
  position: relative;
  z-index: 1;
  width: 76px;
  height: 76px;
  margin: 0 auto 14px;
  border-radius: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, rgba(255,255,255,0.96), rgba(240,248,252,0.92));
  color: #0b95b4;
  font-size: 34px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
}

.aboutTitle {
  position: relative;
  z-index: 1;
  font-size: 28px;
  font-weight: 500;
  letter-spacing: 0.02em;
}

.aboutSubtitle {
  position: relative;
  z-index: 1;
  margin-top: 8px;
  font-size: 14px;
  opacity: 0.92;
}

.aboutMetaBar {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  padding: 14px 24px;
  background: linear-gradient(180deg, #fafafa 0%, #f2f2f2 100%);
  border-top: 1px solid rgba(255,255,255,0.55);
  border-bottom: 1px solid #dedede;
  font-size: 13px;
  color: #4f4f4f;
}

.aboutBody {
  padding: 20px 24px 24px;
  background: #fff;
}

.aboutSectionTitle {
  font-size: 15px;
  font-weight: 600;
  color: #1f1f1f;
  margin-bottom: 10px;
}

.aboutParagraph {
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.6;
  color: #444;
}

.aboutGrid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.aboutCard {
  border: 1px solid #dadada;
  background: linear-gradient(180deg, #fbfbfb 0%, #f6f6f6 100%);
  padding: 12px 14px;
}

.aboutCardLabel {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #6b6b6b;
  margin-bottom: 6px;
}

.aboutCardValue {
  font-size: 13px;
  color: #1f1f1f;
  line-height: 1.5;
}
</style>
