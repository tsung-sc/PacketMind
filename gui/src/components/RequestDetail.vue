<template>
  <div ref="containerEl" :class="[$style.container, isNarrow ? $style.narrowDetail : '']">
    <div v-if="!store.selectedRequest" :class="$style.empty">
      <a-empty description="Select a request to view details" />
    </div>

    <template v-else>
      <div :class="$style.headerBar">
        <div :class="$style.headerLine">
          <span :class="$style.headerText">{{ headerSummary }}</span>
        </div>

        <div :class="$style.topTabs">
          <button
            v-for="tab in topTabs"
            :key="tab.key"
            type="button"
            :class="[$style.topTab, { [$style.topTabActive]: activeTopTab === tab.key }]"
            @click="activeTopTab = tab.key"
          >
            {{ tab.label }}
          </button>
        </div>
      </div>

      <div v-if="activeTopTab === 'contents'" :class="$style.contentShell">
        <div :class="$style.requestSection" :style="{ height: requestHeight + 'px' }">
          <RequestSection
            :requestId="req.id"
            :method="req.method"
            :url="req.url"
            :path="req.path"
            :queryString="req.query_string"
            :httpVersion="req.http_version"
            :headers="req.headers"
            :body="req.body"
            :contentType="req.content_type"
            :cookies="req.cookies"
            :websocketFrames="req.websocket_frames"
            :isNarrow="isNarrow"
          />
        </div>

        <div :class="$style.divider" @mousedown="startResize">
          <div :class="$style.dividerLine"></div>
        </div>

        <div :class="$style.responseSection">
          <ResponseSection
            :requestId="req.id"
            :statusCode="req.status_code"
            :statusReason="req.status_reason"
            :httpVersion="req.http_version"
            :headers="req.resp_headers"
            :body="req.resp_body"
            :contentType="req.resp_content_type"
            :websocketFrames="req.websocket_frames"
            :isNarrow="isNarrow"
          />
        </div>
      </div>

      <div v-else-if="activeTopTab === 'overview'" :class="$style.overviewPanel">
        <div :class="$style.overviewContent">
          <div v-for="row in overviewRows" :key="row.key" :class="$style.overviewRow">
            <div :class="$style.overviewKey">{{ row.key }}</div>
            <div :class="[row.error ? $style.overviewValueError : $style.overviewValue]">{{ row.value }}</div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSection('connection')">
              <span :class="$style.expandIcon">{{ expandedSections.connection ? '▼' : '▶' }}</span>
              <span>Connection</span>
            </div>
            <div v-show="expandedSections.connection" :class="$style.sectionContent">
              <div v-for="row in connectionRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSection('timing')">
              <span :class="$style.expandIcon">{{ expandedSections.timing ? '▼' : '▶' }}</span>
              <span>Timing</span>
            </div>
            <div v-show="expandedSections.timing" :class="$style.sectionContent">
              <div v-for="row in timingRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSection('size')">
              <span :class="$style.expandIcon">{{ expandedSections.size ? '▼' : '▶' }}</span>
              <span>Size</span>
            </div>
            <div v-show="expandedSections.size" :class="$style.sectionContent">
              <div v-for="row in sizeRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeTopTab === 'ssl'" :class="$style.overviewPanel">
        <div :class="$style.overviewContent">
          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('protocol')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.protocol ? '▼' : '▶' }}</span>
              <span>Protocol</span>
            </div>
            <div v-show="expandedSSLSections.protocol" :class="$style.sectionContent">
              <div v-for="row in sslProtocolRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('session')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.session ? '▼' : '▶' }}</span>
              <span>Session Resumed</span>
            </div>
            <div v-show="expandedSSLSections.session" :class="$style.sectionContent">
              <div v-for="row in sslSessionRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('cipher')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.cipher ? '▼' : '▶' }}</span>
              <span>Cipher Suite</span>
            </div>
            <div v-show="expandedSSLSections.cipher" :class="$style.sectionContent">
              <div v-for="row in sslCipherRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('alpn')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.alpn ? '▼' : '▶' }}</span>
              <span>ALPN</span>
            </div>
            <div v-show="expandedSSLSections.alpn" :class="$style.sectionContent">
              <div v-for="row in sslALPNRows" :key="row.key" :class="$style.overviewRow">
                <div :class="$style.overviewKey">{{ row.key }}</div>
                <div :class="$style.overviewValue">{{ row.value }}</div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('certificates')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.certificates ? '▼' : '▶' }}</span>
              <span>Server Certificates</span>
            </div>
            <div v-show="expandedSSLSections.certificates" :class="$style.sectionContent">
              <div v-if="!sslCertificates.length" :class="$style.overviewRow">
                <div :class="$style.overviewKey">Certificates</div>
                <div :class="$style.overviewValue">-</div>
              </div>
              <div v-for="(cert, index) in sslCertificates" :key="`${cert.serial_number}-${index}`" :class="$style.nestedSection">
                <div :class="$style.nestedHeader" @click="toggleCertificateSection(index)">
                  <span :class="$style.expandIcon">{{ expandedCertificateSections[index] ? '▼' : '▶' }}</span>
                  <span>{{ cert.subject_common_name || cert.subject || `Certificate ${index + 1}` }}</span>
                </div>
                <div v-show="expandedCertificateSections[index]" :class="$style.sectionContent">
                  <div v-for="row in buildCertificateRows(cert)" :key="row.key" :class="$style.overviewRow">
                    <div :class="$style.overviewKey">{{ row.key }}</div>
                    <div :class="$style.overviewValue">{{ row.value }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div :class="$style.collapsibleSection">
            <div :class="$style.sectionHeader" @click="toggleSSLSection('extensions')">
              <span :class="$style.expandIcon">{{ expandedSSLSections.extensions ? '▼' : '▶' }}</span>
              <span>Extensions</span>
            </div>
            <div v-show="expandedSSLSections.extensions" :class="$style.sectionContent">
              <div :class="$style.nestedSection">
                <div :class="$style.nestedHeader" @click="toggleExtensionSection('server')">
                  <span :class="$style.expandIcon">{{ expandedExtensionSections.server ? '▼' : '▶' }}</span>
                  <span>Server</span>
                </div>
                <div v-show="expandedExtensionSections.server" :class="$style.sectionContent">
                  <div v-if="!sslServerExtensions.length" :class="$style.overviewRow">
                    <div :class="$style.overviewKey">Extensions</div>
                    <div :class="$style.overviewValue">-</div>
                  </div>
                  <div v-for="extension in sslServerExtensions" :key="`${extension.id}-${extension.name}`" :class="$style.overviewRow">
                    <div :class="$style.overviewKey">{{ extension.name }} ({{ extension.id }})</div>
                    <div :class="$style.overviewValue">{{ extension.value || '-' }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeTopTab === 'summary'" :class="$style.summaryPanel">
        <div :class="$style.summaryTableWrap">
          <table :class="$style.summaryTable">
            <thead>
              <tr>
                <th :class="$style.summaryColIndex">#</th>
                <th :class="$style.summaryColResource">Resource</th>
                <th :class="$style.summaryColHost">Host</th>
                <th :class="$style.summaryColCode">Code</th>
                <th :class="$style.summaryColMime">Mime Type</th>
                <th :class="$style.summaryColNumber">Header</th>
                <th :class="$style.summaryColNumber">Body</th>
                <th :class="$style.summaryColTime">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td :class="$style.summaryCellIndex">1</td>
                <td :class="$style.summaryCellText">{{ summaryResource }}</td>
                <td :class="$style.summaryCellText">{{ req.host || '-' }}</td>
                <td :class="$style.summaryCellNumber">{{ req.status_code || '-' }}</td>
                <td :class="$style.summaryCellText">{{ req.resp_content_type || req.content_type || '-' }}</td>
                <td :class="$style.summaryCellNumber">{{ summaryHeaderSize }}</td>
                <td :class="$style.summaryCellNumber">{{ summaryBodySize }}</td>
                <td :class="$style.summaryCellTime">{{ summaryTime }}</td>
              </tr>
              <tr>
                <td></td>
                <td :class="$style.summaryTotalLabel">Total</td>
                <td></td>
                <td></td>
                <td></td>
                <td :class="$style.summaryCellNumber">{{ summaryHeaderSize }}</td>
                <td :class="$style.summaryCellNumber">{{ summaryBodySize }}</td>
                <td :class="$style.summaryCellTime">{{ summaryTime }}</td>
              </tr>
              <tr>
                <td></td>
                <td :class="$style.summaryTotalLabel">Grand Total</td>
                <td></td>
                <td></td>
                <td></td>
                <td :class="$style.summaryCellNumber">{{ summaryHeaderSize }}</td>
                <td :class="$style.summaryCellNumber">{{ summaryBodySize }}</td>
                <td :class="$style.summaryCellTime">{{ summaryTime }}</td>
              </tr>
              <tr>
                <td></td>
                <td :class="$style.summaryTotalLabel">Duration</td>
                <td></td>
                <td></td>
                <td></td>
                <td></td>
                <td></td>
                <td :class="$style.summaryCellTime">{{ summaryTime }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else-if="activeTopTab === 'websocket'" :class="$style.websocketPanel">
        <div :class="$style.websocketTableWrap">
          <table :class="$style.websocketTable">
            <thead>
              <tr>
                <th :class="$style.wsColDirection">Direction</th>
                <th :class="$style.wsColType">Type</th>
                <th :class="$style.wsColSize">Size</th>
                <th :class="$style.wsColTime">Time</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!websocketFrames.length">
                <td colspan="5" :class="$style.websocketEmpty">No WebSocket frames captured</td>
              </tr>
              <tr v-for="frame in websocketFrames" :key="frame.id">
                <td :class="$style.websocketDirection">{{ frameDirectionLabel(frame.direction) }}</td>
                <td>{{ frame.frame_type }}</td>
                <td :class="$style.websocketNumber">{{ formatBytes(frame.payload_size) }}</td>
                <td :class="$style.websocketNumber">{{ formatFrameTime(frame.created_at) }}</td>
                <td :class="$style.websocketPayload">{{ frame.payload || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div v-else :class="$style.placeholderPanel">
        <div :class="$style.placeholderTitle">{{ currentTopTabLabel }}</div>
        <div :class="$style.placeholderText">该视图暂未实现，当前先提供 Contents 视图用于请求/响应分析。</div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from 'vue';
import dayjs from 'dayjs';
import { RightOutlined, DownOutlined } from '@ant-design/icons-vue';
import { useRequestStore } from '@/stores/requestStore';
import type { TLSCertificate, TLSExtension } from '@/types';
import RequestSection from './RequestSection.vue';
import ResponseSection from './ResponseSection.vue';
import { getWindowSize, onWindowResize } from '../utils/wails';
import { useNarrowContainer } from '@/composables/useNarrowContainer';

const store = useRequestStore();
const selectedRequest = computed(() => store.selectedRequest);
const req = computed(() => store.selectedRequest!);
const containerEl = ref<HTMLElement | null>(null);
const { isNarrow } = useNarrowContainer(containerEl, 400);

const topTabs = computed(() => {
  if (!selectedRequest.value) {
    return [
      { key: 'overview', label: 'Overview' },
      { key: 'contents', label: 'Contents' },
      { key: 'summary', label: 'Summary' },
      { key: 'chart', label: 'Chart' },
      { key: 'notes', label: 'Notes' },
    ];
  }
  const tabs = [
    { key: 'overview', label: 'Overview' },
    { key: 'contents', label: 'Contents' },
  ];
  if (req.value.scheme === 'https' || req.value.scheme === 'wss' || req.value.tls_version) {
    tabs.push({ key: 'ssl', label: 'SSL' });
  }
  if (req.value.is_websocket || websocketFrames.value.length > 0) {
    tabs.push({ key: 'websocket', label: 'WebSocket' });
  }
  tabs.push({ key: 'summary', label: 'Summary' });
  tabs.push({ key: 'chart', label: 'Chart' });
  tabs.push({ key: 'notes', label: 'Notes' });
  return tabs;
});

type TopTabKey = 'overview' | 'contents' | 'ssl' | 'websocket' | 'summary' | 'chart' | 'notes';
const activeTopTab = ref<TopTabKey>('contents');

const expandedSections = ref({
  connection: false,
  timing: false,
  size: false,
});

const expandedSSLSections = ref({
  protocol: true,
  session: true,
  cipher: true,
  alpn: true,
  certificates: true,
  extensions: true,
});

const expandedExtensionSections = ref({
  server: true,
});

const expandedCertificateSections = ref<Record<number, boolean>>({});

const toggleSection = (section: 'connection' | 'timing' | 'size') => {
  expandedSections.value[section] = !expandedSections.value[section];
};

const toggleSSLSection = (section: keyof typeof expandedSSLSections.value) => {
  expandedSSLSections.value[section] = !expandedSSLSections.value[section];
};

const toggleExtensionSection = (section: keyof typeof expandedExtensionSections.value) => {
  expandedExtensionSections.value[section] = !expandedExtensionSections.value[section];
};

const toggleCertificateSection = (index: number) => {
  expandedCertificateSections.value[index] = !(expandedCertificateSections.value[index] ?? true);
};

const currentTopTabLabel = computed(
  () => topTabs.value.find((tab) => tab.key === activeTopTab.value)?.label || 'View',
);

const websocketFrames = computed(() => selectedRequest.value?.websocket_frames || []);

const frameDirectionLabel = (direction: string) => {
  if (direction === 'sent') return '↑ Sent';
  if (direction === 'received') return '↓ Received';
  return direction || '-';
};

const formatFrameTime = (value?: string) => {
  if (!value) return '-';
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format('HH:mm:ss') : '-';
};

const headerSummary = computed(() => {
  const left = `${req.value.method} ${req.value.url}`;
  let right: string;
  if (req.value.error) {
    right = `Failed — ${req.value.error}`;
  } else {
    right = `${req.value.status_code} ${req.value.status_reason || ''}`.trim();
  }
  return `${left} → ${right}`;
});

const formatDateTime = (value?: string) => {
  if (!value) return '-';
  if (value.startsWith('0001-01-01')) return '-';
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '-';
};

const formatDuration = (value?: number) => {
  if (!value && value !== 0) return '-';
  return value > 0 ? `${value} ms` : '-';
};

const formatBytes = (value?: number) => {
  if (!value && value !== 0) return '-';
  if (value === 0) return '0 bytes';
  const units = ['bytes', 'KB', 'MB', 'GB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  const display = unit === 0 ? `${Math.round(size)}` : size.toFixed(size >= 10 ? 1 : 2);
  return `${display} ${units[unit]} (${value.toLocaleString()} bytes)`;
};

const formatSpeed = (bytes?: number, durationMs?: number) => {
  if (!bytes || !durationMs) return '-';
  const bytesPerSecond = bytes / (durationMs / 1000);
  if (!Number.isFinite(bytesPerSecond) || bytesPerSecond <= 0) return '-';
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
  let size = bytesPerSecond;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(size >= 10 ? 1 : 2)} ${units[unit]}`;
};

const protocolLabel = computed(() => {
  const proto = req.value.http_version || '-';
  if (req.value.scheme === 'socks5') return proto;
  return proto;
});

const sslLabel = computed(() => {
  if (!req.value.tls_version) return req.value.scheme === 'https' ? 'TLS' : '-';
  const cipher = req.value.tls_cipher_suite ? ` (${req.value.tls_cipher_suite})` : '';
  return `${req.value.tls_version}${cipher}`;
});

const overviewRows = computed(() => {
  const rows: { key: string; value: string; error?: boolean }[] = [
    { key: 'URL', value: req.value.url || '-' },
    { key: 'Method', value: req.value.method || '-' },
    { key: 'Status', value: req.value.error ? 'Failed' : (req.value.status_code ? 'Complete' : '-') },
    { key: 'Response Code', value: req.value.status_code ? String(req.value.status_code) : '-' },
  ];
  if (req.value.error) {
    rows.push({ key: 'Error', value: req.value.error, error: true });
  }
  rows.push(
    { key: 'Content-Type', value: req.value.resp_content_type || req.value.content_type || '-' },
    { key: 'Client Address', value: req.value.client_addr || req.value.remote_addr || '-' },
    { key: 'Remote Address', value: req.value.server_addr || req.value.host || '-' },
    { key: 'Protocol', value: protocolLabel.value },
    { key: 'Tags', value: '-' },
    { key: 'Kept Alive', value: req.value.keep_alive ? 'Yes' : 'No' },
    { key: 'SSL', value: sslLabel.value },
  );
  return rows;
});

const connectionRows = computed(() => [
  { key: 'Client Connection', value: req.value.client_addr || req.value.remote_addr || '-' },
  { key: 'Server Connection', value: req.value.server_addr || req.value.host || '-' },
]);

const timingRows = computed(() => [
  { key: 'Request Start Time', value: formatDateTime(req.value.request_start_time || req.value.created_at) },
  { key: 'Request End Time', value: formatDateTime(req.value.request_end_time) },
  { key: 'Response Start Time', value: formatDateTime(req.value.response_start_time) },
  { key: 'Response End Time', value: formatDateTime(req.value.response_end_time) },
  { key: 'Duration', value: formatDuration(req.value.duration) },
  { key: 'DNS', value: formatDuration(req.value.dns_duration) },
  { key: 'Connect', value: formatDuration(req.value.connect_duration) },
  { key: 'TLS Handshake', value: formatDuration(req.value.tls_handshake_duration) },
  { key: 'Request', value: formatDuration(req.value.request_duration) },
  { key: 'Response', value: formatDuration(req.value.response_duration) },
  { key: 'Latency', value: formatDuration(req.value.latency_duration) },
  { key: 'Speed', value: formatSpeed((req.value.body_size || 0) + (req.value.resp_body_size || 0), req.value.duration) },
  { key: 'Request Speed', value: formatSpeed(req.value.body_size || 0, req.value.request_duration) },
  { key: 'Response Speed', value: formatSpeed(req.value.resp_body_size || 0, req.value.response_duration) },
]);

const sizeRows = computed(() => {
  const reqSize = req.value.body_size || 0;
  const respSize = req.value.resp_body_size || 0;
  return [
    { key: 'Request', value: formatBytes(reqSize) },
    { key: 'Response', value: formatBytes(respSize) },
    { key: 'Total', value: formatBytes(reqSize + respSize) },
  ];
});

const dash = (value?: string | number | boolean | null) => {
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (value === null || value === undefined || value === '') return '-';
  return String(value);
};

const sslCertificates = computed(() => req.value.tls_server_certificates || []);
const sslServerExtensions = computed(() => req.value.tls_server_extensions || []);

const sslProtocolRows = computed(() => [
  { key: 'Server Chosen', value: dash(req.value.tls_version) },
  { key: 'Server Name', value: dash(req.value.tls_server_name || req.value.host) },
  { key: 'Curve', value: dash(req.value.tls_curve_id) },
]);

const sslSessionRows = computed(() => [
  { key: 'Resumed', value: dash(req.value.tls_did_resume) },
  { key: 'Client Requested', value: '-' },
  { key: 'Client Session ID', value: '-' },
  { key: 'PacketMind Requested', value: req.value.tls_did_resume ? 'pre_shared_key (41)' : '-' },
  { key: 'Server Session ID', value: req.value.tls_did_resume ? 'session resumed' : '-' },
]);

const sslCipherRows = computed(() => [
  { key: 'Server Chosen', value: dash(req.value.tls_cipher_suite) },
]);

const sslALPNRows = computed(() => [
  { key: 'Server Chosen', value: dash(req.value.tls_alpn) },
  { key: 'Client Certificates', value: '0' },
  { key: 'OCSP Stapled', value: dash(req.value.tls_ocsp_stapled) },
  { key: 'Signed Certificate Timestamps', value: req.value.tls_sct_count ? String(req.value.tls_sct_count) : '-' },
]);

const joinList = (values?: string[]) => values && values.length ? values.join(', ') : '-';

const buildCertificateRows = (cert: TLSCertificate) => [
  { key: 'Subject', value: dash(cert.subject) },
  { key: 'Issuer', value: dash(cert.issuer) },
  { key: 'Serial Number', value: dash(cert.serial_number) },
  { key: 'Version', value: dash(cert.version) },
  { key: 'Is CA', value: dash(cert.is_ca) },
  { key: 'Signature Algorithm', value: dash(cert.signature_algorithm) },
  { key: 'Public Key', value: dash(cert.public_key_algorithm) },
  { key: 'Not Before', value: formatDateTime(cert.not_before) },
  { key: 'Not After', value: formatDateTime(cert.not_after) },
  { key: 'DNS Names', value: joinList(cert.dns_names) },
  { key: 'Email Addresses', value: joinList(cert.email_addresses) },
  { key: 'IP Addresses', value: joinList(cert.ip_addresses) },
  { key: 'OCSP Servers', value: joinList(cert.ocsp_servers) },
  { key: 'Issuing Certificate URL', value: joinList(cert.issuing_certificate_url) },
  { key: 'Extensions', value: cert.extensions?.length ? String(cert.extensions.length) : '-' },
];

const summaryResource = computed(() => {
  const path = req.value.path || '/';
  const query = req.value.query_string ? `?${req.value.query_string}` : '';
  return `${path}${query}`;
});

const headerByteLength = (headers?: Record<string, string[]> | null) => {
  if (!headers) return 0;
  return Object.entries(headers).reduce((total, [key, values]) => {
    return total + values.reduce((sum, value) => sum + key.length + value.length + 4, 0);
  }, 0);
};

const summaryHeaderSize = computed(() => String(headerByteLength(req.value.headers) + headerByteLength(req.value.resp_headers)));
const summaryBodySize = computed(() => String((req.value.body_size || 0) + (req.value.resp_body_size || 0)));
const summaryTime = computed(() => formatDuration(req.value.duration));

const MIN_HEIGHT = 140;
const DEFAULT_REQUEST_HEIGHT = 360;
const requestHeight = ref(DEFAULT_REQUEST_HEIGHT);
const containerRefHeight = ref(720);

let cleanupResize: (() => void) | null = null;

onMounted(async () => {
  const savedHeight = localStorage.getItem('detail_request_height');
  if (savedHeight) {
    const parsed = parseInt(savedHeight, 10);
    if (parsed >= MIN_HEIGHT) {
      requestHeight.value = parsed;
    }
  }
  
  const size = await getWindowSize();
  containerRefHeight.value = size.height;
  
  cleanupResize = onWindowResize((data) => {
    containerRefHeight.value = data.height;
  });
  
  // Register document listeners for resizing on mount instead of dynamically
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', stopResize);
});

let isResizing = false;
let startY = 0;
let startHeight = 0;

const startResize = (event: MouseEvent) => {
  isResizing = true;
  startY = event.clientY;
  startHeight = requestHeight.value;
  document.body.classList.add('resizing-row');
};

const onMouseMove = (event: MouseEvent) => {
  if (!isResizing) return;
  const deltaY = event.clientY - startY;
  const maxHeight = Math.max(MIN_HEIGHT, containerRefHeight.value - 260);
  requestHeight.value = Math.min(maxHeight, Math.max(MIN_HEIGHT, startHeight + deltaY));
};

const stopResize = () => {
  if (!isResizing) return;
  isResizing = false;
  localStorage.setItem('detail_request_height', String(requestHeight.value));
  document.body.classList.remove('resizing-row');
};

watch(topTabs, (tabs) => {
  if (!tabs.some((tab) => tab.key === activeTopTab.value)) {
    activeTopTab.value = 'contents';
  }
}, { immediate: true });

onUnmounted(() => {
  if (cleanupResize) {
    cleanupResize();
  }
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mouseup', stopResize);
});
</script>

<style module>
.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #d7d7d7;
  border: 1px solid #9f9f9f;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.35);
}

.empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}

.headerBar {
  flex-shrink: 0;
  background: linear-gradient(180deg, #ebebeb 0%, #dddddd 100%);
  border-bottom: 1px solid #ababab;
}

.headerLine {
  height: 28px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  border-bottom: 1px solid #bcbcbc;
}

.headerText {
  font-size: 12px;
  color: #101010;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.topTabs {
  height: 34px;
  display: flex;
  align-items: stretch;
  background: #e5e5e5;
}

.topTab {
  position: relative;
  border: none;
  background: transparent;
  padding: 0 14px;
  font-size: 12px;
  color: #171717;
  cursor: pointer;
}

.topTab:hover {
  background: #f2f2f2;
}

.topTabActive {
  background: #d7d7d7;
}

.topTabActive::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  background: #75a9df;
}

.contentShell {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.requestSection {
  flex-shrink: 0;
  min-height: 140px;
  overflow: hidden;
}

.divider {
  height: 1px;
  background: #bdbdbd;
  cursor: row-resize;
  position: relative;
  flex-shrink: 0;
}

.divider::before {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: -3px;
  bottom: -3px;
}

.dividerLine {
  width: 100%;
  height: 100%;
  background: #bdbdbd;
}

.divider:hover .dividerLine,
.divider:hover {
  background: #75a9df;
}

.responseSection {
  flex: 1;
  min-height: 140px;
  overflow: hidden;
}

.overviewPanel {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: #e6e6e6;
}

.overviewGrid {
  display: grid;
  grid-template-columns: minmax(420px, 1fr) minmax(420px, 1fr);
  gap: 24px;
  padding: 10px 14px 18px;
}

.overviewSection {
  min-width: 0;
}

.overviewContent {
  padding: 10px 14px 18px;
}

.collapsibleSection {
  border-top: 1px solid #c5c5c5;
}

.sectionHeader {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 0;
  font-size: 12px;
  font-weight: 700;
  color: #2b2b2b;
  cursor: pointer;
  user-select: none;
}

.sectionContent {
  padding: 0 0 4px 0;
}

.nestedSection {
  margin-left: 20px;
}

.nestedHeader {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  font-size: 12px;
  color: #2f2f2f;
  cursor: pointer;
  user-select: none;
}

.sectionTitle {
  margin: 14px 0 6px;
  padding-top: 6px;
  border-top: 1px solid #c5c5c5;
  font-size: 12px;
  font-weight: 700;
  color: #2b2b2b;
}

.overviewRow {
  display: grid;
  grid-template-columns: minmax(96px, 160px) minmax(0, 1fr);
  gap: 14px;
  padding: 4px 0;
}

.overviewKey {
  text-align: right;
  font-size: 12px;
  color: #2f2f2f;
}

.overviewValue {
  font-size: 12px;
  color: #111;
  word-break: break-all;
}

.overviewValueError {
  color: #d32f2f;
  word-break: break-all;
  font-size: 11px;
}

.narrowDetail .overviewRow {
  grid-template-columns: 1fr;
  gap: 2px 0;
}

.narrowDetail .overviewKey {
  text-align: left;
  font-weight: 700;
}

.narrowDetail .overviewValue {
  padding-left: 12px;
  word-break: break-word;
}

.summaryPanel {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: #e6e6e6;
}

.summaryTableWrap {
  padding: 8px 10px 12px;
}

.summaryTable {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: 12px;
  color: #111;
}

.summaryTable thead th {
  padding: 4px 6px;
  border-bottom: 1px solid #bdbdbd;
  border-right: 1px solid #d2d2d2;
  text-align: left;
  font-weight: 400;
}

.summaryTable tbody td {
  padding: 6px 6px;
  border-right: 1px solid #d2d2d2;
  vertical-align: top;
}

.summaryTable thead th:last-child,
.summaryTable tbody td:last-child {
  border-right: none;
}

.summaryColIndex {
  width: 32px;
}

.summaryColResource {
  width: 260px;
}

.summaryColHost {
  width: 300px;
}

.summaryColCode {
  width: 56px;
}

.summaryColMime {
  width: 220px;
}

.summaryColNumber {
  width: 72px;
}

.summaryColTime {
  width: 82px;
}

.summaryCellIndex,
.summaryCellNumber,
.summaryCellTime {
  text-align: right;
  white-space: nowrap;
}

.summaryCellText {
  word-break: break-all;
}

.summaryTotalLabel {
  font-weight: 400;
}

.placeholderPanel {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: #e6e6e6;
  color: #444;
}

.placeholderTitle {
  font-size: 16px;
  font-weight: 600;
}

.placeholderText {
  font-size: 13px;
}

:global(.resizing-row) {
  cursor: row-resize !important;
  user-select: none !important;
}
</style>
