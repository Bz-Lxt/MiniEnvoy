<script setup lang="ts">
import type { Upstream } from "../api/client";

const props = defineProps<{ items: Upstream[] }>();

function ratio() {
  if (!props.items.length) return 0;
  const ok = props.items.filter((u) => u.state === "healthy" || u.state === "degraded").length;
  return ok / props.items.length;
}
function sum(key: keyof Upstream) {
  return props.items.reduce((n, u) => n + Number(u[key] || 0), 0);
}
function dash() {
  const r = 54;
  const c = 2 * Math.PI * r;
  return `${c * ratio()} ${c}`;
}
</script>

<template>
  <div class="health">
    <svg class="ring" viewBox="0 0 140 140" role="img" aria-label="连接池健康比例">
      <circle cx="70" cy="70" r="54" fill="none" stroke="#243040" stroke-width="10" />
      <circle
        cx="70"
        cy="70"
        r="54"
        fill="none"
        stroke="#3ee0c8"
        stroke-width="10"
        :stroke-dasharray="dash()"
        stroke-linecap="butt"
        transform="rotate(-90 70 70)"
      />
      <text x="70" y="74" text-anchor="middle" fill="#3ee0c8" font-size="22" font-family="IBM Plex Mono">
        {{ Math.round(ratio() * 100) }}%
      </text>
    </svg>
    <div class="stats">
      <div>活动连接 <b class="mono">{{ sum("active") }}</b></div>
      <div>空闲连接 <b class="mono">{{ sum("idle") }}</b></div>
      <div>排队量 <b class="mono">{{ sum("queued") }}</b></div>
      <div>失败累计 <b class="mono">{{ sum("fails") }}</b></div>
    </div>
  </div>
</template>
