<template>
  <div :class="$style.container">
    <div :class="$style.header">
      <span :class="$style.title">AGENT</span>
      <div :class="$style.headerActions">
        <a-popover
          trigger="click"
          placement="bottomRight"
          v-model:open="modelDropdownVisible"
          :overlayInnerStyle="{ padding: 0, borderRadius: '4px', overflow: 'hidden' }"
          :overlayClassName="$style.modelPopoverOverlay"
        >
          <div :class="[$style.modelSelectTrigger, !agentStore.modelsLoaded ? $style.modelSelectTriggerLoading : '']">
            <span v-if="!agentStore.modelsLoaded"><a-spin size="small" /></span>
            <template v-else>
              <span :class="$style.modelSelectProvider">{{ selectedProviderName }}</span>
              <span :class="$style.modelSelectName">{{ selectedModel?.name || 'Select Model' }}</span>
              <DownOutlined :class="$style.modelSelectChevron" />
            </template>
          </div>
          <template #content>
            <div :class="$style.modelDropdownContent">
              <div :class="$style.modelDropdownHeader">
                <a-input
                  v-model:value="modelSearchText"
                  placeholder="Search models..."
                  size="small"
                  :class="$style.modelSearchInput"
                >
                  <template #prefix><SearchOutlined style="color: #bfbfbf" /></template>
                </a-input>
              </div>
              <div :class="$style.modelDropdownBody">
                <template v-if="filteredGroupedModels.length > 0">
                  <div v-for="group in filteredGroupedModels" :key="group.provider" :class="$style.modelGroup">
                    <div :class="$style.modelGroupTitle" :style="providerAccentStyle(group.provider)">
                      <span>{{ group.provider_icon ? `${group.provider_icon} ${group.provider_name}` : group.provider_name }}</span>
                    </div>
                    <div
                      v-for="model in group.models"
                      :key="model.id"
                      :class="[$style.modelItem, agentStore.selectedModelId === model.id ? $style.modelItemActive : '']"
                      @click="handleModelSelect(model.id)"
                    >
                      <div :class="$style.modelItemName">{{ model.name }}</div>
                      <span v-if="model.max_tokens" :class="$style.modelItemMeta">{{ formatTokenLimit(model.max_tokens) }}</span>
                      <CheckOutlined v-if="agentStore.selectedModelId === model.id" :class="$style.modelItemCheck" />
                    </div>
                  </div>
                </template>
                <div v-else-if="modelSearchText" :class="$style.modelDropdownEmpty">
                  No models found
                </div>
                <template v-else-if="agentStore.models.length > 0">
                  <div
                    v-for="model in agentStore.models"
                    :key="model.id"
                    :class="[$style.modelItem, agentStore.selectedModelId === model.id ? $style.modelItemActive : '']"
                    @click="handleModelSelect(model.id)"
                  >
                    <div :class="$style.modelItemName">{{ model.name }}</div>
                    <span v-if="model.max_tokens" :class="$style.modelItemMeta">{{ formatTokenLimit(model.max_tokens) }}</span>
                    <CheckOutlined v-if="agentStore.selectedModelId === model.id" :class="$style.modelItemCheck" />
                  </div>
                </template>
                <div v-else :class="$style.modelDropdownEmpty">
                  No models configured
                </div>
              </div>
            </div>
          </template>
        </a-popover>
        <a-button size="small" :class="$style.contextBtn" @click="showContextModal">
          <template #icon><InfoCircleOutlined /></template>
        </a-button>
        <a-button size="small" @click="showModelConfig">
          <template #icon><SettingOutlined /></template>
        </a-button>
        <a-button size="small" @click="agentStore.clearMessages">
          <template #icon><ClearOutlined /></template>
        </a-button>
      </div>
    </div>


    <div :class="$style.messagesWrapper">
      <div :class="$style.messages" ref="messagesContainer" @scroll="handleScroll">
        <div v-if="agentStore.messages.length === 0 && !agentStore.streaming" :class="$style.empty">
          <p>Type a question below to chat with the Agent freely.</p>
          <p>Use field actions elsewhere when you want request-aware analysis.</p>
        </div>

        <template v-for="turn in chatTurns" :key="turn.key">
          <div v-if="turn.user" :class="[$style.message, $style.user]">
            <div :class="$style.role">You <span v-if="turn.user.deliveryStatus" :class="$style.deliveryBadge">{{ turn.user.deliveryStatus === 'queued' ? 'queued' : 'sent' }}</span></div>
            <div :class="$style.content">
              <p>{{ turn.user.content }}</p>
            </div>
          </div>

          <div v-if="turn.items.length > 0 || turn.isActive" :class="[$style.message, $style.assistant, $style.agentTurn]">
            <div :class="$style.role">Agent</div>
            <div :class="$style.content">
              <div v-if="turn.items.length > 0" :class="$style.agentStepFlow">
                <template v-for="item in turn.items" :key="item.key">
                  <div v-if="item.type === 'thought' && item.message" :class="[$style.cycleBox, $style.thoughtBox]">
                    <button type="button" :class="$style.thoughtToggle" @click="toggleThoughtExpanded(item.key)">
                      <span>Thinking</span>
                      <span>{{ isThoughtExpanded(item.key) ? '-' : '+' }}</span>
                    </button>
                    <div v-if="isThoughtExpanded(item.key)" :class="$style.boxContent" v-html="renderMarkdown(item.message.content)"></div>
                  </div>

                  <div v-else-if="item.type === 'assistant' && item.message" :class="$style.answerBlock" v-html="renderMarkdown(item.message.content)"></div>

                  <div v-else-if="item.type === 'tool' && item.cycle" :class="[$style.cycleBox, $style.toolBox]">
                    <div :class="$style.toolCard" @click="toggleToolExpanded(item.cycle.id)">
                      <div :class="$style.toolCardHeader">
                        <span :class="$style.toolCardTitle">Tool - {{ item.cycle.action?.toolName }}</span>
                        <div :class="$style.toolCardStatus">
                          <span v-if="!item.cycle.observation" :class="$style.statusPulse"></span>
                          <span v-else-if="item.cycle.observation.role === 'agent_error'" :class="$style.statusError">Failed</span>
                          <span v-else :class="$style.statusSuccess">Done</span>
                        </div>
                      </div>
                      <div v-if="toolExpanded.get(item.cycle.id)" :class="$style.toolCardBody" @click.stop>
                        <div v-if="item.cycle.action?.arguments" :class="$style.toolArgs">
                          <div :class="$style.toolSectionTitle">Arguments</div>
                          <pre><code>{{ formatArgs(item.cycle.action.arguments) }}</code></pre>
                        </div>
                        <div v-if="item.cycle.observation" :class="[$style.toolResult, item.cycle.observation.role === 'agent_error' ? $style.toolResultError : '']">
                          <div :class="$style.toolSectionTitle">Result</div>
                          <div :class="$style.boxContent" v-html="renderMarkdown(item.cycle.observation.content)"></div>
                          <div v-if="item.cycle.observation.role === 'agent_error'" :class="$style.errorMeta">
                            <span v-if="item.cycle.observation.errorCategory">category={{ item.cycle.observation.errorCategory }}</span>
                            <span v-if="item.cycle.observation.errorToolName">tool={{ item.cycle.observation.errorToolName }}</span>
                            <span v-if="item.cycle.observation.errorTimeout">timeout={{ item.cycle.observation.errorTimeout }}</span>
                            <span v-if="typeof item.cycle.observation.errorRecovered === 'boolean'">recovered={{ item.cycle.observation.errorRecovered }}</span>
                          </div>
                          <div v-if="item.cycle.observation.role === 'agent_error'" :class="$style.errorActions">
                            <button type="button" :class="$style.errorActionBtn" :disabled="agentStore.streaming || !agentStore.lastUserQuery" @click.stop="retryLastQuery">Retry</button>
                            <button type="button" :class="$style.errorActionBtn" @click.stop="copyError(item.cycle)">Copy Error</button>
                          </div>
                          <div v-if="item.cycle.observation.requestIds && item.cycle.observation.requestIds.length > 0" :class="$style.requestIds">
                            <span>Related: </span>
                            <a-tag v-for="id in item.cycle.observation.requestIds.slice(0, 3)" :key="id" size="small" @click.stop="jumpToRequest(id)">{{ shortId(id) }}</a-tag>
                            <span v-if="item.cycle.observation.requestIds.length > 3">+{{ item.cycle.observation.requestIds.length - 3 }} more</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="item.type === 'observation' && item.message" :class="[$style.cycleBox, item.message.role === 'agent_error' ? $style.toolResultError : $style.toolResult]">
                    <div :class="$style.boxRole">Tool Result</div>
                    <div :class="$style.boxContent" v-html="renderMarkdown(item.message.content)"></div>
                  </div>

                  <div v-else-if="item.type === 'decision' && item.message" :class="[$style.cycleBox, $style.decisionBox]">
                    <div :class="$style.boxRole">Decision</div>
                    <div :class="$style.boxContent">{{ item.message.content }}</div>
                  </div>

                  <div v-else-if="item.type === 'retry' && item.message" :class="[$style.cycleBox, $style.retryBox]">
                    <div :class="$style.retryContent">
                      <span>{{ item.message.content }}</span>
                    </div>
                  </div>
                </template>
              </div>

              <div v-if="turn.isActive && agentStore.llmWaiting && !agentStore.currentContent" :class="$style.inlineThinking">
                <span :class="$style.stepCircleLoading"></span>
                <span :class="$style.shimmerText">thinking...</span>
              </div>
              <div v-if="turn.isActive && agentStore.currentContent" :class="$style.answerBlock" v-html="renderMarkdown(agentStore.currentContent)"></div>
            </div>
          </div>
        </template>
        <div v-if="agentStore.hasPendingQuery">
          <div v-for="(item, idx) in agentStore.pendingQueue" :key="'pending_' + idx" :class="[$style.message, $style.user, $style.pendingMessage]">
            <div :class="$style.role">👤 You <span :class="$style.pendingTag">(Pending...)</span></div>
            <div :class="$style.content">
              <p>{{ item.query }}</p>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="!isAutoFollowing && agentStore.streaming"
        :class="$style.jumpToBottomBtn"
        @click="resumeAutoFollow"
      >
        <ArrowDownOutlined />
      </div>
    </div>

    <div v-if="showTokenFooter" :class="$style.tokenFooterBar">
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryLabel">Tokens:</span>
      </div>
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryValue">{{ formatTokenUsage?.input ?? '-' }}</span>
        <span :class="$style.tokenSummaryLabel">in</span>
      </div>
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryLabel">/</span>
      </div>
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryValue">{{ formatTokenUsage?.output ?? '-' }}</span>
        <span :class="$style.tokenSummaryLabel">out</span>
      </div>
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryLabel">=</span>
      </div>
      <div :class="$style.tokenSummaryItem">
        <span :class="$style.tokenSummaryValue">{{ formatTokenUsage?.total ?? 'N/A' }}</span>
        <span :class="$style.tokenSummaryLabel">total</span>
      </div>
      <div v-if="isEstimatedTokenUsage" :class="$style.tokenSummaryItem">
        <span :class="$style.tokenEstimateTag">estimated</span>
      </div>
    </div>

    <div :class="$style.input">
      <div v-if="agentStore.pendingQueue.length > 0" :class="$style.pendingIndicator">
        <a-spin size="small" />
        <span>{{ agentStore.pendingQueue.length }} queued for next run</span>
      </div>
      <div v-if="contextChips.length > 0" :class="$style.contextChips">
        <span
          v-for="chip in contextChips"
          :key="chip.key"
          :class="[$style.contextChip, chip.dismissible ? '' : $style.contextChipDim]"
        >
          <span>{{ chip.label }}</span>
          <button v-if="chip.dismissible" type="button" :class="$style.contextChipClose" @click="dismissRequestContext">×</button>
        </span>
      </div>
      <a-textarea
        v-model:value="input"
            :placeholder="agentStore.streaming ? 'Steer current analysis...' : 'Ask anything...'"
        :autoSize="{ minRows: 2, maxRows: 4 }"
        @keydown.enter="handleEnter"
      />
      <div :class="$style.inputActions">
        <template v-if="agentStore.streaming">
          <a-button type="default" @click="handleQueue" :disabled="!input.trim()">Queue</a-button>
          <a-button type="primary" @click="handleSend" :disabled="!input.trim()">Steer</a-button>
          <a-button type="default" danger @click="handleStop">
            <template #icon><StopOutlined /></template>
          </a-button>
        </template>
        <a-button
          v-else
          type="primary"
          :disabled="!input.trim()"
          @click="handleSend"
        >
          <template #icon><SendOutlined /></template>
        </a-button>
      </div>
    </div>
  </div>
  <ModelConfigModal v-model:visible="modelConfigVisible" @saved="() => agentStore.loadModels(true)" />
  <ContextModal v-model:visible="contextModalVisible" />

</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, reactive } from 'vue';
import { SendOutlined, ClearOutlined, SettingOutlined, StopOutlined, InfoCircleOutlined, DownOutlined, SearchOutlined, CheckOutlined } from '@ant-design/icons-vue';
import MarkdownIt from 'markdown-it';
import { useAgentStore } from '@/stores/agentStore';
import type { Message } from '@/stores/agentStore';
import { useRequestStore } from '@/stores/requestStore';
import type { AgentModel } from '@/types';
import ModelConfigModal from './ModelConfigModal.vue';
import ContextModal from './ContextModal.vue';

interface ReActCycle {
  id: string;
  stepNumber: number;
  thought?: Message;
  action?: Message;
  observation?: Message;
  decision?: Message;
  retry?: Message;
  specialistName?: string;
  isStreaming?: boolean;
}

interface AgentTurnItem {
  key: string;
  type: 'thought' | 'assistant' | 'tool' | 'observation' | 'decision' | 'retry';
  message?: Message;
  cycle?: ReActCycle;
}

interface ChatTurn {
  key: string;
  user?: Message;
  messages: Message[];
  items: AgentTurnItem[];
  isActive: boolean;
}

const agentStore = useAgentStore();
const requestStore = useRequestStore();
const input = ref('');
const messagesContainer = ref<HTMLElement | null>(null);
const modelConfigVisible = ref(false);
const contextModalVisible = ref(false);
const modelDropdownVisible = ref(false);
const modelSearchText = ref('');
const dismissedRequestContextId = ref<string | null>(null);
const isAutoFollowing = ref(true);

const toolExpanded = reactive(new Map<string, boolean>());
const thoughtExpanded = reactive(new Map<string, boolean>());

const toggleToolExpanded = (cycleId: string) => {
  toolExpanded.set(cycleId, !toolExpanded.get(cycleId));
};

const toggleThoughtExpanded = (key: string) => {
  thoughtExpanded.set(key, !thoughtExpanded.get(key));
};

const isThoughtExpanded = (key: string) => thoughtExpanded.get(key) === true;

const handleScroll = (e: Event) => {
  const target = e.target as HTMLElement;
  const threshold = 50;
  const isAtBottom = target.scrollHeight - target.scrollTop - target.clientHeight <= threshold;
  isAutoFollowing.value = isAtBottom;
};

const md = new MarkdownIt();

const selectedModel = computed<AgentModel | undefined>(() => agentStore.getSelectedModel());
const selectedModelGroup = computed(() => agentStore.modelsGrouped.find((group) => group.models?.some((model) => model.id === agentStore.selectedModelId)) || null);
const selectedProviderName = computed(() => selectedModelGroup.value?.provider_name || 'Provider');
const hasGroupedModels = computed(() => agentStore.modelsGrouped.some((group) => (group.models?.length || 0) > 0));
const normalizedModelSearch = computed(() => modelSearchText.value.trim().toLowerCase());
const filteredGroupedModels = computed(() => {
  const keyword = normalizedModelSearch.value;
  if (!hasGroupedModels.value) {
    return [];
  }

  const sourceGroups = agentStore.modelsGrouped.filter((group) => (group.models?.length || 0) > 0);
  if (!keyword) {
    return sourceGroups;
  }

  return sourceGroups
    .map((group) => ({
      ...group,
      models: (group.models || []).filter((model) => {
        const searchable = [
          model.name,
          model.id,
          group.provider_name,
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase();
        return searchable.includes(keyword);
      }),
    }))
    .filter((group) => group.models.length > 0);
});
const filteredFallbackModels = computed(() => {
  if (hasGroupedModels.value) {
    return [] as AgentModel[];
  }

  const keyword = normalizedModelSearch.value;
  if (!keyword) {
    return agentStore.models;
  }

  return agentStore.models.filter((model) => {
    const searchable = [model.name, model.id]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();
    return searchable.includes(keyword);
  });
});
const effectiveRequestId = computed(() => {
  const selectedId = requestStore.selectedId;
  if (!selectedId || dismissedRequestContextId.value === selectedId) return undefined;
  return selectedId;
});
const contextChips = computed(() => {
  const chips: Array<{ key: string; label: string; dismissible: boolean }> = [];
  const selectedRequest = requestStore.selectedRequest;
  if (selectedRequest && effectiveRequestId.value) {
    const path = selectedRequest.path || selectedRequest.url || selectedRequest.id;
    chips.push({ key: 'request', label: `${selectedRequest.method || 'REQ'} ${path}`, dismissible: true });
  }
  if (agentStore.ensureChatSessionId()) {
    chips.push({ key: 'session', label: `Session ${agentStore.ensureChatSessionId()}`, dismissible: false });
  }
  return chips;
});

const isTraceRole = (role: string) => ['agent_thought', 'agent_action', 'agent_observation', 'agent_error', 'agent_decision', 'agent_retry'].includes(role);

const buildTurnItems = (messages: Message[]): AgentTurnItem[] => {
  const items: AgentTurnItem[] = [];
  const pendingTools: AgentTurnItem[] = [];
  const toolsByCallId = new Map<string, AgentTurnItem>();
  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i];
    if (msg.role === 'agent_thought') {
      items.push({ key: msg.id, type: 'thought', message: msg });
      continue;
    }
    if (msg.role === 'assistant') {
      items.push({ key: msg.id, type: 'assistant', message: msg });
      continue;
    }
    if (msg.role === 'agent_action') {
      const item: AgentTurnItem = {
        key: msg.id,
        type: 'tool',
        cycle: {
          id: msg.id,
          stepNumber: items.length + 1,
          action: msg,
          specialistName: msg.specialistName,
        },
      };
      items.push(item);
      pendingTools.push(item);
      if (msg.toolCallId) {
        toolsByCallId.set(msg.toolCallId, item);
      }
      continue;
    }
    if (msg.role === 'agent_observation' || msg.role === 'agent_error') {
      const exactTool = msg.toolCallId ? toolsByCallId.get(msg.toolCallId) : undefined;
      const fallbackTool = [...items]
        .reverse()
        .find((item) => item.type === 'tool' && item.cycle && !item.cycle.observation)
        || [...items].reverse().find((item) => item.type === 'tool' && item.cycle);
      const pendingTool = exactTool || pendingTools.shift() || fallbackTool;
      if (exactTool) {
        const pendingIndex = pendingTools.indexOf(exactTool);
        if (pendingIndex >= 0) pendingTools.splice(pendingIndex, 1);
      }
      if (pendingTool?.cycle) {
        pendingTool.cycle.observation = msg;
        continue;
      }
      items.push({ key: msg.id, type: 'observation', message: msg });
      continue;
    }
    if (msg.role === 'agent_decision') {
      items.push({ key: msg.id, type: 'decision', message: msg });
      continue;
    }
    if (msg.role === 'agent_retry') {
      items.push({ key: msg.id, type: 'retry', message: msg });
    }
  }
  return items;
};

const chatTurns = computed<ChatTurn[]>(() => {
  const turns: ChatTurn[] = [];
  let current: ChatTurn | null = null;

  const ensureTurn = () => {
    if (!current) {
      current = {
        key: `turn_${turns.length + 1}`,
        messages: [],
        items: [],
        isActive: false,
      };
      turns.push(current);
    }
    return current;
  };

  for (const msg of agentStore.messages) {
    if (msg.role === 'user') {
      current = {
        key: msg.id,
        user: msg,
        messages: [],
        items: [],
        isActive: false,
      };
      turns.push(current);
      continue;
    }

    const turn = ensureTurn();
    turn.messages.push(msg);
  }

  if (agentStore.streaming) {
    const turn = ensureTurn();
    turn.isActive = true;
  }

  return turns.map((turn) => ({
    ...turn,
    items: buildTurnItems(turn.messages),
  }));
});

const formatArgs = (args: string) => {
  try {
    const obj = JSON.parse(args);
    return JSON.stringify(obj, null, 2);
  } catch {
    return args;
  }
};

const providerAccentStyle = (provider: string) => {
  const palette: Record<string, string> = {
    openai: '#2f9a45',
    zhipu: '#2f6faf',
    ark: '#2f6faf',
    mock: '#8c8c8c',
  };
  return { borderLeftColor: palette[provider] || '#8c8c8c' };
};

const formatTokenLimit = (tokens: number) => {
  if (tokens >= 1000) return `${Math.round(tokens / 1000)}K`;
  return String(tokens);
};

const latestAssistantMessage = computed(() => {
  const assistantMessages = agentStore.messages.filter((msg) => msg.role === 'assistant');
  return assistantMessages[assistantMessages.length - 1] || null;
});

const estimateTokensFromText = (text: string) => {
  const normalized = text.replace(/\s+/g, ' ').trim();
  if (!normalized) return 0;
  return Math.ceil(normalized.length / 4);
};

const formatTokenUsage = computed(() => {
  const usage = agentStore.tokenUsage;
  const tokensUsed = agentStore.tokensUsed;
  if (!usage) {
    if (tokensUsed > 0) {
      return { input: '-', output: '-', total: tokensUsed };
    }
    const estimated = estimateTokensFromText(latestAssistantMessage.value?.content || '');
    if (estimated > 0) {
      return { input: '-', output: '-', total: estimated };
    }
    return null;
  }
  const total = usage.input_tokens + usage.output_tokens + (usage.cache_creation_input_tokens || 0) + (usage.cache_read_input_tokens || 0);
  return {
    input: usage.input_tokens,
    output: usage.output_tokens,
    total,
  };
});

const showTokenFooter = computed(() => {
  return !agentStore.streaming && (agentStore.messages.some((msg) => msg.role === 'assistant') || agentStore.tokensUsed > 0 || !!agentStore.tokenUsage);
});

const isEstimatedTokenUsage = computed(() => !agentStore.tokenUsage && !!formatTokenUsage.value);

const renderMarkdown = (content: string) => {
  return md.render(content);
};

const shortId = (id: string) => {
  if (!id) return '';
  return id.length > 12 ? id.substring(0, 12) + '...' : id;
};

const scrollToBottom = async (force = false) => {
  await nextTick();
  if (!force && !isAutoFollowing.value) return;
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
  }
};

const resumeAutoFollow = () => {
  isAutoFollowing.value = true;
  scrollToBottom(true);
};

const retryLastQuery = () => {
  if (!agentStore.lastUserQuery || agentStore.streaming) return;
  agentStore.analyzeWithQuery(effectiveRequestId.value, agentStore.lastUserQuery);
};

const copyError = async (cycle: ReActCycle) => {
  const payload = [
    cycle.observation?.content || '',
    cycle.observation?.errorCategory ? `category=${cycle.observation.errorCategory}` : '',
    cycle.observation?.errorToolName ? `tool=${cycle.observation.errorToolName}` : '',
    cycle.observation?.errorTimeout ? `timeout=${cycle.observation.errorTimeout}` : '',
  ].filter(Boolean).join('\n');
  await navigator.clipboard.writeText(payload);
};

const handleModelSelect = (modelId: string) => {
  agentStore.switchModel(modelId);
  modelDropdownVisible.value = false;
  modelSearchText.value = '';
};

const showModelConfigFromDropdown = () => {
  modelDropdownVisible.value = false;
  modelSearchText.value = '';
  showModelConfig();
};

watch(() => agentStore.messages, () => scrollToBottom(false), { deep: true });
watch(() => agentStore.currentContent, () => scrollToBottom(false));
watch(modelDropdownVisible, (open) => {
  if (!open) {
    modelSearchText.value = '';
  }
});
onMounted(() => {
  agentStore.loadModels();
});

watch(() => requestStore.selectedId, (nextId) => {
  if (nextId !== dismissedRequestContextId.value) {
    dismissedRequestContextId.value = null;
  }
});

const handleEnter = (e: KeyboardEvent) => {
  if (!e.shiftKey) {
    e.preventDefault();
    handleSend();
  }
};

const handleSend = () => {
  if (!input.value.trim()) return;
  agentStore.analyzeWithQuery(effectiveRequestId.value, input.value, undefined, agentStore.streaming ? 'steer' : 'queue');
  input.value = '';
};

const handleQueue = () => {
  if (!input.value.trim()) return;
  agentStore.analyzeWithQuery(effectiveRequestId.value, input.value, undefined, 'queue');
  input.value = '';
};

const handleStop = () => {
  agentStore.stopAnalysis();
};

const dismissRequestContext = () => {
  dismissedRequestContextId.value = requestStore.selectedId;
};

const truncateValue = (value: string) => {
  if (!value) return '';
  return value.length > 64 ? `${value.slice(0, 64)}...` : value;
};

const formatConfidence = (value: number) => {
  if (!Number.isFinite(value)) return '0.00';
  return value.toFixed(2);
};

const jumpToRequest = async (requestId: string) => {
  if (!requestId) return;
  try {
    await requestStore.selectRequestById(requestId);
  } catch (error) {
    console.error('[AgentPanel] Failed to jump to request:', error);
  }
};

const showModelConfig = () => {
  modelConfigVisible.value = true;
};

const showContextModal = () => {
  contextModalVisible.value = true;
};
</script>

<style module>
.container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  min-height: 0;
  border-left: 1px solid #9f9f9f;
  background-color: #ededed;
  color: #1f1f1f;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 7px 10px;
  border-bottom: 1px solid #a9a9a9;
  background: linear-gradient(180deg, #ececec 0%, #dfdfdf 100%);
}

.headerActions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.modelPopoverOverlay {
  padding: 0;
}

.modelSelectTrigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 140px;
  max-width: 220px;
  height: 26px;
  padding: 0 8px;
  border: 1px solid #b2b2b2;
  border-radius: 3px;
  background: linear-gradient(180deg, #ffffff 0%, #ececec 100%);
  box-sizing: border-box;
  color: #2f2f2f;
  font-size: 12px;
  cursor: pointer;
}

.modelSelectTrigger:hover {
  border-color: #8d8d8d;
  background: linear-gradient(180deg, #ffffff 0%, #e6e6e6 100%);
}

.modelSelectTriggerLoading {
  justify-content: center;
}

.modelSelectName {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modelSelectProvider {
  flex-shrink: 0;
  max-width: 70px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 1px 5px;
  border: 1px solid #b2b2b2;
  border-radius: 2px;
  background: #dedede;
  font-size: 10px;
  font-weight: 700;
}

.modelSelectChevron {
  color: #6c6c6c;
  font-size: 10px;
}

.modelDropdownContent {
  width: 308px;
  background: #fff;
}

.modelDropdownHeader {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-bottom: 1px solid #ededed;
  background: #fafafa;
}

.modelSearchInput {
  flex: 1;
}

.modelConfigBtn {
  flex-shrink: 0;
}

.modelDropdownBody {
  max-height: 320px;
  overflow-y: auto;
  padding: 6px 0;
}

.modelGroup + .modelGroup {
  margin-top: 4px;
}

.modelGroupTitle {
  margin: 4px 8px;
  padding: 5px 8px;
  border-left: 3px solid #8c8c8c;
  background: #f0f0f0;
  font-size: 10px;
  line-height: 1.4;
  color: #555;
  font-weight: 700;
}

.modelItem {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-left: 2px solid transparent;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.modelItem:hover {
  background: #f5f5f5;
}

.modelItemActive {
  background: #eef6ff;
  border-left-color: #1677ff;
}

.modelItemName {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #1f1f1f;
}

.modelItemMeta {
  flex-shrink: 0;
  color: #777;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 10px;
}

.modelItemCheck {
  color: #1677ff;
  font-size: 12px;
}

.modelDropdownEmpty {
  padding: 22px 12px;
  text-align: center;
  color: #8c8c8c;
  font-size: 12px;
}

.switchLabel {
  font-size: 10px;
}

.modelInfo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background-color: #e7e7e7;
  border-bottom: 1px solid #b3b3b3;
  font-size: 11px;
}

.provider {
  background-color: #d5d5d5;
  color: #222;
  padding: 1px 5px;
  border-radius: 2px;
  border: 1px solid #b2b2b2;
  font-weight: 600;
}

.description {
  color: #5d5d5d;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agentBadge {
  background-color: #1890ff;
  color: white;
  padding: 1px 5px;
  border-radius: 2px;
  font-size: 10px;
  font-weight: 600;
}

.title {
  font-weight: 600;
  font-size: 12px;
  letter-spacing: 0.02em;
}

.messagesWrapper {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
}

.messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #efefef;
}

.turnDivider {
  display: flex;
  align-items: center;
  margin: 10px 0 2px 0;
}

.turnDividerTime {
  font-size: 10px;
  color: #8c8c8c;
  margin-right: 8px;
}

.turnDividerLine {
  flex: 1;
  height: 1px;
  background-color: #d9d9d9;
}

.jumpToBottomBtn {
  position: absolute;
  bottom: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #ffffff;
  color: #1890ff;
  border: 1px solid #1890ff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  z-index: 10;
  transition: all 0.2s;
}

.jumpToBottomBtn:hover {
  background-color: #e6f7ff;
  color: #096dd9;
  border-color: #096dd9;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.message {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.role {
  font-size: 11px;
  color: #666;
}

.content {
  padding: 8px 10px;
  border-radius: 2px;
  background-color: #e2e2e2;
  border: 1px solid #c2c2c2;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.45;
}

.content p {
  margin: 0 0 6px 0;
}

.content p:last-child {
  margin-bottom: 0;
}

.content pre {
  background-color: #dbdbdb;
  padding: 6px;
  border-radius: 2px;
  overflow-x: auto;
  margin: 4px 0;
}

.content code {
  background-color: #dbdbdb;
  padding: 1px 3px;
  border-radius: 2px;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 11px;
}

.user {
  align-items: flex-end;
}

.user .role {
  text-align: right;
}

.user .content {
  max-width: 86%;
  background-color: #dde8f8;
  border-color: #b7cbe7;
}

.deliveryBadge {
  margin-left: 6px;
  padding: 1px 5px;
  border: 1px solid #c8c8c8;
  border-radius: 999px;
  color: #777;
  background: #efefef;
  font-size: 10px;
}

.assistant {
  align-items: flex-start;
}

.agentTurn .content {
  width: 100%;
  background-color: #ededed;
}

.answerBlock {
  color: #222;
}

.answerBlock + .answerBlock,
.agentStepFlow + .answerBlock,
.inlineThinking + .answerBlock,
.answerBlock + .relatedTimeline,
.answerBlock + .provenanceSection {
  margin-top: 8px;
}

.agentStepFlow {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 8px;
}

.agentStep {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.inlineThinking {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
  color: #666;
}

/* Timeline - inline ReAct cycle steps */

.timelineItem {
  display: flex;
  position: relative;
  padding-bottom: 8px;
}

.timelineItem:last-child {
  padding-bottom: 0;
}

.timelineLeft {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-right: 10px;
  width: 16px;
  flex-shrink: 0;
}

.stepCircle {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background-color: #b7b7b7;
  color: #fff;
  font-size: 9px;
  font-weight: bold;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
}

.stepCircleLoading {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #b7b7b7;
  margin-top: 3px;
  animation: pulse 1.5s infinite ease-in-out;
}

@keyframes pulse {
  0% { transform: scale(0.8); opacity: 0.5; }
  50% { transform: scale(1.2); opacity: 1; }
  100% { transform: scale(0.8); opacity: 0.5; }
}

.stepLine {
  width: 1px;
  background-color: #d9d9d9;
  flex: 1;
  margin-top: 2px;
}

.timelineRight {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.cycleHeader {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 2px;
}

.cycleSummary {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid #c8c8c8;
  border-radius: 2px;
  background: linear-gradient(180deg, #f8f8f8 0%, #e8e8e8 100%);
  cursor: pointer;
  font-size: 11px;
}

.cycleSummaryExpanded {
  border-color: #9f9f9f;
  background: #f2f2f2;
}

.cycleSummaryError {
  border-color: #d6a2a2;
  background: #fff7f7;
  color: #9f2727;
}

.cycleSummaryStep {
  font-weight: 700;
  color: #444;
}

.cycleSummaryText {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stepDuration {
  font-size: 10px;
  color: #8c8c8c;
}

.cycleBox {
  display: flex;
  flex-direction: column;
  border: 1px solid #d9d9d9;
  border-radius: 2px;
  background-color: #fff;
}

.boxRole {
  font-size: 10px;
  padding: 2px 6px;
  background-color: #f5f5f5;
  border-bottom: 1px solid #d9d9d9;
  color: #666;
}

.boxContent {
  padding: 6px;
  font-size: 11px;
  color: #333;
  word-break: break-word;
}

.boxContent p {
  margin: 0 0 6px 0;
}

.boxContent p:last-child {
  margin-bottom: 0;
}

.boxContent pre {
  background-color: #f5f5f5;
  padding: 6px;
  border-radius: 2px;
  overflow-x: auto;
  margin: 4px 0;
}

.boxContent code {
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  background-color: #f5f5f5;
  padding: 1px 3px;
  border-radius: 2px;
}

.thoughtBox {
  border-color: #d9d9d9;
}
.thoughtBox .boxRole {
  background-color: #fafafa;
  border-color: #e8e8e8;
  color: #8a6d3b;
}

.decisionBox {
  border-color: #b7eb8f;
}
.decisionBox .boxRole {
  background-color: #f6ffed;
  border-color: #b7eb8f;
  color: #52c41a;
}

.retryBox {
  border-color: #ffe58f;
  background-color: #fffbe6;
  padding: 6px 10px;
}
.retryContent {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #ad6800;
  animation: retryPulse 1.5s ease-in-out infinite;
}
.retryIcon {
  font-size: 13px;
  animation: retrySpin 1s linear infinite;
}
@keyframes retrySpin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
@keyframes retryPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.toolBox {
  border-color: #91d5ff;
}

.toolCard {
  display: flex;
  flex-direction: column;
  cursor: pointer;
}

.toolCardHeader {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px;
  background-color: #e6f7ff;
  border-bottom: 1px solid transparent;
}

.toolCardTitle {
  font-size: 11px;
  font-weight: 600;
  color: #096dd9;
}

.toolCardStatus {
  display: flex;
  align-items: center;
}

.statusSuccess {
  color: #52c41a;
  font-weight: bold;
  font-size: 12px;
}

.statusError {
  color: #ff4d4f;
  font-weight: bold;
  font-size: 12px;
}

.statusPulse {
  width: 8px;
  height: 8px;
  background-color: #1890ff;
  border-radius: 50%;
  animation: pulse 1.5s infinite ease-in-out;
}

.toolCardBody {
  border-top: 1px solid #91d5ff;
  background-color: #fafafa;
  cursor: default;
}

.toolSectionTitle {
  font-size: 10px;
  color: #888;
  text-transform: uppercase;
  padding: 4px 6px;
  background-color: #f0f0f0;
  border-bottom: 1px solid #e8e8e8;
}

.toolArgs pre {
  margin: 0;
  padding: 6px;
  background-color: #fff;
  border-bottom: 1px solid #e8e8e8;
  font-size: 10px;
  color: #333;
  overflow-x: auto;
}

.toolResult {
  display: flex;
  flex-direction: column;
}

.toolResultError .toolSectionTitle {
  background-color: #fff1f0;
  color: #cf1322;
  border-color: #ffa39e;
}

.toolResultError .boxContent {
  background-color: #fff;
}

@keyframes shimmer-sweep {
  0% { background-position: -200% center; }
  100% { background-position: 200% center; }
}

.shimmerText {
  font-size: 11px;
  color: transparent;
  background: linear-gradient(90deg, #8c8c8c 25%, #d9d9d9 50%, #8c8c8c 75%);
  background-size: 200% 100%;
  -webkit-background-clip: text;
  background-clip: text;
  animation: shimmer-sweep 2s infinite linear;
  margin-top: 2px;
}

.specialistBadge {
  display: inline-block;
  padding: 1px 5px;
  border-radius: 2px;
  font-size: 9px;
  font-weight: 600;
  background-color: #1890ff;
  color: #fff;
}

.requestIds {
  padding: 6px;
  font-size: 11px;
  color: #666;
  background-color: #fff;
  border-top: 1px solid #e8e8e8;
}

.errorMeta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px;
  font-size: 11px;
  color: #cf1322;
  background-color: #fff;
  border-top: 1px solid #ffa39e;
}

.errorActions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  padding: 6px;
  background: #fff7f7;
  border-top: 1px solid #ffa39e;
}

.errorActionBtn {
  height: 22px;
  padding: 0 8px;
  border: 1px solid #bd9b9b;
  border-radius: 2px;
  background: #fff;
  color: #9f2727;
  font-size: 11px;
  cursor: pointer;
}

.errorActionBtn:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.empty {
  color: #666;
  text-align: center;
  margin-top: 32px;
  font-size: 12px;
}

.empty p {
  margin: 8px 0;
}

.agentHint {
  background-color: #e6f7ff;
  padding: 8px;
  border-radius: 4px;
  border: 1px solid #91d5ff;
  color: #1890ff;
}

.relatedTimeline {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.relatedHeader {
  font-size: 11px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 6px;
}

.relatedIds {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.relatedRequestBtn {
  display: inline-flex !important;
  align-items: center;
  gap: 3px;
  padding: 2px 8px !important;
  height: auto !important;
  font-size: 11px !important;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace !important;
  background: linear-gradient(180deg, #ffffff 0%, #f5f5f5 100%) !important;
  border: 1px solid #91d5ff !important;
  border-radius: 3px !important;
  color: #1890ff !important;
  cursor: pointer;
  transition: all 0.15s ease;
  text-decoration: none !important;
}

.relatedRequestBtn:hover {
  background: linear-gradient(180deg, #e6f7ff 0%, #bae7ff 100%) !important;
  border-color: #40a9ff !important;
  color: #096dd9 !important;
  box-shadow: 0 1px 3px rgba(24, 144, 255, 0.25);
}

.relatedRequestBtn:active {
  background: linear-gradient(180deg, #bae7ff 0%, #91d5ff 100%) !important;
  border-color: #1890ff !important;
}

.moreCount {
  font-size: 11px;
  color: #666;
}

.agent_related .content {
  background-color: #f0f5ff;
  border-color: #adc6ff;
}

.agent_provenance .content {
  background-color: #f7f7f7;
  border-color: #c7c7c7;
}

.provenanceSection {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.provenanceHeader {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  font-weight: 600;
  color: #444;
  margin-bottom: 6px;
}

.provenanceConfidence {
  color: #1f6feb;
}

.provenanceTarget {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  margin-bottom: 8px;
  background: #fff;
  border: 1px solid #d9d9d9;
}

.provenanceLabel {
  font-size: 10px;
  color: #888;
  text-transform: uppercase;
}

.provenanceField {
  font-size: 11px;
  color: #333;
  font-weight: 600;
}

.provenanceValue {
  font-size: 11px;
  color: #666;
  font-family: monospace;
  word-break: break-all;
}

.provenanceLinks {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.provenanceLink {
  padding: 6px 8px;
  background: #fff;
  border: 1px solid #d9d9d9;
}

.provenanceLinkTop {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}

.provenanceRequestBtn {
  display: inline-flex !important;
  align-items: center;
  padding: 2px 6px !important;
  height: auto !important;
  font-size: 11px !important;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace !important;
  background: linear-gradient(180deg, #ffffff 0%, #f5f5f5 100%) !important;
  border: 1px solid #d9d9d9 !important;
  border-radius: 3px !important;
  color: #1f6feb !important;
  cursor: pointer;
  transition: all 0.15s ease;
  text-decoration: none !important;
}

.provenanceRequestBtn:hover {
  background: linear-gradient(180deg, #e6f7ff 0%, #bae7ff 100%) !important;
  border-color: #40a9ff !important;
  color: #096dd9 !important;
}

.provenanceArrow {
  color: #999;
}

.provenanceScore {
  margin-left: auto;
  color: #1f6feb;
  font-size: 11px;
  font-weight: 600;
}

.provenanceFields {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 11px;
  color: #333;
}

.provenanceTransform {
  color: #cf6a00;
  font-weight: 600;
}

.provenanceMeta {
  margin-top: 4px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 10px;
  color: #777;
}

.provenanceEvidence {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #ddd;
}

.provenanceEvidenceTitle {
  font-size: 10px;
  text-transform: uppercase;
  color: #888;
  margin-bottom: 4px;
}

.provenanceEvidenceItem {
  font-size: 11px;
  color: #555;
  margin-bottom: 3px;
  word-break: break-word;
}

.input {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  gap: 6px;
  padding: 8px 10px;
  border-top: 1px solid #ababab;
  background: linear-gradient(180deg, #e6e6e6 0%, #dddddd 100%);
}

.tokenFooterBar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: #e0e0e0;
  border-top: 1px solid #c6c6c6;
  border-bottom: 1px solid #c6c6c6;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', monospace;
  font-size: 10px;
  color: #4f4f4f;
}

.tokenSummaryItem {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.tokenSummaryLabel {
  color: #6f6f6f;
}

.tokenSummaryValue {
  color: #1f1f1f;
  font-weight: 600;
}

.tokenEstimateTag {
  display: inline-block;
  padding: 1px 4px;
  border-radius: 2px;
  background: #f5e6b4;
  color: #7a5f17;
  font-size: 9px;
  text-transform: uppercase;
}

.contextBtn {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid #b2b2b2;
  background: linear-gradient(180deg, #f0f0f0 0%, #e8e8e8 100%);
  cursor: pointer;
  color: #666;
  font-size: 11px;
  transition: all 0.15s;
}

.contextBtn:hover {
  border-color: #888;
  color: #333;
}

.pendingIndicator {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background-color: #fff7e6;
  border: 1px solid #ffd591;
  border-radius: 4px;
  font-size: 11px;
  color: #d48806;
}

.inputActions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
}

.contextChips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.contextChip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 2px 7px;
  border: 1px solid #b2b2b2;
  border-radius: 2px;
  background: linear-gradient(180deg, #f8f8f8 0%, #dfdfdf 100%);
  color: #333;
  font-size: 11px;
}

.contextChipDim {
  color: #666;
  background: #e4e4e4;
}

.contextChipClose {
  border: 0;
  background: transparent;
  color: #666;
  cursor: pointer;
  font-size: 12px;
  line-height: 1;
  padding: 0;
}

:global(.ant-select-selector),
:global(.ant-input),
:global(.ant-input-textarea textarea),
:global(.ant-btn) {
  border-radius: 2px !important;
}

:global(.ant-btn) {
  height: 24px;
  font-size: 12px;
  padding: 0 8px;
}

:global(.ant-input-textarea textarea) {
  font-size: 12px;
  background: #f1f1f1;
}

:global(.ant-switch) {
  font-size: 10px;
}

:global(.ant-tag) {
  font-size: 10px;
  margin: 0;
  padding: 0 4px;
  line-height: 16px;
}
</style>
