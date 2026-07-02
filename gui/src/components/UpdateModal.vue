<template>
  <a-modal
    :open="open"
    title="Software Update"
    :width="480"
    :footer="null"
    :maskClosable="false"
    @cancel="closeModal"
    wrapClassName="packetmind-update-modal"
  >
    <div :class="$style.container">
      <div v-if="checking" :class="$style.stateView">
        <a-spin />
        <div :class="$style.stateText">Checking for updates...</div>
      </div>

      <div v-else-if="error" :class="$style.stateView">
        <a-alert type="error" :message="error" show-icon />
        <div :class="$style.actions">
          <a-button type="primary" @click="checkForUpdate">Retry</a-button>
          <a-button @click="closeModal">Cancel</a-button>
        </div>
      </div>

      <div v-else-if="updateReady" :class="$style.stateView">
        <div :class="$style.readyText">
          <strong>Installer downloaded!</strong><br />
          Open the installer to update PacketMind.
          <div v-if="downloadedUpdate?.path" :class="$style.downloadedPath">{{ downloadedUpdate.path }}</div>
        </div>
        <div :class="$style.actions">
          <a-button @click="showDownloadedUpdate">Show in Folder</a-button>
          <a-button type="primary" @click="openDownloadedUpdate">Open Installer</a-button>
          <a-button @click="closeModal">Later</a-button>
        </div>
      </div>

      <div v-else-if="downloading" :class="$style.stateView">
        <div :class="$style.stateText">Downloading update...</div>
        <a-progress :percent="progress?.percent || 0" status="active" />
        <div :class="$style.progressDetail" v-if="progress">
          {{ formatBytes(progress.downloaded) }} / {{ formatBytes(progress.total) }}
        </div>
      </div>

      <div v-else-if="updateInfo?.has_update" :class="$style.updateAvailable">
        <div :class="$style.versionBadge">
          A new version is available: <strong>{{ updateInfo.latest_version }}</strong>
          <span :class="$style.currentVersion">(Current: {{ updateInfo.current_version }})</span>
        </div>

        <div :class="$style.releaseNotes">
          <div :class="$style.releaseNotesTitle">Release Notes:</div>
          <div :class="$style.releaseNotesContent" v-html="formatReleaseNotes(updateInfo.release_notes)"></div>
        </div>

        <div :class="$style.actions">
          <a-button @click="closeModal">Skip</a-button>
          <a-button type="primary" @click="performUpdate">Download Installer</a-button>
        </div>
      </div>

      <div v-else-if="updateInfo && !updateInfo.has_update" :class="$style.stateView">
        <div :class="$style.upToDateText">
          <strong>You're up to date!</strong><br />
          PacketMind {{ updateInfo.current_version }} is currently the newest version available.
        </div>
        <div :class="$style.actions">
          <a-button type="primary" @click="closeModal">OK</a-button>
        </div>
      </div>

      <div v-else :class="$style.stateView">
        <!-- Initial empty state before auto-check -->
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useUpdater } from '@/composables/useUpdater'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const {
  checking,
  downloading,
  updateInfo,
  progress,
  error,
  updateReady,
  downloadedUpdate,
  checkForUpdate,
  performUpdate,
  openDownloadedUpdate,
  showDownloadedUpdate,
  reset
} = useUpdater()

watch(() => props.open, (newVal) => {
  if (newVal) {
    checkForUpdate()
  } else {
    // wait for modal exit animation before reset
    setTimeout(() => reset(), 300)
  }
})

const closeModal = () => {
  emit('close')
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatReleaseNotes = (notes: string): string => {
  // basic markdown to html conversion for release notes
  return notes
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    .replace(/\[(.*?)\]\((.*?)\)/g, '<a href="$2" target="_blank">$1</a>')
}
</script>

<style module>
.container {
  padding: 16px 0 0;
  color: #333;
}

.stateView {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 0;
  min-height: 160px;
  width: 100%;
}

.stateText {
  margin-top: 16px;
  font-size: 14px;
  color: #555;
}

.readyText, .upToDateText {
  text-align: center;
  font-size: 14px;
  line-height: 1.5;
  color: #333;
  margin-bottom: 24px;
}

.downloadedPath {
  max-width: 420px;
  margin-top: 8px;
  padding: 6px 8px;
  border: 1px solid #d9d9d9;
  background: #f7f7f7;
  color: #666;
  font-size: 12px;
  word-break: break-all;
}

.progressDetail {
  margin-top: 8px;
  font-size: 12px;
  color: #888;
}

.updateAvailable {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.versionBadge {
  font-size: 14px;
  padding: 12px;
  background: #f5f5f5;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
}

.currentVersion {
  color: #888;
  margin-left: 8px;
  font-size: 13px;
}

.releaseNotes {
  max-height: 240px;
  overflow-y: auto;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fafafa;
}

.releaseNotesTitle {
  padding: 8px 12px;
  background: #f0f0f0;
  border-bottom: 1px solid #d9d9d9;
  font-weight: 600;
  font-size: 13px;
}

.releaseNotesContent {
  padding: 12px;
  font-size: 13px;
  line-height: 1.5;
  color: #444;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  width: 100%;
}
</style>