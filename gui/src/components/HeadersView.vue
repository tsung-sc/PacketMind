<template>
  <div :class="$style.container">
    <div v-if="hasContent" :class="[$style.rows, isNarrow ? $style.narrowDetail : '']">
      <div v-if="leadLine" :class="[$style.row, $style.leadRow]">
        <div :class="$style.key"></div>
        <div :class="[$style.value, $style.leadValue]">{{ leadLine }}</div>
      </div>

      <div v-for="record in flattenHeaders" :key="record.key" :class="$style.row">
        <div :class="$style.key">{{ record.name }}</div>
        <div :class="$style.value">
          {{ record.value }}
        </div>
      </div>
    </div>

    <a-empty v-else :class="$style.empty" description="No data" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  headers: Record<string, string[]> | null;
  requestId: string;
  location?: string;
  leadLine?: string;
  isNarrow?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  location: 'Headers',
  leadLine: '',
});

const flattenHeaders = computed(() => {
  if (!props.headers) return [];

  const result: { key: string; name: string; fullName: string; value: string }[] = [];
  Object.entries(props.headers).forEach(([name, values]) => {
    if (Array.isArray(values)) {
      values.forEach((value, index) => {
        result.push({
          key: `${name}-${index}`,
          name: index === 0 ? name : '',
          fullName: name,
          value,
        });
      });
    } else {
      result.push({
        key: name,
        name,
        fullName: name,
        value: String(values),
      });
    }
  });
  return result;
});

const hasContent = computed(() => Boolean(props.leadLine || flattenHeaders.value.length > 0));
</script>

<style module>
.container {
  height: 100%;
  overflow: auto;
}

.rows {
  padding: 10px 14px 14px;
}

.row {
  display: grid;
  grid-template-columns: minmax(96px, 180px) minmax(0, 1fr);
  align-items: start;
  gap: 16px;
  padding: 5px 0;
}

.leadRow {
  padding-bottom: 10px;
}

.key {
  color: #2f2f2f;
  font-size: 12px;
  font-weight: 600;
  text-align: right;
  line-height: 1.45;
  word-break: break-word;
}

.value {
  color: #222;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-all;
  border-radius: 3px;
  padding: 0 2px;
}

.leadValue {
  padding: 0;
}

.narrowDetail .row {
  grid-template-columns: 1fr;
  gap: 2px 0;
}

.narrowDetail .key {
  text-align: left;
}

.narrowDetail .value {
  padding-left: 12px;
  word-break: break-word;
}

.empty {
  margin-top: 24px;
}
</style>
