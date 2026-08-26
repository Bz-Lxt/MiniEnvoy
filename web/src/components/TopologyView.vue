<script setup lang="ts">
import { computed } from "vue";
import type { Topology } from "../api/client";

const props = defineProps<{ data: Topology | null; selected: string }>();
const emit = defineEmits<{ (e: "select", id: string): void }>();

const layout = computed(() => {
  const cols: Record<string, number> = {
    client: 40,
    gateway: 180,
    reactor: 320,
    route: 470,
    pool: 620,
    upstream: 780,
  };
  const counts: Record<string, number> = {};
  const pos: Record<string, { x: number; y: number }> = {};
  for (const n of props.data?.nodes || []) {
    counts[n.kind] = (counts[n.kind] || 0) + 1;
  }
  const seen: Record<string, number> = {};
  for (const n of props.data?.nodes || []) {
    const i = seen[n.kind] || 0;
    seen[n.kind] = i + 1;
    const total = counts[n.kind] || 1;
    pos[n.id] = { x: cols[n.kind] || 100, y: 40 + ((i + 1) * 280) / (total + 1) };
  }
  return pos;
});

function color(status: string) {
  if (status === "healthy") return "#3ee0c8";
  if (status === "degraded" || status === "probing") return "#e8c36a";
  return "#ff5d6c";
}
</script>

<template>
  <div class="topo">
    <svg role="img" aria-label="代理拓扑">
      <line
        v-for="e in data?.edges || []"
        :key="e.id"
        :x1="layout[e.from]?.x"
        :y1="layout[e.from]?.y"
        :x2="layout[e.to]?.x"
        :y2="layout[e.to]?.y"
        class="edge"
        :class="{ hot: e.pps > 0 || e.bps > 0, bad: e.status === 'down' || e.status === 'ejected' }"
      />
      <g
        v-for="n in data?.nodes || []"
        :key="n.id"
        :transform="`translate(${layout[n.id]?.x || 0}, ${layout[n.id]?.y || 0})`"
        @click="emit('select', n.id)"
        style="cursor: pointer"
      >
        <rect
          class="node-card"
          :class="{ active: selected === n.id }"
          x="-58"
          y="-22"
          width="116"
          height="44"
          :stroke="color(n.status)"
        />
        <text class="node-title" x="0" y="-2" text-anchor="middle">{{ n.label }}</text>
        <text class="node-sub" x="0" y="14" text-anchor="middle">{{ n.status }}</text>
      </g>
    </svg>
  </div>
</template>
