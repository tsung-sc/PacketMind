<template>
  <div :class="$style.menuBar" @mouseleave="closeMenu" data-application-menu-root="true">
    <div
      v-for="group in menuGroups"
      :key="group.key"
      :class="$style.menuGroup"
      @mouseenter="handleGroupEnter(group.key)"
    >
      <button
        type="button"
        :class="[$style.menuTrigger, { [$style.menuTriggerActive]: openMenuKey === group.key }]"
        @click.stop="toggleMenu(group.key)"
      >
        {{ group.label }}
      </button>

      <div v-if="openMenuKey === group.key" :class="$style.dropdown">
        <template v-for="item in group.items" :key="item.key">
          <div v-if="item.separator" :class="$style.divider"></div>
          
          <div 
            v-else 
            :class="$style.menuItemWrapper"
            @mouseenter="hoveredItemKey = item.key"
          >
            <button
              type="button"
              :disabled="item.disabled"
              :class="[$style.menuItem, { [$style.menuItemDisabled]: item.disabled }]"
              @click.stop="handleItemClick(item)"
            >
              <span :class="$style.checkSlot">{{ item.checked ? '✓' : '' }}</span>
              <span :class="$style.itemLabel">{{ item.label }}</span>
              <span v-if="item.hint && !item.children" :class="$style.itemHint">{{ item.hint }}</span>
              <span v-if="item.children" :class="$style.itemArrow">▶</span>
            </button>

            <!-- Submenu flyout -->
            <div 
              v-if="item.children && hoveredItemKey === item.key" 
              :class="$style.subDropdown"
            >
              <template v-for="child in item.children" :key="child.key">
                <div v-if="child.separator" :class="$style.divider"></div>
                <button
                  v-else
                  type="button"
                  :disabled="child.disabled"
                  :class="[$style.menuItem, { [$style.menuItemDisabled]: child.disabled }]"
                  @click.stop="handleItemClick(child)"
                >
                  <span :class="$style.checkSlot">{{ child.checked ? '✓' : '' }}</span>
                  <span :class="$style.itemLabel">{{ child.label }}</span>
                  <span v-if="child.hint" :class="$style.itemHint">{{ child.hint }}</span>
                </button>
              </template>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';

interface MenuItem {
  key: string;
  label?: string;
  action?: string;
  disabled?: boolean;
  checked?: boolean;
  separator?: boolean;
  hint?: string;
  children?: MenuItem[];
}

interface MenuGroup {
  key: string;
  label: string;
  items: MenuItem[];
}

interface Props {
  proxyRunning: boolean;
  hasSelectedRequest: boolean;
  hasActiveSession: boolean;
  isMaximised: boolean;
  externalProxyEnabled?: boolean;
  throttlingEnabled?: boolean;
  sslProxyingEnabled?: boolean;
  accessControlEnabled?: boolean;
  mcpServerEnabled?: boolean;
  breakpointsEnabled?: boolean;
  noCachingEnabled?: boolean;
  blockCookiesEnabled?: boolean;
  hasProxySettings?: boolean;
  
  // New props for dynamic submenus
  recentSessions?: { id: string; name: string }[];
  hostFilterOptions?: { host: string; checked: boolean }[];
  structureViewState?: boolean;
  statusBarState?: boolean;
  autoSaveState?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  recentSessions: () => [],
  hostFilterOptions: () => [],
  structureViewState: true,
  statusBarState: true,
  autoSaveState: false,
});

const emit = defineEmits<{
  (e: 'action', action: string): void;
}>();

const openMenuKey = ref<string | null>(null);
const hoveredItemKey = ref<string | null>(null);

const menuGroups = computed<MenuGroup[]>(() => [
  {
    key: 'file',
    label: 'File',
    items: [
      { key: 'file-new-session', label: 'New Session', action: 'file:new-session' },
      { key: 'file-open-session', label: 'Open...', action: 'file:open-session' },
      { 
        key: 'file-open-recent', 
        label: 'Open Recent', 
        disabled: props.recentSessions.length === 0,
        children: [
          ...props.recentSessions.map(session => ({
            key: `recent-${session.id}`,
            label: session.name,
            action: `file:open-recent:${session.id}`
          })),
          { key: 'file-recent-sep', separator: true },
          { key: 'file-clear-recent', label: 'Clear Recent', action: 'file:clear-recent' }
        ]
      },
      { key: 'file-close', label: 'Close', action: 'file:close', disabled: !props.hasActiveSession },
      { key: 'file-close-all', label: 'Close All', action: 'file:close-all', disabled: !props.hasActiveSession },
      { key: 'file-save-session', label: 'Save', action: 'file:save-session', disabled: !props.hasActiveSession },
      { key: 'file-save-session-as', label: 'Save As...', action: 'file:save-session-as', disabled: !props.hasActiveSession },
      { key: 'file-sep-1', separator: true },
      { key: 'file-export-session', label: 'Export...', action: 'file:export-session', disabled: !props.hasActiveSession },
      { key: 'file-import', label: 'Import...', action: 'file:import-session' },
      { key: 'file-sep-2', separator: true },
      { key: 'file-export-curl', label: 'Export Selected as cURL', action: 'file:export-curl', disabled: !props.hasSelectedRequest },
      { key: 'file-export-har', label: 'Export Selected as HAR', action: 'file:export-har', disabled: !props.hasSelectedRequest },
      { key: 'file-clear-session', label: 'Clear Session', action: 'file:clear-session', disabled: !props.hasActiveSession },
      { key: 'file-sep-3', separator: true },
      { key: 'file-exit', label: navigator.platform.includes('Mac') ? 'Quit PacketMind' : 'Exit PacketMind', action: 'file:exit' },
    ],
  },
  {
    key: 'edit',
    label: 'Edit',
    items: [
      { key: 'edit-cut', label: 'Cut', action: 'edit:cut' },
      { key: 'edit-copy', label: 'Copy', action: 'edit:copy' },
      { key: 'edit-copy-curl', label: 'Copy cURL', action: 'edit:copy-curl', disabled: !props.hasSelectedRequest },
      { key: 'edit-paste', label: 'Paste', action: 'edit:paste' },
      { key: 'edit-delete', label: 'Delete', action: 'edit:delete', disabled: !props.hasSelectedRequest },
      { key: 'edit-select-all', label: 'Select All', action: 'edit:select-all' },
      { key: 'edit-find', label: 'Find...', action: 'edit:find' },
      { key: 'edit-find-next', label: 'Find Next', action: 'edit:find-next', disabled: true },
      { key: 'edit-find-prev', label: 'Find Previous', action: 'edit:find-prev', disabled: true },
      { key: 'edit-sep-1', separator: true },
      { key: 'edit-preferences', label: 'Preferences...', action: 'edit:preferences' },
    ],
  },
  {
    key: 'view',
    label: 'View',
    items: [
      { key: 'view-structure', label: 'Structure', checked: props.structureViewState, action: 'view:structure' },
      { key: 'view-sequence', label: 'Sequence', checked: !props.structureViewState, action: 'view:sequence' },
      { key: 'view-sep-1', separator: true },
      { key: 'view-refresh', label: 'Refresh Requests', action: 'view:refresh-requests', disabled: !props.hasActiveSession },
      { key: 'view-expand', label: 'Expand All', action: 'view:expand-all', disabled: !props.hasActiveSession },
      { key: 'view-collapse', label: 'Collapse All', action: 'view:collapse-all', disabled: !props.hasActiveSession },
      { key: 'view-sep-2', separator: true },
      { 
        key: 'view-focused-hosts', 
        label: 'Focused Hosts...', 
        disabled: props.hostFilterOptions.length === 0,
        children: props.hostFilterOptions.length > 0 ? [
          ...props.hostFilterOptions.map(h => ({
            key: `host-${h.host}`,
            label: h.host,
            checked: h.checked,
            action: `view:toggle-host:${h.host}`
          })),
          { key: 'host-sep', separator: true },
          { key: 'host-clear', label: 'Clear Focus', action: 'view:clear-focus' }
        ] : []
      },
      { key: 'view-chart', label: 'Chart', action: 'view:chart', disabled: true },
      { key: 'view-status-bar', label: 'Status Bar', checked: props.statusBarState, action: 'view:toggle-status-bar' },
      { key: 'view-full-screen', label: 'Full Screen', action: 'view:fullscreen' },
    ],
  },
  {
    key: 'proxy',
    label: 'Proxy',
    items: [
      {
        key: 'proxy-throttling-toggle',
        label: props.throttlingEnabled ? 'Stop Throttling' : 'Start Throttling',
        action: 'proxy:toggle-throttling',
        checked: props.throttlingEnabled,
      },
      { key: 'proxy-sep-1', separator: true },
      { key: 'proxy-settings', label: 'Proxy Settings...', action: 'proxy:settings', disabled: !props.hasProxySettings },
      { key: 'proxy-ssl-settings', label: 'SSL Proxying Settings...', action: 'proxy:ssl-settings', checked: props.sslProxyingEnabled, disabled: !props.hasProxySettings },
      { key: 'proxy-access-control', label: 'Access Control Settings...', action: 'proxy:access-control', checked: props.accessControlEnabled, disabled: !props.hasProxySettings },
      { key: 'proxy-web-interface', label: 'Web Interface Settings...', action: 'proxy:web-interface', disabled: !props.hasProxySettings },
      { key: 'proxy-mcp-server', label: 'MCP Server Settings...', action: 'proxy:mcp-server', checked: props.mcpServerEnabled },
      { key: 'proxy-external-proxy', label: 'External Proxy Settings...', action: 'proxy:external-proxy', checked: props.externalProxyEnabled, disabled: !props.hasProxySettings },
      { key: 'proxy-sep-2', separator: true },
      { 
        key: 'proxy-system-menu', 
        label: navigator.platform.includes('Mac') ? 'macOS Proxy' : 'Windows Proxy',
        children: [
          { key: 'proxy-sys-enable', label: 'Enable System Proxy', action: 'proxy:sys-enable', checked: false },
          { key: 'proxy-sys-settings', label: 'System Proxy Settings...', action: 'proxy:sys-settings' }
        ]
      },
      { key: 'proxy-firefox', label: 'Firefox Proxy', action: 'proxy:firefox', disabled: true },
      { key: 'proxy-sep-3', separator: true },
      { key: 'proxy-throttle-settings', label: 'Throttling Settings...', action: 'proxy:throttling-settings', disabled: !props.hasProxySettings },
      { key: 'proxy-reverse-proxies', label: 'Reverse Proxies...', action: 'proxy:reverse-proxies', disabled: !props.hasProxySettings },
      { key: 'proxy-port-forwarding', label: 'Port Forwarding...', action: 'proxy:port-forwarding', disabled: !props.hasProxySettings },
      { key: 'proxy-sep-4', separator: true },
      { key: 'proxy-breakpoint-settings', label: 'Breakpoint Settings...', action: 'proxy:breakpoint-settings', disabled: !props.hasProxySettings },
      { key: 'proxy-enable-breakpoints', label: 'Enable Breakpoints', action: 'proxy:toggle-breakpoints', checked: props.breakpointsEnabled },
      { key: 'proxy-recording-settings', label: 'Recording Settings...', action: 'proxy:recording-settings', disabled: !props.hasProxySettings },
    ],
  },
  {
    key: 'tools',
    label: 'Tools',
    items: [
      { key: 'tools-no-caching', label: 'No Caching', action: 'tools:toggle-no-caching', checked: props.noCachingEnabled },
      { key: 'tools-block-cookies', label: 'Block Cookies', action: 'tools:toggle-block-cookies', checked: props.blockCookiesEnabled },
      { key: 'tools-sep-1', separator: true },
      { key: 'tools-map-remote', label: 'Map Remote...', action: 'tools:map-remote', disabled: true },
      { key: 'tools-map-local', label: 'Map Local...', action: 'tools:map-local', disabled: true },
      { key: 'tools-rewrite', label: 'Rewrite...', action: 'tools:rewrite', disabled: true },
      { key: 'tools-block-list', label: 'Block List...', action: 'tools:block-list', disabled: true },
      { key: 'tools-dns-spoofing', label: 'DNS Spoofing...', action: 'tools:dns-spoofing', disabled: true },
      { key: 'tools-mirror', label: 'Mirror...', action: 'tools:mirror', disabled: true },
      { 
        key: 'tools-auto-save', 
        label: 'Auto Save', 
        children: [
          { key: 'tools-auto-save-enable', label: 'Enable Auto Save', action: 'tools:toggle-auto-save', checked: props.autoSaveState },
          { key: 'tools-auto-save-settings', label: 'Settings...', action: 'tools:auto-save-settings' }
        ]
      },
      { key: 'tools-client-process', label: 'Client Process...', action: 'tools:client-process', disabled: true },
      { key: 'tools-sep-2', separator: true },
      { 
        key: 'tools-repeat', 
        label: 'Repeat', 
        action: 'tools:repeat-request', 
        disabled: !props.hasSelectedRequest 
      },
      { 
        key: 'tools-repeat-advanced', 
        label: 'Repeat Advanced...', 
        action: 'tools:repeat-advanced', 
        disabled: !props.hasSelectedRequest 
      },
      { key: 'tools-edit', label: 'Edit...', action: 'tools:edit-request', disabled: !props.hasSelectedRequest },
      { key: 'tools-validate', label: 'Validate', action: 'tools:validate', disabled: !props.hasSelectedRequest },
      { key: 'tools-sep-3', separator: true },
      { key: 'tools-command-line', label: 'Command-line Tools...', action: 'tools:command-line', disabled: true },
    ],
  },
  {
    key: 'window',
    label: 'Window',
    items: [
      { key: 'window-session', label: 'Session 1', action: 'window:switch-session-1', checked: true },
      { key: 'window-sep-1', separator: true },
      { key: 'window-breakpoints', label: 'Breakpoints', action: 'window:breakpoints', disabled: true },
      { key: 'window-log', label: 'Log', action: 'window:log', disabled: true },
      { key: 'window-sep-2', separator: true },
      { key: 'window-minimize', label: 'Minimize', action: 'window:minimize' },
      { key: 'window-maximize', label: props.isMaximised ? 'Restore' : 'Zoom', action: 'window:toggle-maximize' },
      { key: 'window-bring-front', label: 'Bring All to Front', action: 'window:bring-front', disabled: true },
      { key: 'window-sep-3', separator: true },
      { key: 'window-detach-request-viewer', label: 'Detach Request Viewer', action: 'window:detach-viewer', disabled: true },
      { key: 'window-close', label: 'Close', action: 'window:close' },
    ],
  },
  {
    key: 'help',
    label: 'Help',
    items: [
      { key: 'help-local-ip', label: 'Local IP Address', action: 'help:local-ip' },
      { key: 'help-install-cert', label: 'Install Root Certificate', action: 'help:install-cert' },
      { key: 'help-sep-05', separator: true },
      { key: 'help-check-updates', label: 'Check for Updates', action: 'help:check-updates' },
      { key: 'help-register', label: 'Register PacketMind...', action: 'help:register', disabled: true },
      { key: 'help-sep-1', separator: true },
      { key: 'help-about', label: 'About PacketMind', action: 'help:about' },
    ],
  },
]);

const closeMenu = () => {
  openMenuKey.value = null;
  hoveredItemKey.value = null;
};

const toggleMenu = (key: string) => {
  if (openMenuKey.value === key) {
    openMenuKey.value = null;
    hoveredItemKey.value = null;
  } else {
    openMenuKey.value = key;
    hoveredItemKey.value = null;
  }
};

const handleGroupEnter = (key: string) => {
  if (openMenuKey.value) {
    openMenuKey.value = key;
    hoveredItemKey.value = null;
  }
};

const handleItemClick = (item: MenuItem) => {
  if (item.disabled) return;
  if (item.children) {
    // If clicking a parent item with children, don't close, maybe toggle
    hoveredItemKey.value = item.key;
    return;
  }
  if (!item.action) return;
  emit('action', item.action);
  closeMenu();
};

const handleDocumentClick = (event: MouseEvent) => {
  const target = event.target as HTMLElement | null;
  if (!target?.closest('[data-application-menu-root="true"]')) {
    closeMenu();
  }
};

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    closeMenu();
  }
};

onMounted(() => {
  document.addEventListener('click', handleDocumentClick);
  document.addEventListener('keydown', handleEscape);
});

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick);
  document.removeEventListener('keydown', handleEscape);
});
</script>

<style module>
.menuBar {
  display: flex;
  align-items: stretch;
  gap: 0;
  min-width: 0;
  height: 100%;
  --wails-draggable: no-drag;
}

.menuGroup {
  position: relative;
  height: 100%;
}

.menuTrigger {
  height: 100%;
  border: none;
  background: transparent;
  color: #1f1f1f;
  font-size: 12px;
  padding: 0 10px;
  min-width: 42px;
  cursor: default;
}

.menuTrigger:hover,
.menuTriggerActive {
  background: #e8e8e8;
}

.dropdown {
  position: absolute;
  top: calc(100% - 1px);
  left: 0;
  min-width: 226px;
  background: #f0f0f0;
  border: 1px solid #9e9e9e;
  box-shadow: 1px 1px 0 rgba(255, 255, 255, 0.75), 2px 2px 8px rgba(0, 0, 0, 0.15);
  padding: 4px 0;
  z-index: 1000;
}

.subDropdown {
  position: absolute;
  top: -5px; /* Aligned with top padding of dropdown */
  left: 100%;
  min-width: 226px;
  background: #f0f0f0;
  border: 1px solid #9e9e9e;
  box-shadow: 1px 1px 0 rgba(255, 255, 255, 0.75), 2px 2px 8px rgba(0, 0, 0, 0.15);
  padding: 4px 0;
  z-index: 1001;
}

.divider {
  height: 1px;
  margin: 4px 10px;
  background: #c4c4c4;
}

.menuItemWrapper {
  position: relative;
  width: 100%;
}

.menuItem {
  width: 100%;
  border: none;
  background: transparent;
  min-height: 24px;
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 3px 10px 3px 8px;
  text-align: left;
  color: #202020;
  cursor: default;
  font-size: 12px;
}

.menuItemWrapper:hover > .menuItem:not(:disabled) {
  background: #2d72d2;
  color: #fff;
}

.menuItemWrapper:hover > .menuItem:not(:disabled) .itemHint,
.menuItemWrapper:hover > .menuItem:not(:disabled) .itemArrow {
  color: #fff;
}

.menuItem:hover:not(:disabled) {
  background: #2d72d2;
  color: #fff;
}

.menuItemDisabled {
  color: #8c8c8c;
}

.checkSlot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 12px;
  font-size: 12px;
}

.itemLabel {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.itemHint {
  color: inherit;
  opacity: 0.72;
  font-size: 11px;
  white-space: nowrap;
}

.itemArrow {
  font-size: 10px;
  margin-left: 8px;
  color: #555;
}
.menuItemDisabled .itemArrow {
  color: #a0a0a0;
}
</style>

