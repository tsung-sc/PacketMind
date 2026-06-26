import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import { fetchModels, agentApi, configApi } from '@/api/wails'
import type { AgentModel, AnalysisRequest, AgentEvent, ProvenanceChain, TokenUsage, SessionContextStats, ModelListResponse, ModelsByProvider } from '@/types'

export interface Message {
  id: string
  role:
    | 'user'
    | 'assistant'
    | 'agent_thought'
    | 'agent_action'
    | 'agent_observation'
    | 'agent_error'
    | 'agent_decision'
    | 'agent_related'
    | 'agent_provenance'
    | 'agent_retry'
  content: string
  timestamp: number
  toolName?: string
  arguments?: string
  requestIds?: string[]
  depth?: number
  provenance?: ProvenanceChain
  errorCategory?: string
  errorToolName?: string
  errorTimeout?: string
  errorRecovered?: boolean
  specialistName?: string
  retryAttempt?: number
  retryMax?: number
}

export interface PendingQuery {
  requestId?: string
  query: string
  sessionId?: string
}

interface SessionSnapshot {
  messages: Message[]
  messageSequence: number
  traceSequence: number
  tokensUsed: number
  tokenUsage: TokenUsage | null
  streaming: boolean
  analysisId: string | null
  currentContent: string
  agentEvents: AgentEvent[]
  currentToolCalls: number
  currentDepth: number
  relatedRequestIds: string[]
  traceExpanded: boolean
  provenanceChain: ProvenanceChain | null
  pendingQueue: PendingQuery[]
}

export const useAgentStore = defineStore('agent', () => {
  const messages = ref<Message[]>([])
  const messageSequence = ref<number>(0)
  const traceSequence = ref<number>(0)
  const streaming = ref<boolean>(false)
  const currentContent = ref<string>('')
  const analysisId = ref<string | null>(null)
  const pendingQueue = ref<PendingQuery[]>([])
  const models = ref<AgentModel[]>([])
  const selectedModelId = ref<string>('')
  const defaultModel = ref<string>('')
  const modelsLoaded = ref<boolean>(false)
  const providers = ref<import('@/types').ProviderInfo[]>([])
  const modelsGrouped = ref<import('@/types').ModelsByProvider[]>([])
  const agentEvents = ref<AgentEvent[]>([])
  const currentToolCalls = ref<number>(0)
  const currentDepth = ref<number>(0)
  const relatedRequestIds = ref<Set<string>>(new Set())
  const traceExpanded = ref<boolean>(false)
  const provenanceChain = ref<ProvenanceChain | null>(null)
  const tokenUsage = ref<TokenUsage | null>(null)
  const tokensUsed = ref<number>(0)
  const chatSessionId = ref<string>('')
  const sessionContext = ref<SessionContextStats | null>(null)
  const lastUserQuery = ref<string>('')

  const sessionMessageCache = new Map<string, SessionSnapshot>()

  const agentTraceChain = computed(() => {
    return messages.value.filter(msg =>
      ['agent_thought', 'agent_action', 'agent_observation', 'agent_error', 'agent_decision', 'agent_retry'].includes(msg.role)
    )
  })

  const hasAgentTrace = computed(() => agentTraceChain.value.length > 0)
  const uniqueRequestIds = computed<string[]>(() => Array.from(relatedRequestIds.value))
  const hasPendingQuery = computed(() => pendingQueue.value.length > 0)

  const fallbackProviderName = (provider: string) => {
    const normalized = provider.trim()
    if (!normalized) return 'Unknown'
    return normalized.charAt(0).toUpperCase() + normalized.slice(1)
  }

  const sortModels = (items: AgentModel[]) => {
    return [...items].sort((a, b) => {
      const nameCompare = a.name.localeCompare(b.name)
      if (nameCompare !== 0) return nameCompare
      return a.id.localeCompare(b.id)
    })
  }

  const buildFallbackGrouping = (items: AgentModel[]): ModelsByProvider[] => {
    const groupedMap = new Map<string, AgentModel[]>()

    items.forEach((model) => {
      const provider = model.provider || 'unknown'
      const existing = groupedMap.get(provider) || []
      existing.push(model)
      groupedMap.set(provider, existing)
    })

    return Array.from(groupedMap.entries())
      .sort(([providerA], [providerB]) => providerA.localeCompare(providerB))
      .map(([provider, providerModels]) => ({
        provider,
        provider_name: fallbackProviderName(provider),
        models: sortModels(providerModels),
      }))
  }

  const normalizeGroupedModels = (grouped: ModelsByProvider[] | undefined, flatModels: AgentModel[]) => {
    if (grouped && grouped.length > 0) {
      return grouped
        .map((group) => ({
          ...group,
          models: sortModels(group.models || []),
        }))
        .sort((a, b) => a.provider_name.localeCompare(b.provider_name))
    }

    if (flatModels.length > 0) {
      return buildFallbackGrouping(flatModels)
    }

    return []
  }

  const resolveSelectedModelId = (
    data: ModelListResponse,
    flatModels: AgentModel[],
    grouped: ModelsByProvider[],
  ) => {
    const availableIds = new Set(flatModels.map((model) => model.id))
    const isValidModelId = (modelId?: string) => !!modelId && availableIds.has(modelId)

    if (isValidModelId(selectedModelId.value)) {
      return selectedModelId.value
    }

    if (isValidModelId(data.active_model)) {
      return data.active_model as string
    }

    if (isValidModelId(data.default_model)) {
      return data.default_model
    }

    if (data.active_provider) {
      const activeGroup = grouped.find((group) => group.provider === data.active_provider)
      const activeGroupModelId = activeGroup?.models?.[0]?.id
      if (isValidModelId(activeGroupModelId)) {
        return activeGroupModelId as string
      }
    }

    return grouped[0]?.models?.[0]?.id || flatModels[0]?.id || ''
  }

  const loadModels = async (force = false) => {
    if (modelsLoaded.value && !force) return
    try {
      const data = await fetchModels()
      if (data) {
        const nextModels = sortModels(data.models || [])
        const nextGrouped = normalizeGroupedModels(data.grouped, nextModels)

        models.value = nextModels
        defaultModel.value = data.default_model || ''
        modelsGrouped.value = nextGrouped
        selectedModelId.value = resolveSelectedModelId(data, nextModels, nextGrouped)
      }
      modelsLoaded.value = true
    } catch (error) {
      console.error('Failed to load models:', error)
    }
  }

  const loadProviders = async () => {
    try {
      const response = await agentApi.listProviders()
      if (response && response.data) {
        providers.value = response.data.providers || []
      }
    } catch (error) {
      console.error('Failed to load providers:', error)
    }
  }

  const activeProvider = computed(() => {
    const model = models.value.find((m: AgentModel) => m.id === selectedModelId.value)
    return model?.provider || ''
  })

  const switchModel = async (modelId: string) => {
    const model = models.value.find((m: AgentModel) => m.id === modelId)
    if (!model) return
    selectedModelId.value = modelId
    try {
      await configApi.updateAgent({
        provider: model.provider,
        model: modelId,
      })
    } catch (e) {
      console.error('Failed to persist model switch:', e)
    }
  }

  const addModel = async (data: { provider: string; id: string; name: string; context: number; output: number }) => {
    const response = await configApi.upsertModel(data)
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to save model')
    }
    await loadModels(true)
    await loadProviders()
    message.success('Model saved')
  }

  const removeModel = async (providerId: string, modelId: string) => {
    const response = await configApi.deleteModel(providerId, modelId)
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to delete model')
    }
    if (selectedModelId.value === modelId) {
      selectedModelId.value = ''
    }
    await loadModels(true)
    await loadProviders()
    message.success('Model deleted')
  }

  const updateRuntimeParams = async (maxTokens: number, temperature: number) => {
    const response = await configApi.updateAgent({
      max_tokens: maxTokens,
      temperature,
    })
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to update runtime parameters')
    }
  }

  const setSelectedModel = (modelId: string) => {
    selectedModelId.value = modelId
  }

  const getSelectedModel = (): AgentModel | undefined => {
    return models.value.find((m: AgentModel) => m.id === selectedModelId.value)
  }

  const nextMessageId = (prefix: 'msg' | 'trace') => {
    if (prefix === 'trace') {
      traceSequence.value += 1
      return `trace_${Date.now()}_${traceSequence.value}`
    }

    messageSequence.value += 1
    return `msg_${Date.now()}_${messageSequence.value}`
  }

  const addMessage = (
    role: Message['role'], 
    content: string, 
    options?: {
      toolName?: string
      arguments?: string
      requestIds?: string[]
      depth?: number
      provenance?: ProvenanceChain
      errorCategory?: string
      errorToolName?: string
      errorTimeout?: string
      errorRecovered?: boolean
      specialistName?: string
      retryAttempt?: number
      retryMax?: number
    }
  ) => {
    messages.value.push({
      id: nextMessageId('msg'),
      role,
      content,
      timestamp: Date.now(),
      ...options
    })
  }

  const startStreaming = (id: string, userMessage: string) => {
    streaming.value = true
    currentContent.value = ''
    analysisId.value = id
    agentEvents.value = []
    traceExpanded.value = false
    currentToolCalls.value = 0
    currentDepth.value = 0
    relatedRequestIds.value = new Set()
    provenanceChain.value = null
    tokenUsage.value = null
    tokensUsed.value = 0
    addMessage('user', userMessage)
  }

  const appendContent = (content: string) => {
    currentContent.value += content
  }

  const finishStreaming = () => {
    if (currentContent.value) {
      addMessage('assistant', currentContent.value)
    }
    streaming.value = false
    if (agentTraceChain.value.length > 0) {
      traceExpanded.value = false
    }
    currentContent.value = ''
    analysisId.value = null

    saveCurrentSnapshot()
    
    if (pendingQueue.value.length > 0) {
      setTimeout(() => processNextPending(), 100)
    }
  }

  const clearMessages = () => {
    const sid = chatSessionId.value
    if (sid) {
      sessionMessageCache.delete(sid)
    }
    messages.value = []
    messageSequence.value = 0
    traceSequence.value = 0
    currentContent.value = ''
    agentEvents.value = []
    traceExpanded.value = false
    currentToolCalls.value = 0
    currentDepth.value = 0
    relatedRequestIds.value = new Set()
    provenanceChain.value = null
    tokenUsage.value = null
    tokensUsed.value = 0
    pendingQueue.value = []
    sessionContext.value = null
  }

  const setTraceExpanded = (expanded: boolean) => {
    traceExpanded.value = expanded
  }

  const toggleTraceExpanded = () => {
    traceExpanded.value = !traceExpanded.value
  }

  const addToPendingQueue = (requestId: string | undefined, query: string, sessionId?: string) => {
    pendingQueue.value.push({ requestId, query, sessionId })
  }

  const processNextPending = async () => {
    if (pendingQueue.value.length === 0) return
    const next = pendingQueue.value.shift()
    if (next) {
      await analyzeWithQuery(next.requestId, next.query, next.sessionId)
    }
  }

  const stopAnalysis = async () => {
    if (!analysisId.value || !streaming.value) return
    
    try {
      await agentApi.cancel(analysisId.value)
    } catch (error) {
      console.error('[AgentStore] Failed to cancel analysis:', error)
    }
    
    finishStreaming()
  }

  const addAgentEvent = (event: AgentEvent) => {
    agentEvents.value.push(event)
    currentToolCalls.value = event.tool_calls ?? 0
    currentDepth.value = event.depth

    if (event.request_ids && event.request_ids.length > 0) {
      event.request_ids.forEach(id => relatedRequestIds.value.add(id))
    }

    switch (event.type) {
      case 'start':
        addMessage('agent_thought', event.content || 'Starting analysis...', {
          depth: event.depth,
          specialistName: event.specialist_name,
        })
        break
      
      case 'thought':
        if (event.content) {
          const last = messages.value[messages.value.length - 1]
          if (last && last.role === 'agent_thought') {
            last.content += event.content
          } else {
            addMessage('agent_thought', event.content, {
              depth: event.depth,
              specialistName: event.specialist_name,
            })
          }
        }
        break
      
      case 'decision':
        if (event.content) {
          addMessage('agent_decision', event.content, {
            depth: event.depth,
            specialistName: event.specialist_name,
          })
        }
        break
      
      case 'action':
        addMessage('agent_action', `Calling tool: ${event.tool_name}`, {
          toolName: event.tool_name,
          arguments: event.arguments,
          depth: event.depth,
          specialistName: event.specialist_name,
        })
        break
      
      case 'observation': {
        const isErrorObservation = !!event.error_category || !!event.error_tool_name || !!event.error_timeout || !!event.error_recovered
        const preview = event.content
          ? (event.content.length > 200 ? event.content.substring(0, 200) + '...' : event.content)
          : 'No result preview'
        addMessage(isErrorObservation ? 'agent_error' : 'agent_observation', preview, {
          toolName: event.tool_name,
          requestIds: event.request_ids,
          depth: event.depth,
          errorCategory: event.error_category,
          errorToolName: event.error_tool_name,
          errorTimeout: event.error_timeout,
          errorRecovered: event.error_recovered,
          specialistName: event.specialist_name,
        })
        break
      }
      
      case 'warning':
        addMessage('agent_thought', `⚠️ ${event.content}`, {
          depth: event.depth,
          specialistName: event.specialist_name,
        })
        break
      
      case 'final':
        currentContent.value = event.content || ''
        break

      case 'text_delta':
        currentContent.value += event.content || ''
        break

      case 'provider_retry':
        addMessage('agent_retry', event.content || '正在重试...', {
          depth: event.depth,
          retryAttempt: event.retry_attempt,
          retryMax: event.retry_max,
        })
        break
    }
  }

  const ensureChatSessionId = () => {
    return chatSessionId.value
  }

  const saveCurrentSnapshot = () => {
    const sid = chatSessionId.value
    if (!sid) return
    sessionMessageCache.set(sid, {
      messages: [...messages.value],
      messageSequence: messageSequence.value,
      traceSequence: traceSequence.value,
      tokensUsed: tokensUsed.value,
      tokenUsage: tokenUsage.value ? { ...tokenUsage.value } : null,
      streaming: streaming.value,
      analysisId: analysisId.value,
      currentContent: currentContent.value,
      agentEvents: [...agentEvents.value],
      currentToolCalls: currentToolCalls.value,
      currentDepth: currentDepth.value,
      relatedRequestIds: Array.from(relatedRequestIds.value),
      traceExpanded: traceExpanded.value,
      provenanceChain: provenanceChain.value,
      pendingQueue: [...pendingQueue.value],
    })
  }

  const restoreSnapshot = (sid: string) => {
    const snapshot = sessionMessageCache.get(sid)
    if (snapshot) {
      messages.value = [...snapshot.messages]
      messageSequence.value = snapshot.messageSequence
      traceSequence.value = snapshot.traceSequence
      tokensUsed.value = snapshot.tokensUsed
      tokenUsage.value = snapshot.tokenUsage
      streaming.value = snapshot.streaming
      analysisId.value = snapshot.analysisId
      currentContent.value = snapshot.currentContent
      agentEvents.value = [...snapshot.agentEvents]
      currentToolCalls.value = snapshot.currentToolCalls
      currentDepth.value = snapshot.currentDepth
      relatedRequestIds.value = new Set(snapshot.relatedRequestIds)
      traceExpanded.value = snapshot.traceExpanded
      provenanceChain.value = snapshot.provenanceChain
      pendingQueue.value = [...snapshot.pendingQueue]
    } else {
      messages.value = []
      messageSequence.value = 0
      traceSequence.value = 0
      tokensUsed.value = 0
      tokenUsage.value = null
      streaming.value = false
      analysisId.value = null
      currentContent.value = ''
      agentEvents.value = []
      currentToolCalls.value = 0
      currentDepth.value = 0
      relatedRequestIds.value = new Set()
      traceExpanded.value = false
      provenanceChain.value = null
      pendingQueue.value = []
    }
    sessionContext.value = null
  }

  const restoreFromBackend = async (sid: string) => {
    try {
      const response = await agentApi.getChatHistory(sid)
      if (response.code === 0 && response.data?.messages?.length) {
        const restored: Message[] = response.data.messages.map((m, idx) => ({
          id: `restored_${idx}_${Date.now()}`,
          role: (m.role === 'user' || m.role === 'assistant') ? m.role : 'assistant' as Message['role'],
          content: m.content,
          timestamp: m.created_at ? new Date(m.created_at).getTime() : Date.now(),
        }))
        messages.value = restored
        messageSequence.value = restored.length
        traceSequence.value = 0
        tokensUsed.value = 0
        tokenUsage.value = null
        streaming.value = false
        analysisId.value = null
        currentContent.value = ''
        agentEvents.value = []
        currentToolCalls.value = 0
        currentDepth.value = 0
        relatedRequestIds.value = new Set()
        traceExpanded.value = false
        provenanceChain.value = null
        pendingQueue.value = []
        sessionContext.value = null
        saveCurrentSnapshot()
        return true
      }
    } catch (e) {
      console.error('[AgentStore] Failed to load chat history from backend:', e)
    }
    return false
  }

  const bindSession = async (captureSessionId: string) => {
    if (chatSessionId.value === captureSessionId) return
    saveCurrentSnapshot()
    chatSessionId.value = captureSessionId

    const cached = sessionMessageCache.get(captureSessionId)
    if (cached) {
      restoreSnapshot(captureSessionId)
    } else {
      await restoreFromBackend(captureSessionId)
      if (!sessionMessageCache.has(captureSessionId)) {
        restoreSnapshot(captureSessionId)
      }
    }
  }

  const switchSession = async (captureSessionId: string) => {
    await bindSession(captureSessionId)
  }

  const routeEventToSession = (sessionId: string, handler: () => void) => {
    if (!sessionId || sessionId === chatSessionId.value) {
      handler()
      return
    }

    const snapshot = sessionMessageCache.get(sessionId)
    if (!snapshot) {
      return
    }

    if (chatSessionId.value && chatSessionId.value !== sessionId) {
      saveCurrentSnapshot()
    }

    const savedSessionId = chatSessionId.value
    chatSessionId.value = sessionId
    restoreSnapshot(sessionId)
    handler()
    saveCurrentSnapshot()

    chatSessionId.value = savedSessionId
    restoreSnapshot(savedSessionId)
  }

  const invalidateSessionCache = (sessionId: string) => {
    if (!sessionId) return
    sessionMessageCache.delete(sessionId)
    if (chatSessionId.value === sessionId) {
      chatSessionId.value = ''
      clearMessages()
    }
  }

  const setAgentResult = (result: { 
    final_answer?: string
    depth_used?: number
    tool_calls?: number
    stopped_early?: boolean
    stop_reason?: string
    token_usage?: TokenUsage
    tokens_used?: number
    provenance?: ProvenanceChain | null
  }) => {
    const requestIds = Array.from(relatedRequestIds.value)
    if (requestIds.length > 0) {
      addMessage('agent_related', '', { requestIds })
    }

    if (result.provenance && result.provenance.links.length > 0) {
      addMessage('agent_provenance', '', { provenance: result.provenance })
    }

    if (result.final_answer) {
      currentContent.value = result.final_answer
    }
    tokenUsage.value = result.token_usage || null
    if (result.token_usage) {
      const usageWithTotal = result.token_usage as TokenUsage & { total_tokens?: number }
      if (typeof usageWithTotal.total_tokens === 'number') {
        tokensUsed.value = usageWithTotal.total_tokens
      } else {
        tokensUsed.value =
          (result.token_usage.input_tokens || 0) +
          (result.token_usage.output_tokens || 0) +
          (result.token_usage.cache_creation_input_tokens || 0) +
          (result.token_usage.cache_read_input_tokens || 0)
      }
    } else {
      tokensUsed.value = typeof result.tokens_used === 'number' ? result.tokens_used : 0
    }
    provenanceChain.value = result.provenance || null
    finishStreaming()
  }

  const analyzeParameter = async (params: {
    requestId: string
    fieldName: string
    fieldValue: string
    location: string
    sessionId?: string
  }) => {
    if (streaming.value) return

    const resolvedSessionId = ensureChatSessionId()

    const userMessage = `Analyze **${params.fieldName}** from ${params.location}:\n\`\`\`\n${params.fieldValue}\n\`\`\``
    startStreaming(`analysis_${Date.now()}`, userMessage)

    const request: AnalysisRequest = {
      request_id: params.requestId,
      target_field: params.fieldName,
      target_location: params.location,
      target_value: params.fieldValue,
      model_id: selectedModelId.value || undefined,
      session_id: resolvedSessionId,
    }

    try {
      const response = await agentApi.analyze(request)
      if (response.data?.analysis_id) {
        analysisId.value = response.data.analysis_id
      }
    } catch (error: any) {
      appendContent(`\n\n**Error:** ${error.message || 'Failed to start analysis'}`)
      finishStreaming()
    }
  }

  const analyzeWithQuery = async (requestId: string | undefined, query: string, sessionId?: string) => {
    if (streaming.value) return

    const resolvedSessionId = sessionId || ensureChatSessionId()
    lastUserQuery.value = query

    addMessage('user', query)

    streaming.value = true
    currentContent.value = ''
    analysisId.value = `query_${Date.now()}`
    agentEvents.value = []
    traceExpanded.value = false
    currentToolCalls.value = 0
    currentDepth.value = 0
    relatedRequestIds.value = new Set()
    provenanceChain.value = null
    tokenUsage.value = null
    tokensUsed.value = 0

    const request: AnalysisRequest = {
      request_id: requestId || undefined,
      target_field: 'custom',
      target_location: 'user_query',
      target_value: query,
      query: query,
      model_id: selectedModelId.value || undefined,
      session_id: resolvedSessionId,
    }

    try {
      const response = await agentApi.analyze(request)
      if (response.data?.analysis_id) {
        analysisId.value = response.data.analysis_id
      }
    } catch (error: any) {
      appendContent(`\n\n**Error:** ${error.message || 'Failed to start analysis'}`)
      finishStreaming()
    }
  }

  return {
    messages,
    streaming,
    currentContent,
    analysisId,
    models,
    selectedModelId,
    defaultModel,
    modelsLoaded,
    providers,
    modelsGrouped,
    activeProvider,
    agentEvents,
    agentTraceChain,
    currentToolCalls,
    currentDepth,
    relatedRequestIds,
    traceExpanded,
    provenanceChain,
    tokenUsage,
    tokensUsed,
    pendingQueue,
    sessionContext,
    lastUserQuery,
    hasAgentTrace,
    uniqueRequestIds,
    hasPendingQuery,
    loadModels,
    loadProviders,
    switchModel,
    addModel,
    removeModel,
    updateRuntimeParams,
    setSelectedModel,
    getSelectedModel,
    addMessage,
    startStreaming,
    appendContent,
    finishStreaming,
    clearMessages,
    setTraceExpanded,
    toggleTraceExpanded,
    addAgentEvent,
    setAgentResult,
    analyzeParameter,
    analyzeWithQuery,
    addToPendingQueue,
    processNextPending,
    stopAnalysis,
    ensureChatSessionId,
    bindSession,
    switchSession,
    routeEventToSession,
    invalidateSessionCache,
    fetchSessionContext: async (sessionId?: string) => {
      const resolvedSessionId = sessionId || ensureChatSessionId()
      try {
        sessionContext.value = await agentApi.getSessionContext(resolvedSessionId)
        return sessionContext.value
      } catch (error) {
        console.error('[AgentStore] Failed to fetch session context:', error)
        return null
      }
    },
  }
})
