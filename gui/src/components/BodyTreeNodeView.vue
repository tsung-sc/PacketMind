<template>
  <div :class="$style.nodeBlock">
    <div
      :class="[$style.nodeRow, { [$style.nodeBranch]: hasChildren }]"
      @click="toggle"
    >
      <span :class="$style.nodeIndent" :style="{ width: `${depth * 14}px` }"></span>
      <span v-if="hasChildren" :class="$style.toggle">{{ expanded ? '▼' : '▶' }}</span>
      <span v-else :class="$style.leafDot">•</span>
      <span :class="$style.key">{{ node.label }}</span>
      <span v-if="node.value !== undefined" :class="[$style.value, valueClass]">{{ displayValue }}</span>
      <span v-else :class="$style.meta">{{ containerMeta }}</span>
    </div>

    <div v-if="hasChildren && expanded">
      <BodyTreeNodeView
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :depth="depth + 1"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useCssModule } from 'vue';
import type { BodyTreeNode } from './bodyTreeUtils';

defineOptions({
  name: 'BodyTreeNodeView',
});

interface Props {
  node: BodyTreeNode;
  depth: number;
}

const props = defineProps<Props>();
const style = useCssModule();

const hasChildren = computed(() => Boolean(props.node.children?.length));
const expanded = ref(true);

const toggle = () => {
  if (!hasChildren.value) return;
  expanded.value = !expanded.value;
};

const displayValue = computed(() => {
  if (props.node.nodeType === 'string' || props.node.nodeType === 'xml-text' || props.node.nodeType === 'xml-cdata') {
    return `"${props.node.value || ''}"`;
  }
  return props.node.value || '';
});

const containerMeta = computed(() => {
  if (!hasChildren.value) return '';
  if (props.node.nodeType === 'array') {
    return `[${props.node.children?.length || 0}]`;
  }
  if (props.node.nodeType === 'object') {
    return `{${props.node.children?.length || 0}}`;
  }
  if (props.node.nodeType === 'xml-element') {
    return `<${props.node.label}>`;
  }
  return '';
});

const valueClass = computed(() => {
  switch (props.node.nodeType) {
    case 'string':
    case 'xml-text':
    case 'xml-cdata':
      return style.valueString;
    case 'number':
      return style.valueNumber;
    case 'boolean':
      return style.valueBoolean;
    case 'null':
      return style.valueNull;
    case 'xml-attribute':
      return style.valueAttribute;
    case 'xml-comment':
      return style.valueComment;
    default:
      return style.valueDefault;
  }
});
</script>

<style module>
.nodeBlock {
  font-size: 12px;
  line-height: 1.5;
}

.nodeRow {
  display: flex;
  align-items: baseline;
  min-height: 20px;
  color: #1f1f1f;
  white-space: nowrap;
}

.nodeBranch {
  cursor: pointer;
}

.nodeBranch:hover {
  background: #e7edf5;
}

.nodeIndent {
  flex-shrink: 0;
}

.toggle,
.leafDot {
  width: 14px;
  text-align: center;
  flex-shrink: 0;
  color: #5e5e5e;
}

.leafDot {
  font-size: 11px;
}

.key {
  color: #c05000;
  flex-shrink: 0;
}

.value,
.meta {
  margin-left: 8px;
}

.meta {
  color: #6d6d6d;
}

.valueDefault {
  color: #1f1f1f;
}

.valueString {
  color: #8b3fa0;
}

.valueNumber {
  color: #2a6cb8;
}

.valueBoolean {
  color: #7a3ea3;
}

.valueNull {
  color: #777;
}

.valueAttribute {
  color: #0f6e5f;
}

.valueComment {
  color: #8a8a8a;
  font-style: italic;
}
</style>
