<template>
  <a-modal :open="visible" title="Model Configuration" :width="760" @cancel="handleCancel" :footer="null">
    <div :class="$style.layout">
      <!-- Left: Provider List -->
      <div :class="$style.providerList">
        <div :class="$style.providerTools">
          <button type="button" :class="$style.toolButton" @click="openAddProvider">New Channel</button>
        </div>
        <div
          v-for="p in providerList"
          :key="p.id"
          :class="[$style.providerItem, { [$style.active]: p.id === selectedProviderId }]"
          @click="selectProvider(p.id)"
        >
          <span>{{ p.icon || '🔌' }}</span>
          <span :class="$style.providerName">{{ p.name }}</span>
          <span v-if="p.has_api_key" :class="$style.statusDot" title="API key configured" />
          <span :class="$style.modelBadge">{{ p.model_count }}</span>
          <div :class="$style.providerActions" @click.stop>
            <button type="button" :class="$style.inlineAction" @click="openEditProvider(p)">Edit</button>
            <a-popconfirm
              v-if="p.deletable"
              title="Delete this LLM channel?"
              ok-text="Delete"
              cancel-text="Cancel"
              @confirm="deleteProvider(p.id)"
            >
              <button type="button" :class="[$style.inlineAction, $style.dangerAction]" :disabled="deletingProviderId === p.id">Delete</button>
            </a-popconfirm>
          </div>
        </div>
      </div>

      <!-- Right: Provider Config -->
      <div :class="$style.providerConfig">
        <h4>{{ providerDraftVisible ? (editingProviderId ? 'Edit LLM Channel' : 'New LLM Channel') : `Configure ${selectedProviderName}` }}</h4>

        <div v-if="providerList.length === 0 && !providerDraftVisible" :class="$style.emptyState">
          <p :class="$style.emptyTitle">No channels configured</p>
          <p :class="$style.emptyHint">Click "New Channel" on the left to add your first LLM provider.</p>
        </div>

        <a-form v-else-if="providerDraftVisible" layout="vertical">
          <a-row :gutter="12">
            <a-col :span="12">
                <a-form-item label="Channel ID" required :class="$style.formItem">
                  <a-input v-model:value="providerDraft.id" :disabled="!!editingProviderId" placeholder="unique-channel-id" />
                </a-form-item>
            </a-col>
            <a-col :span="12">
                <a-form-item label="Display Name" required :class="$style.formItem">
                  <a-input v-model:value="providerDraft.name" placeholder="My LLM Channel" />
                </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="API Type" :class="$style.formItem">
            <a-select v-model:value="providerDraft.apiType" style="width: 100%">
              <a-select-option value="openai-compatible">OpenAI-Compatible</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="API Key" :class="$style.formItem">
            <a-input-password :key="modalKey" v-model:value="providerDraft.apiKey" placeholder="Channel API key" />
          </a-form-item>
          <a-form-item label="Base URL" required :class="$style.formItem">
            <a-input v-model:value="providerDraft.baseUrl" placeholder="OpenAI-compatible endpoint, e.g. https://api.example.com/v1" />
          </a-form-item>
          <div :class="$style.formActions">
            <button type="button" :class="$style.secondaryButton" @click="cancelProviderForm">Cancel</button>
            <button type="button" :class="$style.primaryButton" :disabled="providerSaving" @click="saveProvider">Save Channel</button>
          </div>
        </a-form>
        
        <a-form v-else layout="vertical">
          <a-form-item label="API Key" :class="$style.formItem">
            <a-input-password :key="`${modalKey}-${selectedProviderId}`" v-model:value="form.apiKey" :placeholder="`Enter ${selectedProviderName} API key`" />
          </a-form-item>
          
          <a-form-item label="Base URL" :class="$style.formItem">
            <a-input v-model:value="form.baseUrl" placeholder="Custom API endpoint (leave empty for default)" />
          </a-form-item>
          
          <div :class="$style.sectionHeader">
            <span>Models</span>
            <button type="button" :class="$style.compactButton" @click="openAddModel">Add Model</button>
          </div>
          
          <div v-if="providerModels.length > 0" :class="$style.modelList">
            <div
              v-for="model in providerModels"
              :key="model.id"
              :class="[$style.modelItem, { [$style.activeModel]: model.id === form.activeModelId }]"
              @click="form.activeModelId = model.id"
            >
              <a-radio :checked="model.id === form.activeModelId" />
              <div :class="$style.modelInfo">
                <div :class="$style.modelName">{{ model.name || model.id }}</div>
                <div :class="$style.modelMeta">
                  <span>{{ model.id }}</span>
                  <span>Output {{ model.max_tokens || 0 }}</span>
                </div>
              </div>
              <div :class="$style.modelActions" @click.stop>
                <button type="button" :class="$style.inlineAction" @click="openEditModel(model)">Edit</button>
                <a-popconfirm
                  title="Delete this model?"
                  ok-text="Delete"
                  cancel-text="Cancel"
                  @confirm="deleteModel(model.id)"
                >
                  <button type="button" :class="[$style.inlineAction, $style.dangerAction]" :disabled="deletingModelId === model.id">Delete</button>
                </a-popconfirm>
              </div>
            </div>
          </div>
          <div v-else :class="$style.noModels">
            No models configured. Add a model manually below.
          </div>
          <div v-if="modelFormVisible" :class="$style.modelForm">
            <div :class="$style.modelFormTitle">{{ editingModelId ? 'Edit Model' : 'Add Model' }}</div>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="Model ID" required :class="$style.formItem">
                  <a-input v-model:value="modelDraft.id" :disabled="!!editingModelId" placeholder="model-id-sent-to-provider" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="Display Name" required :class="$style.formItem">
                  <a-input v-model:value="modelDraft.name" placeholder="Readable model name" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="12">
              <a-col :span="12">
                <a-form-item label="Context Window" :class="$style.formItem">
                  <a-input-number v-model:value="modelDraft.context" :min="1024" :max="2000000" :step="1024" style="width: 100%" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="Max Output Tokens" :class="$style.formItem">
                  <a-input-number v-model:value="modelDraft.output" :min="1" :max="200000" :step="256" style="width: 100%" />
                </a-form-item>
              </a-col>
            </a-row>
            <div :class="$style.formActions">
              <button type="button" :class="$style.secondaryButton" @click="cancelModelForm">Cancel</button>
              <button type="button" :class="$style.primaryButton" :disabled="modelSaving" @click="saveModel">Save Model</button>
            </div>
          </div>
          
          <a-divider />
          
          <a-row :gutter="12">
            <a-col :span="12">
              <a-form-item label="Active Max Tokens" :class="$style.formItem">
                <a-input-number v-model:value="form.maxTokens" :min="256" :max="200000" :step="256" style="width: 100%" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="Temperature" :class="$style.formItem">
                <a-slider v-model:value="form.temperature" :min="0" :max="2" :step="0.1" />
              </a-form-item>
            </a-col>
          </a-row>

          <div :class="$style.footerActions">
            <button type="button" :class="$style.secondaryButton" @click="handleCancel">Cancel</button>
            <button type="button" :class="$style.primaryButton" :disabled="saving" @click="handleOk">Save Configuration</button>
          </div>
        </a-form>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { message } from 'ant-design-vue';
import { configApi } from '@/api/wails';
import { useAgentStore } from '@/stores/agentStore';
import type { AgentModel, ProviderInfo } from '@/types';

interface Props {
  visible: boolean;
}

const props = defineProps<Props>();
const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'saved'): void;
}>();

const agentStore = useAgentStore();
const providerList = computed(() => agentStore.providers);
const providerModels = computed(() => {
  const group = agentStore.modelsGrouped.find((g) => g.provider === selectedProviderId.value);
  return group?.models || [];
});

const modalKey = ref(0); // increments on open to force-recreate components with internal state (e.g. a-input-password visibility)
const selectedProviderId = ref('');
const selectedProviderName = computed(() => {
  const provider = providerList.value.find((item) => item.id === selectedProviderId.value);
  return provider ? provider.name : '';
});

const form = ref({
  apiKey: '',
  baseUrl: '',
  activeModelId: '',
  maxTokens: 4096,
  temperature: 0.7,
});

const modelDraft = ref({
  id: '',
  name: '',
  context: 128000,
  output: 4096,
});

const providerDraft = ref({
  id: '',
  name: '',
  apiType: 'openai-compatible',
  apiKey: '',
  baseUrl: '',
});

const saving = ref(false);
const modelSaving = ref(false);
const providerSaving = ref(false);
const modelFormVisible = ref(false);
const providerDraftVisible = ref(false);
const editingModelId = ref('');
const editingProviderId = ref('');
const deletingModelId = ref('');
const deletingProviderId = ref('');

const resetModelDraft = () => {
  modelDraft.value = {
    id: '',
    name: '',
    context: 128000,
    output: 4096,
  };
};

const resetProviderDraft = () => {
  providerDraft.value = {
    id: '',
    name: '',
    apiType: 'openai-compatible',
    apiKey: '',
    baseUrl: '',
  };
};

const loadProviderConfig = async (providerId: string) => {
  try {
    const keyResponse = await configApi.getAgentProviderKey(providerId);
    if (keyResponse.code === 0 && keyResponse.data) {
      form.value.apiKey = keyResponse.data.api_key || '';
      form.value.baseUrl = keyResponse.data.base_url || '';
    } else {
      form.value.apiKey = '';
      form.value.baseUrl = '';
    }

    const response = await configApi.get();
    if (response.code === 0 && response.data) {
      const cfg = response.data.agent;
      if (cfg.provider === providerId) {
        form.value.activeModelId = cfg.model || '';
        form.value.maxTokens = cfg.max_tokens || 4096;
        form.value.temperature = cfg.temperature || 0.7;
      } else {
        const activeModel = providerModels.value.find((model) => model.id === form.value.activeModelId);
        if (!activeModel && providerModels.value.length > 0) {
          form.value.activeModelId = providerModels.value[0].id;
        }
        form.value.maxTokens = 4096;
        form.value.temperature = 0.7;
      }
    }
  } catch (error) {
    console.error('Failed to load provider config:', error);
  }
};

const selectProvider = async (id: string) => {
  if (providerDraftVisible.value) {
    return;
  }
  selectedProviderId.value = id;
  await loadProviderConfig(id);
};

const openAddProvider = () => {
  editingProviderId.value = '';
  selectedProviderId.value = '';
  resetProviderDraft();
  providerDraftVisible.value = true;
};

const openEditProvider = async (provider: ProviderInfo) => {
  editingProviderId.value = provider.id;
  selectedProviderId.value = provider.id;
  const response = await configApi.getAgentProviderKey(provider.id);
  providerDraft.value = {
    id: provider.id,
    name: provider.name,
    apiType: response.data?.api_type || provider.api_type || 'openai-compatible',
    apiKey: response.data?.api_key || '',
    baseUrl: response.data?.base_url || '',
  };
  providerDraftVisible.value = true;
};

const cancelProviderForm = () => {
  editingProviderId.value = '';
  providerDraftVisible.value = false;
  resetProviderDraft();
  if (!selectedProviderId.value && providerList.value.length > 0) {
    selectProvider(providerList.value[0].id);
  }
};

const saveProvider = async () => {
  const id = providerDraft.value.id.trim();
  const name = providerDraft.value.name.trim();
  const baseUrl = providerDraft.value.baseUrl.trim();
  if (!id || !name || !baseUrl) {
    message.warning('Channel ID, display name, and Base URL are required');
    return;
  }

  providerSaving.value = true;
  try {
    const response = await configApi.upsertAgentProvider({
      id,
      name,
      api_type: providerDraft.value.apiType,
      api_key: providerDraft.value.apiKey,
      base_url: baseUrl,
    });
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to save LLM channel');
    }
    await agentStore.loadProviders();
    await configApi.updateAgent({ provider: id });
    providerDraftVisible.value = false;
    editingProviderId.value = '';
    selectedProviderId.value = id;
    await loadProviderConfig(id);
    message.success('LLM channel saved');
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Failed to save LLM channel');
  } finally {
    providerSaving.value = false;
  }
};

const deleteProvider = async (providerId: string) => {
  deletingProviderId.value = providerId;
  try {
    const response = await configApi.deleteAgentProvider(providerId);
    if (response.code !== 0) {
      throw new Error(response.message || 'Failed to delete LLM channel');
    }
    await agentStore.loadProviders();
    const nextProvider = providerList.value[0]?.id || '';
    selectedProviderId.value = nextProvider;
    if (nextProvider) {
      await loadProviderConfig(nextProvider);
    }
    message.success('LLM channel deleted');
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Failed to delete LLM channel');
  } finally {
    deletingProviderId.value = '';
  }
};

const openAddModel = () => {
  editingModelId.value = '';
  resetModelDraft();
  modelFormVisible.value = true;
};

const openEditModel = (model: AgentModel) => {
  editingModelId.value = model.id;
  modelDraft.value = {
    id: model.id,
    name: model.name || model.id,
    context: 128000,
    output: model.max_tokens || 4096,
  };
  modelFormVisible.value = true;
};

const cancelModelForm = () => {
  editingModelId.value = '';
  modelFormVisible.value = false;
  resetModelDraft();
};

const saveModel = async () => {
  const id = modelDraft.value.id.trim();
  const name = modelDraft.value.name.trim();
  if (!id || !name) {
    message.warning('Model ID and display name are required');
    return;
  }

  modelSaving.value = true;
  try {
    await agentStore.addModel({
      provider: selectedProviderId.value,
      id,
      name,
      context: modelDraft.value.context,
      output: modelDraft.value.output,
    });
    form.value.activeModelId = id;
    cancelModelForm();
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Failed to save model');
  } finally {
    modelSaving.value = false;
  }
};

const deleteModel = async (modelId: string) => {
  deletingModelId.value = modelId;
  try {
    await agentStore.removeModel(selectedProviderId.value, modelId);
    if (form.value.activeModelId === modelId) {
      form.value.activeModelId = providerModels.value[0]?.id || '';
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Failed to delete model');
  } finally {
    deletingModelId.value = '';
  }
};

const handleSave = async () => {
  saving.value = true;

  try {
    await configApi.updateAgent({
      provider: selectedProviderId.value,
      api_key: form.value.apiKey || undefined,
      base_url: form.value.baseUrl || undefined,
      model: form.value.activeModelId || undefined,
      max_tokens: form.value.maxTokens,
      temperature: form.value.temperature,
    });

    message.success('Model configuration saved');
    await agentStore.loadProviders();
    await agentStore.loadModels(true);
    emit('update:visible', false);
    emit('saved');
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Failed to save configuration');
  } finally {
    saving.value = false;
  }
};

const handleOk = () => {
  handleSave();
};

const handleCancel = () => {
  emit('update:visible', false);
};

const resetAllState = () => {
  selectedProviderId.value = '';
  form.value = { apiKey: '', baseUrl: '', activeModelId: '', maxTokens: 4096, temperature: 0.7 };
  providerDraftVisible.value = false;
  modelFormVisible.value = false;
  editingProviderId.value = '';
  editingModelId.value = '';
  deletingModelId.value = '';
  deletingProviderId.value = '';
  saving.value = false;
  modelSaving.value = false;
  providerSaving.value = false;
  resetProviderDraft();
  resetModelDraft();
};

watch(() => props.visible, async (visible) => {
  if (!visible) {
    resetAllState();
    return;
  }

  modalKey.value++;
  await agentStore.loadProviders();
  await agentStore.loadModels(true);

  const activeProvider = providerList.value.find((provider) => provider.is_active);
  if (activeProvider) {
    selectProvider(activeProvider.id);
  } else if (providerList.value.length > 0) {
    selectProvider(providerList.value[0].id);
  }
});
</script>

<style module>
.layout {
  display: flex;
  min-height: 430px;
  max-height: 64vh;
  margin: -6px -10px -10px;
  background: #d8d8d8;
  border-top: 1px solid #a8a8a8;
  color: #1f1f1f;
}

.providerList {
  width: 220px;
  padding: 8px;
  overflow-y: auto;
  background: linear-gradient(180deg, #e5e5e5 0%, #d8d8d8 100%);
  border-right: 1px solid #9f9f9f;
}

.providerTools {
  margin-bottom: 8px;
}

.providerItem {
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr) auto auto auto;
  align-items: center;
  gap: 6px;
  padding: 7px 8px;
  margin-bottom: 4px;
  cursor: pointer;
  background: #eeeeee;
  border: 1px solid #b9b9b9;
  border-radius: 2px;
}

.providerItem:hover {
  background: #f5f5f5;
  border-color: #9f9f9f;
}

.providerItem.active {
  background: #d7e5f8;
  border-color: #7d9fc8;
  box-shadow: inset 3px 0 0 #2f6faf;
}

.providerName {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  font-size: 12px;
}

.providerActions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  padding-top: 4px;
  border-top: 1px solid rgba(0, 0, 0, 0.08);
}

.statusDot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #2f9a45;
}

.modelBadge {
  padding: 1px 5px;
  border: 1px solid #adadad;
  border-radius: 2px;
  background: #f6f6f6;
  font-size: 10px;
  line-height: 14px;
  color: #3a3a3a;
}

.providerConfig {
  flex: 1;
  padding: 10px 12px;
  overflow-y: auto;
  background: #eeeeee;
}

.providerConfig h4 {
  margin: 0 0 10px;
  padding: 0 0 7px;
  border-bottom: 1px solid #b6b6b6;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.formItem {
  margin-bottom: 9px;
}

.formItem :global(.ant-form-item-label > label) {
  height: 18px;
  color: #333;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.providerConfig :global(.ant-input),
.providerConfig :global(.ant-input-affix-wrapper),
.providerConfig :global(.ant-input-number),
.providerConfig :global(.ant-input-number-input),
.providerConfig :global(.ant-input[disabled]) {
  border-radius: 2px;
  font-size: 12px;
}

.providerConfig :global(.ant-input),
.providerConfig :global(.ant-input-affix-wrapper),
.providerConfig :global(.ant-input-number) {
  border-color: #adadad;
  background: #fbfbfb;
}

.sectionHeader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 12px 0 6px;
  padding: 5px 7px;
  background: linear-gradient(180deg, #e3e3e3 0%, #d6d6d6 100%);
  border: 1px solid #b4b4b4;
  font-size: 12px;
  font-weight: 700;
}

.modelList {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 210px;
  overflow-y: auto;
  border: 1px solid #b4b4b4;
  padding: 6px;
  background: #dedede;
}

.modelItem {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 7px 8px;
  cursor: pointer;
  background: #f4f4f4;
  border: 1px solid #bebebe;
  border-radius: 2px;
}

.modelItem:hover {
  border-color: #8f8f8f;
  background: #fbfbfb;
}

.modelItem.activeModel {
  border-color: #6e94bd;
  background: #dce9f8;
  box-shadow: inset 3px 0 0 #2f6faf;
}

.modelInfo {
  flex: 1;
  min-width: 0;
}

.modelActions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.modelName {
  font-size: 12px;
  font-weight: 700;
  color: #222;
}

.modelMeta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 2px;
  font-size: 11px;
  color: #666;
}

.noModels {
  padding: 16px;
  text-align: center;
  color: #666;
  background: #e6e6e6;
  border: 1px dashed #aaa;
}

.emptyState {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
  text-align: center;
}

.emptyTitle {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px 0;
}

.emptyHint {
  font-size: 12px;
  color: #888;
  margin: 0;
}

.modelForm {
  margin-top: 8px;
  padding: 10px;
  border: 1px solid #b5b5b5;
  border-radius: 2px;
  background: #e8e8e8;
}

.modelFormTitle {
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid #c1c1c1;
  font-size: 12px;
  font-weight: 700;
}

.toolButton,
.compactButton,
.primaryButton,
.secondaryButton,
.inlineAction {
  height: 24px;
  padding: 0 10px;
  border: 1px solid #9f9f9f;
  border-radius: 2px;
  background: linear-gradient(180deg, #f8f8f8 0%, #dfdfdf 100%);
  color: #202020;
  font-size: 12px;
  line-height: 22px;
  cursor: pointer;
}

.toolButton {
  width: 100%;
  font-weight: 700;
}

.compactButton {
  height: 22px;
  line-height: 20px;
}

.primaryButton {
  border-color: #2f6faf;
  background: linear-gradient(180deg, #3f7fbd 0%, #286aa8 100%);
  color: #fff;
  font-weight: 700;
}

.secondaryButton {
  background: linear-gradient(180deg, #f7f7f7 0%, #e1e1e1 100%);
}

.inlineAction {
  height: 20px;
  padding: 0 7px;
  background: #ececec;
  line-height: 18px;
  font-size: 11px;
}

.dangerAction {
  color: #9f2727;
  border-color: #bd9b9b;
  background: #f2e7e7;
}

.toolButton:hover,
.compactButton:hover,
.secondaryButton:hover,
.inlineAction:hover {
  border-color: #777;
  background: #f8f8f8;
}

.primaryButton:hover {
  background: #3478bd;
}

.primaryButton:disabled,
.secondaryButton:disabled,
.inlineAction:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.formActions,
.footerActions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.footerActions {
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid #b6b6b6;
}
</style>
