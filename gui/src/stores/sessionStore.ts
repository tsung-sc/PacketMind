import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Session } from '@/types';
import { sessionApi } from '@/api/wails';

export const useSessionStore = defineStore('session', () => {
  const sessions = ref<Session[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const selectedSessionId = ref<string | null>(null);

  const activeSession = computed(() => sessions.value.find(s => s.is_active) || null);
  const selectedSession = computed(() => sessions.value.find(s => s.id === selectedSessionId.value) || null);

  const setSessions = (items: Session[]) => {
    sessions.value = items;
    if (!selectedSessionId.value || !items.some(s => s.id === selectedSessionId.value)) {
      selectedSessionId.value = items.find(s => s.is_active)?.id || items[0]?.id || null;
    }
  };

  const fetchSessions = async () => {
    loading.value = true;
    error.value = null;
    try {
      const response = await sessionApi.list();
      if (response.code === 0 && response.data) {
        setSessions(response.data.items);
      } else {
        error.value = response.message || 'Failed to fetch sessions';
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch sessions';
    } finally {
      loading.value = false;
    }
  };

  const createSession = async (name: string, description?: string) => {
    const response = await sessionApi.create({ name, description });
    if (response.code === 0 && response.data) {
      await fetchSessions();
      return response.data;
    }
    throw new Error(response.message);
  };

  const deleteSession = async (id: string) => {
    const response = await sessionApi.delete(id);
    if (response.code === 0) {
      sessions.value = sessions.value.filter(s => s.id !== id);
      if (selectedSessionId.value === id) {
        selectedSessionId.value = sessions.value.find(s => s.is_active)?.id || sessions.value[0]?.id || null;
      }
      return true;
    }
    throw new Error(response.message);
  };

  const renameSession = async (id: string, name: string) => {
    const response = await sessionApi.update(id, { name });
    if (response.code === 0) {
      await fetchSessions();
      return true;
    }
    throw new Error(response.message);
  };

  const activateSession = async (id: string) => {
    const response = await sessionApi.activate(id);
    if (response.code === 0) {
      sessions.value.forEach(s => {
        s.is_active = (s.id === id);
      });
      selectedSessionId.value = id;
      return true;
    }
    throw new Error(response.message);
  };

  const selectSession = (id: string | null) => {
    selectedSessionId.value = id;
  };

  return {
    sessions,
    loading,
    error,
    activeSession,
    selectedSessionId,
    selectedSession,
    setSessions,
    fetchSessions,
    createSession,
    deleteSession,
    renameSession,
    activateSession,
    selectSession,
  };
});
