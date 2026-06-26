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

  const reset = () => {
    checking.value = false
    downloading.value = false
    updateInfo.value = null
    progress.value = null
    error.value = null
    updateReady.value = false
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
      const response = await updaterApi.performUpdate()
      if (response.code !== 0) {
        error.value = response.message || 'Failed to start download'
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

  const handleDone = (success: boolean) => {
    downloading.value = false
    if (success) {
      updateReady.value = true
    } else {
      error.value = 'Update failed during installation'
    }
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
    checkForUpdate,
    performUpdate,
    reset
  }
}
