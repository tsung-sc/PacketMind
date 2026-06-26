import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import type { Request } from '@/types'
import { requestApi } from '@/api/wails'

const RECENT_NODE_HIGHLIGHT_MS = 1800

export const isRequestPending = (req: Request): boolean => {
  return req.status_code === 0 && !req.error && req.duration === 0
}

const normalizeKeyword = (value?: string) => (value || '').trim().toLowerCase()

const splitPathSegments = (req: Request) => {
  const rawPath = (req.path || '/').replace(/^\/+|\/+$/g, '')
  const segments = rawPath ? rawPath.split('/').filter(Boolean) : []
  const query = req.query_string ? `?${req.query_string}` : ''

  if (!segments.length) {
    return [query || '/']
  }

  if (query) {
    segments[segments.length - 1] = segments[segments.length - 1] + query
  }

  return segments
}

const matchesFilters = (
  req: Request,
  filters: {
    session_id?: string
    host?: string
    method?: string
    search?: string
  },
) => {
  if (filters.session_id && req.session_id !== filters.session_id) {
    return false
  }

  if (filters.host && req.host !== filters.host) {
    return false
  }

  if (filters.method && req.method !== filters.method) {
    return false
  }

  const keyword = normalizeKeyword(filters.search)
  if (!keyword) {
    return true
  }

  const searchTargets = [req.host, req.path]
    .map((item) => normalizeKeyword(item))
    .join(' ')
  return searchTargets.includes(keyword)
}

const buildNodeKeys = (req: Request): string[] => {
  const host = req.host || 'unknown'
  const keys: string[] = [`host:${host}`]
  const segments = splitPathSegments(req)
  let parentKey = `host:${host}`

  segments.forEach((segment, index) => {
    const segmentKey = `${parentKey}/${segment}`
    keys.push(segmentKey)

    if (index === segments.length - 1) {
      keys.push(`${segmentKey}#${req.id}`)
      return
    }

    parentKey = segmentKey
  })

  return keys
}

export const useRequestStore = defineStore('request', () => {
  const requests = ref<Request[]>([])
  const selectedId = ref<string | null>(null)
  const loading = ref(false)
  const total = ref(0)
  const error = ref<string | null>(null)
  const recentNodeKeys = reactive(new Set<string>())
  const filters = ref<{
    session_id?: string
    host?: string
    method?: string
    search?: string
  }>({})
  const highlightTimers = new Map<string, ReturnType<typeof setTimeout>>()

  const selectedRequest = computed(() => 
    requests.value.find(r => r.id === selectedId.value) || null
  )

  const fetchRequests = async (page: number = 1, pageSize: number = 50) => {
    loading.value = true
    error.value = null
    try {
      const response = await requestApi.list(filters.value)
      if (response.code === 0 && response.data) {
        requests.value = response.data.items
        total.value = response.data.total
      } else {
        error.value = response.message || 'Failed to fetch requests'
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch requests'
    } finally {
      loading.value = false
    }
  }

  const selectRequest = (id: string | null) => {
    selectedId.value = id
  }

  const ensureRequestLoaded = async (id: string) => {
    const existing = requests.value.find((req) => req.id === id)
    if (existing) {
      return existing
    }

    const sessionId = filters.value.session_id || ''
    const response = await requestApi.get(sessionId, id)
    if (response.code !== 0 || !response.data) {
      throw new Error(response.message || 'Failed to load request detail')
    }

    const fetched = response.data
    requests.value.unshift(fetched)
    total.value = Math.max(total.value, requests.value.length)
    return fetched
  }

  const selectRequestById = async (id: string | null) => {
    if (!id) {
      selectedId.value = null
      return null
    }

    await ensureRequestLoaded(id)
    selectedId.value = id
    return requests.value.find((req) => req.id === id) || null
  }

  const setFilters = (newFilters: Partial<typeof filters.value>) => {
    filters.value = { ...filters.value, ...newFilters }
    fetchRequests(1)
  }

  const addRequest = (req: Request) => {
    if (!matchesFilters(req, filters.value)) {
      return
    }

    requests.value.unshift(req)
    total.value++

    const nodeKeys = buildNodeKeys(req)
    nodeKeys.forEach((key) => {
      recentNodeKeys.add(key)
      const previousTimer = highlightTimers.get(key)
      if (previousTimer) {
        clearTimeout(previousTimer)
      }

      const timer = setTimeout(() => {
        recentNodeKeys.delete(key)
        highlightTimers.delete(key)
      }, RECENT_NODE_HIGHLIGHT_MS)

      highlightTimers.set(key, timer)
    })
  }

  const updateRequest = (updated: Request) => {
    const index = requests.value.findIndex((req) => req.id === updated.id)
    if (index === -1) {
      return
    }

    requests.value.splice(index, 1, updated)
  }

  const clearRequests = () => {
    requests.value = []
    selectedId.value = null
    total.value = 0
    recentNodeKeys.clear()
    highlightTimers.forEach((timer) => clearTimeout(timer))
    highlightTimers.clear()
  }

  return {
    requests,
    selectedId,
    selectedRequest,
    loading,
    total,
    error,
    recentNodeKeys,
    filters,
    fetchRequests,
    selectRequest,
    selectRequestById,
    ensureRequestLoaded,
    setFilters,
    addRequest,
    updateRequest,
    isRequestPending,
    clearRequests,
  }
})
