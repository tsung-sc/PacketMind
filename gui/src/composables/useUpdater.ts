import { ref, onMounted, onUnmounted } from 'vue'
import { updaterApi } from '@/api/wails'
import type { UpdateInfo, UpdateProgress } from '@/types'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { isWailsRuntime } from '@/utils/wails'

export function useUpdater() {
  const checking = ref(false)
  const downloading = ref(false)
  const updateInfo = ref<UpdateInfo | null>(null)
  const progress = ref<UpdateProgress | null>(null)
  const error = ref<string | null>(null)
  const updateReady = ref(false)
  const downloadedUpdate = ref<{ version: string; path: string; name: string; size: number } | null>(null)

  const reset = () => {
    checking.value = false
    downloading.value = false
    updateInfo.value = null
    progress.value = null
    error.value = null
    updateReady.value = false
    downloadedUpdate.value = null
  }

  const checkForUpdate = async () => {
    reset()
    checking.value = true
    try {
      const response = await updaterApi.checkForUpdate()
      if (response.code === 0 && response.data) {
        updateInfo.value = response.data
      } else {
        error.value = response.message || 'Failed to check for updates'
      }
    } catch (e: any) {
      error.value = e?.message || 'Error checking for updates'
    } finally {
      checking.value = false
    }
  }

  const performUpdate = async () => {
    if (!updateInfo.value || !updateInfo.value.has_update) return
    
    error.value = null
    downloading.value = true
    progress.value = { downloaded: 0, total: 1, percent: 0 }
    
    try {
      const response = await updaterApi.downloadUpdate()
      if (response.code !== 0) {
        error.value = response.message || 'Failed to download update installer'
        downloading.value = false
      } else if (response.data) {
        downloadedUpdate.value = response.data
        updateReady.value = true
        downloading.value = false
      }
    } catch (e: any) {
      error.value = e?.message || 'Error starting update download'
      downloading.value = false
    }
  }

  const handleProgress = (data: UpdateProgress) => {
    downloading.value = true
    progress.value = data
  }

  const handleDone = (data: boolean | { success?: boolean; path?: string; name?: string; version?: string }) => {
    downloading.value = false
    const success = typeof data === 'boolean' ? data : data?.success !== false
    if (success) {
      updateReady.value = true
    } else {
      error.value = 'Update failed during download'
    }
  }

  const openDownloadedUpdate = async () => {
    if (!downloadedUpdate.value?.path) return
    const response = await updaterApi.openDownloadedUpdate(downloadedUpdate.value.path)
    if (response.code !== 0) error.value = response.message || 'Failed to open installer'
  }

  const showDownloadedUpdate = async () => {
    if (!downloadedUpdate.value?.path) return
    const response = await updaterApi.showDownloadedUpdate(downloadedUpdate.value.path)
    if (response.code !== 0) error.value = response.message || 'Failed to show installer in folder'
  }

  onMounted(() => {
    if (isWailsRuntime()) {
      EventsOn('update:progress', handleProgress)
      EventsOn('update:done', handleDone)
    }
  })

  onUnmounted(() => {
    if (isWailsRuntime()) {
      EventsOff('update:progress')
      EventsOff('update:done')
    }
  })

  return {
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
  }
}
