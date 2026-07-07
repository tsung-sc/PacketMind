<template>
  <a-modal
    :open="visible"
    :title="title"
    :width="720"
    :confirmLoading="saving"
    @ok="handleSave"
    @cancel="handleCancel"
  >
    <a-form :class="$style.form" layout="vertical">
      <template v-if="mode === 'proxy'">
        <a-row :gutter="12">
          <a-col :span="24">
            <a-form-item label="Proxy Port">
              <a-input-number v-model:value="draft.proxy.listener.port" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-space direction="vertical" :size="6">
          <a-checkbox v-model:checked="draft.proxy.listener.http_enabled">Enable HTTP Proxy</a-checkbox>
          <a-checkbox v-model:checked="draft.proxy.listener.https_enabled">Enable HTTPS Proxy</a-checkbox>
          <a-checkbox v-model:checked="draft.proxy.listener.socks5_enabled">Enable SOCKS5 Proxy</a-checkbox>
          <a-checkbox v-model:checked="draft.proxy.listener.mitm_enabled">Enable HTTPS MITM</a-checkbox>
        </a-space>
      </template>

      <template v-else-if="mode === 'recording'">
        <a-checkbox v-model:checked="draft.proxy.recording.enabled">Enable recording</a-checkbox>
        <a-form-item label="Max Capture Body Size (MB)" style="margin-top: 12px">
          <a-input-number v-model:value="draft.proxy.recording.max_capture_body_size_mb" :min="1" :max="100" style="width: 100%" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'ssl'">
        <a-checkbox v-model:checked="draft.proxy.ssl_proxying.enabled">Enable SSL Proxying</a-checkbox>
        <a-form-item label="Include Hosts (one per line)" style="margin-top: 12px">
          <a-textarea v-model:value="sslIncludeHosts" :rows="6" />
        </a-form-item>
        <a-form-item label="Exclude Hosts (one per line)">
          <a-textarea v-model:value="sslExcludeHosts" :rows="4" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'access-control'">
        <a-checkbox v-model:checked="draft.proxy.access_control.enabled">Enable Access Control</a-checkbox>
        <a-form-item label="Allowed Client Addresses (one per line)" style="margin-top: 12px">
          <a-textarea v-model:value="allowedClients" :rows="6" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'external-proxy'">
        <a-checkbox v-model:checked="draft.proxy.external_proxy.enabled">Enable External Proxy</a-checkbox>
        <a-row :gutter="12" style="margin-top: 12px">
          <a-col :span="8">
            <a-form-item label="Scheme">
              <a-select v-model:value="draft.proxy.external_proxy.scheme">
                <a-select-option value="http">http</a-select-option>
                <a-select-option value="https">https</a-select-option>
                <a-select-option value="socks5">socks5</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="10">
            <a-form-item label="Host">
              <a-input v-model:value="draft.proxy.external_proxy.host" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="Port">
              <a-input-number v-model:value="draft.proxy.external_proxy.port" :min="1" :max="65535" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="Username">
          <a-input v-model:value="draft.proxy.external_proxy.username" />
        </a-form-item>
        <a-form-item label="Password">
          <a-input-password v-model:value="draft.proxy.external_proxy.password" placeholder="Leave blank to keep existing password" />
          <div v-if="draft.proxy.external_proxy.password_configured" :class="$style.fieldHint">
            A password is already stored locally. Leave blank to keep it unchanged.
          </div>
        </a-form-item>
        <a-form-item label="Bypass Hosts (one per line)">
          <a-textarea v-model:value="externalProxyBypassHosts" :rows="5" placeholder="localhost&#10;127.0.0.1&#10;*.internal.example.com&#10;10.0.0.0/8" />
          <div :class="$style.fieldHint">
            Requests matching these hosts, suffixes, or CIDR ranges bypass the upstream proxy.
          </div>
        </a-form-item>
      </template>

      <template v-else-if="mode === 'throttling'">
        <a-checkbox v-model:checked="draft.proxy.throttling.enabled">Enable Throttling</a-checkbox>
        <a-row :gutter="12" style="margin-top: 12px">
          <a-col :span="8">
            <a-form-item label="Latency (ms)">
              <a-input-number v-model:value="draft.proxy.throttling.latency_ms" :min="0" :max="10000" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Downstream (KB/s)">
              <a-input-number v-model:value="draft.proxy.throttling.downstream_kbps" :min="0" :max="1048576" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Upstream (KB/s)">
              <a-input-number v-model:value="draft.proxy.throttling.upstream_kbps" :min="0" :max="1048576" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
      </template>

      <template v-else-if="mode === 'breakpoints'">
        <a-checkbox v-model:checked="draft.proxy.breakpoints.enabled">Enable Breakpoints</a-checkbox>
        <a-form-item label="Request Matchers (one per line)" style="margin-top: 12px">
          <a-textarea v-model:value="requestMatchers" :rows="5" />
        </a-form-item>
        <a-form-item label="Response Matchers (one per line)">
          <a-textarea v-model:value="responseMatchers" :rows="5" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'reverse-proxy'">
        <a-checkbox v-model:checked="draft.proxy.reverse_proxy.enabled">Enable Reverse Proxies</a-checkbox>
        <a-form-item label="Rules (source => target, one per line)" style="margin-top: 12px">
          <a-textarea v-model:value="reverseProxyRules" :rows="8" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'port-forwarding'">
        <a-checkbox v-model:checked="draft.proxy.port_forwarding.enabled">Enable Port Forwarding</a-checkbox>
        <a-form-item label="Rules (listenHost:listenPort => targetHost:targetPort, one per line)" style="margin-top: 12px">
          <a-textarea v-model:value="portForwardRules" :rows="8" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'web-interface'">
        <a-checkbox v-model:checked="draft.proxy.web_interface.enabled">Enable Web Interface</a-checkbox>
        <a-form-item label="Port" style="margin-top: 12px">
          <a-input-number v-model:value="draft.proxy.web_interface.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'mcp-server'">
        <a-form-item label="MCP Server">
          <a-switch v-model:checked="draft.mcp_server.enabled" />
          <span style="margin-left: 8px; font-size: 12px; color: #888">
            Expose traffic analysis tools to external AI agents
          </span>
        </a-form-item>
        <a-form-item v-if="draft.mcp_server.enabled" label="Host / Port">
          <a-input v-model:value="draft.mcp_server.host" style="width: 120px" :disabled="!draft.mcp_server.enabled" />
          <span style="margin: 0 6px">:</span>
          <a-input-number v-model:value="draft.mcp_server.port" :min="1" :max="65535" style="width: 80px" :disabled="!draft.mcp_server.enabled" />
        </a-form-item>
      </template>

      <template v-else-if="mode === 'preferences'">
        <a-space direction="vertical" :size="8">
          <a-checkbox v-model:checked="draft.window.structure_view">Use Structure View</a-checkbox>
          <a-checkbox v-model:checked="draft.window.use_dark_theme">Use Dark Theme</a-checkbox>
          <a-checkbox v-model:checked="draft.tools.auto_save_enabled">Enable Auto Save</a-checkbox>
        </a-space>
      </template>
    </a-form>

    <a-alert v-if="error" :message="error" type="error" show-icon style="margin-top: 12px" />
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch, toRaw } from 'vue';
import type { AppSettings } from '@/types';

type Mode = 'proxy' | 'recording' | 'ssl' | 'access-control' | 'external-proxy' | 'throttling' | 'breakpoints' | 'reverse-proxy' | 'port-forwarding' | 'web-interface' | 'mcp-server' | 'preferences';

interface Props {
  visible: boolean;
  mode: Mode;
  settings: AppSettings;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'save', value: AppSettings): void;
}>();

const saving = ref(false);
const error = ref('');

const cloneSettings = (settings: AppSettings): AppSettings => {
  return JSON.parse(JSON.stringify(toRaw(settings))) as AppSettings;
};

const draft = ref<AppSettings>(cloneSettings(props.settings));
const sslIncludeHosts = ref('');
const sslExcludeHosts = ref('');
const allowedClients = ref('');
const requestMatchers = ref('');
const responseMatchers = ref('');
const reverseProxyRules = ref('');
const portForwardRules = ref('');
const externalProxyBypassHosts = ref('');

const titles: Record<Mode, string> = {
  proxy: 'Proxy Settings',
  recording: 'Recording Settings',
  ssl: 'SSL Proxying Settings',
  'access-control': 'Access Control Settings',
  'external-proxy': 'External Proxy Settings',
  throttling: 'Throttling Settings',
  breakpoints: 'Breakpoint Settings',
  'reverse-proxy': 'Reverse Proxies',
  'port-forwarding': 'Port Forwarding',
  'web-interface': 'Web Interface Settings',
  'mcp-server': 'MCP Server Settings',
  preferences: 'Preferences',
};

const title = computed(() => titles[props.mode]);

const splitLines = (value: string) => value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean);

const syncDraft = () => {
  draft.value = cloneSettings(props.settings);
  // Ensure mcp_server defaults are populated
  if (!draft.value.mcp_server) {
    draft.value.mcp_server = { enabled: false, host: '127.0.0.1', port: 8889 };
  }
  draft.value.mcp_server.enabled = draft.value.mcp_server.enabled ?? false;
  draft.value.mcp_server.host = draft.value.mcp_server.host || '127.0.0.1';
  draft.value.mcp_server.port = draft.value.mcp_server.port || 8889;
  sslIncludeHosts.value = draft.value.proxy.ssl_proxying.include_hosts.join('\n');
  sslExcludeHosts.value = draft.value.proxy.ssl_proxying.exclude_hosts.join('\n');
  allowedClients.value = draft.value.proxy.access_control.allowed_clients.join('\n');
  externalProxyBypassHosts.value = (draft.value.proxy.external_proxy.bypass_hosts || []).join('\n');
  requestMatchers.value = draft.value.proxy.breakpoints.request_matchers.join('\n');
  responseMatchers.value = draft.value.proxy.breakpoints.response_matchers.join('\n');
  reverseProxyRules.value = draft.value.proxy.reverse_proxy.rules.map((rule) => `${rule.source} => ${rule.target}`).join('\n');
  portForwardRules.value = draft.value.proxy.port_forwarding.rules.map((rule) => `${rule.listen_host}:${rule.listen_port} => ${rule.target_host}:${rule.target_port}`).join('\n');
};

watch(() => [props.visible, props.mode, props.settings] as const, () => {
  if (props.visible) {
    error.value = '';
    syncDraft();
  }
}, { deep: true, immediate: true });

const applyTextFields = () => {
  draft.value.proxy.ssl_proxying.include_hosts = splitLines(sslIncludeHosts.value);
  draft.value.proxy.ssl_proxying.exclude_hosts = splitLines(sslExcludeHosts.value);
  draft.value.proxy.access_control.allowed_clients = splitLines(allowedClients.value);
  draft.value.proxy.external_proxy.bypass_hosts = splitLines(externalProxyBypassHosts.value);
  draft.value.proxy.breakpoints.request_matchers = splitLines(requestMatchers.value);
  draft.value.proxy.breakpoints.response_matchers = splitLines(responseMatchers.value);
  draft.value.proxy.reverse_proxy.rules = splitLines(reverseProxyRules.value)
    .map((line) => line.split(/\s*=>\s*/))
    .filter((parts) => parts.length === 2)
    .map(([source, target]) => ({ source, target }));
  draft.value.proxy.port_forwarding.rules = splitLines(portForwardRules.value)
    .map((line) => line.split(/\s*=>\s*/))
    .filter((parts) => parts.length === 2)
    .map(([listen, target]) => {
      const [listen_host, listen_port] = listen.split(':');
      const [target_host, target_port] = target.split(':');
      return {
        listen_host,
        listen_port: Number(listen_port || 0),
        target_host,
        target_port: Number(target_port || 0),
      };
    })
    .filter((rule) => rule.listen_host && rule.target_host && rule.listen_port > 0 && rule.target_port > 0);
};

const handleSave = async () => {
  error.value = '';
  applyTextFields();
  saving.value = true;
  try {
    emit('save', draft.value);
    emit('update:visible', false);
  } catch (err: any) {
    error.value = err?.message || 'Failed to save settings';
  } finally {
    saving.value = false;
  }
};

const handleCancel = () => {
  emit('update:visible', false);
};
</script>

<style module>
.form {
  margin-top: 4px;
}

.form :global(.ant-form-item) {
  margin-bottom: 8px;
}

.form :global(.ant-form-item-label) {
  padding-bottom: 4px;
}

.form :global(.ant-form-item-label label) {
  font-size: 12px;
  font-weight: 600;
  color: #333;
}

.form :global(.ant-checkbox-wrapper) {
  font-size: 12px;
}

.form :global(.ant-input),
.form :global(.ant-input-number),
.form :global(.ant-select) {
  width: 100%;
}

.fieldHint {
  margin-top: 4px;
  font-size: 11px;
  color: #666;
}
</style>
