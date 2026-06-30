import { onMounted, onUnmounted } from 'vue'
import { useRequestStore } from '@/stores/requestStore'
import { useAgentStore } from '@/stores/agentStore'
import { isWailsRuntime } from '@/utils/wails'
import type { AgentEvent } from '@/types'

export function useWailsEvents() {
  const requestStore = useRequestStore()
  const agentStore = useAgentStore()
  let currentSessionId = ''
  let unbindRequest = () => {}
  let unbindRequestComplete = () => {}
  let unbindAnalysis = () => {}

  const loadRuntime = async () => {
    if (!isWailsRuntime()) {
      return null
    }
    return await import('../../wailsjs/runtime/runtime')
  }

  const handleNewRequest = (data: any) => {
    console.log('[Wails] New request:', data)
    requestStore.addRequest(data)
  }

  const handleCompleteRequest = (data: any) => {
    console.log('[Wails] Complete request:', data)
    requestStore.updateRequest(data)
  }

  const handleAgentAnalysis = (data: any) => {
    console.log('[Wails] Agent analysis:', data)
    const eventSessionId = data.session_id || ''

    // Error case
    if (data.error) {
      agentStore.routeEventToSession(eventSessionId, () => {
        agentStore.appendContent(`\n\n**Error:** ${data.error}`)
        if (data.error_category || data.error_tool_name || data.error_timeout || data.error_recovered) {
          const details: string[] = []
          if (data.error_category) details.push(`category=${data.error_category}`)
          if (data.error_tool_name) details.push(`tool=${data.error_tool_name}`)
          if (data.error_timeout) details.push(`timeout=${data.error_timeout}`)
          if (typeof data.error_recovered === 'boolean') details.push(`recovered=${data.error_recovered}`)
          if (details.length > 0) {
            agentStore.appendContent(`\n\n**Error Details:** ${details.join(', ')}`)
          }
        }
        agentStore.finishStreaming()
      })
      return
    }

    // Agent event
    if (data.agent_event) {
      agentStore.routeEventToSession(eventSessionId, () => {
        const eventData = data.agent_event
        const eventContent = eventData.type === 'text_delta'
          ? (eventData.content ?? data.content ?? '')
          : (eventData.content ?? '')
        const agentEvent: AgentEvent = {
          depth: eventData.depth || 0,
          type: eventData.type,
          content: eventContent,
          tool_name: eventData.tool_name || '',
          tool_call_id: eventData.tool_call_id || '',
          arguments: eventData.arguments || '',
          result: eventData.result || '',
          request_ids: eventData.request_ids || [],
          tool_calls: eventData.tool_calls || 0,
          created_at: eventData.created_at || new Date().toISOString(),
          specialist_name: eventData.specialist_name || '',
          error_category: eventData.error_category || '',
          error_tool_name: eventData.error_tool_name || '',
          error_timeout: eventData.error_timeout || '',
          error_recovered: eventData.error_recovered || false,
          retry_attempt: eventData.retry_attempt || 0,
          retry_max: eventData.retry_max || 0,
          intervention_id: eventData.intervention_id || '',
        }
        agentStore.addAgentEvent(agentEvent)
      })
      return
    }

    // Agent final result
    if (data.agent_result) {
      agentStore.routeEventToSession(eventSessionId, () => {
        agentStore.setAgentResult({
          final_answer: data.agent_result.final_answer || data.content,
          depth_used: data.agent_result.depth_used,
          tool_calls: data.agent_result.tool_calls,
          stopped_early: data.agent_result.stopped_early,
          stop_reason: data.agent_result.stop_reason,
          token_usage: data.agent_result.token_usage || undefined,
          tokens_used: typeof data.agent_result.tokens_used === 'number' ? data.agent_result.tokens_used : undefined,
          provenance: data.agent_result.provenance || null,
        })
      })
      return
    }

    // Done signal
    if (data.done) {
      console.log('[Wails] Agent analysis complete')
      agentStore.routeEventToSession(eventSessionId, () => {
        agentStore.finishStreaming()
      })
      return
    }

    if (data.cancelled) {
      agentStore.routeEventToSession(eventSessionId, () => {
        agentStore.finishStreaming()
      })
      return
    }

    // Legacy streaming content
    if (data.content) {
      agentStore.routeEventToSession(eventSessionId, () => {
        agentStore.appendContent(data.content)
      })
    }
  }

  onMounted(() => {
    void (async () => {
      const runtime = await loadRuntime()
      if (!runtime) {
        return
      }

      unbindRequest = runtime.EventsOn('request:new', handleNewRequest)
      unbindRequestComplete = runtime.EventsOn('request:complete', handleCompleteRequest)
      unbindAnalysis = runtime.EventsOn('agent:analysis', handleAgentAnalysis)
    })()
  })

  onUnmounted(() => {
    unbindRequest()
    unbindRequestComplete()
    unbindAnalysis()
  })

  const subscribeToSession = (sessionId: string) => {
    currentSessionId = sessionId
    // Wails events are broadcast to all listeners, no server-side subscription needed
    console.log('[Wails] Session context set to:', sessionId)
  }

  return {
    subscribeToSession,
  }
}
