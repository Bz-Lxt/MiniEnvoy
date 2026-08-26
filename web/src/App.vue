<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import ScopeChart from "./components/ScopeChart.vue";
import HealthGauge from "./components/HealthGauge.vue";
import TopologyView from "./components/TopologyView.vue";
import ConfirmDialog from "./components/ConfirmDialog.vue";
import {
  ejectUpstream,
  fetchSnapshot,
  formatBps,
  restoreUpstream,
  type Snapshot,
  type Upstream,
} from "./api/client";

const snap = ref<Snapshot | null>(null);
const series = ref<number[]>(Array.from({ length: 300 }, () => 0));
const live = ref<"live" | "stale" | "down">("down");
const clock = ref("");
const selected = ref("");
const toasts = ref<{ id: number; text: string }[]>([]);
const dialog = ref<null | { kind: "eject" | "restore"; u: Upstream }>(null);
let toastId = 1;
let es: EventSource | null = null;
let pollTimer = 0;
let clockTimer = 0;
let lastAt = 0;

function apply(s: Snapshot) {
  snap.value = s;
  lastAt = Date.now();
  live.value = "live";
  const next = series.value.slice(1);
  next.push(s.overview.in_bps || 0);
  series.value = next;
}

function toast(text: string) {
  const id = toastId++;
  toasts.value.push({ id, text });
  window.setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }, 5000);
}

async function poll() {
  try {
    apply(await fetchSnapshot());
  } catch (e) {
    live.value = "down";
    toast(e instanceof Error ? e.message : "拉取指标失败");
  }
}

function startSSE() {
  es?.close();
  es = new EventSource("/api/v1/events");
  es.addEventListener("snapshot", (ev) => {
    try {
      apply(JSON.parse((ev as MessageEvent).data));
    } catch {
      /* ignore */
    }
  });
  es.onerror = () => {
    live.value = "stale";
    es?.close();
    es = null;
    window.setTimeout(startSSE, 2000);
  };
}

function tickClock() {
  const now = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  clock.value = `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())} ${p(now.getHours())}:${p(now.getMinutes())}:${p(now.getSeconds())}`;
  if (lastAt && Date.now() - lastAt > 3000) {
    live.value = live.value === "down" ? "down" : "stale";
    if (!pollTimer) {
      poll();
      pollTimer = window.setInterval(poll, 1000);
    }
  } else if (live.value === "live" && pollTimer) {
    window.clearInterval(pollTimer);
    pollTimer = 0;
  }
}

const inB = computed(() => formatBps(snap.value?.overview.in_bps || 0));
const outB = computed(() => formatBps(snap.value?.overview.out_bps || 0));
const selectedNode = computed(() => snap.value?.topology.nodes.find((n) => n.id === selected.value));

async function confirm() {
  const d = dialog.value;
  if (!d) return;
  dialog.value = null;
  try {
    if (d.kind === "eject") await ejectUpstream(d.u.id, "console eject");
    else await restoreUpstream(d.u.id);
    await poll();
  } catch (e) {
    toast(e instanceof Error ? e.message : "操作失败");
  }
}

onMounted(() => {
  tickClock();
  clockTimer = window.setInterval(tickClock, 1000);
  poll();
  startSSE();
});
onUnmounted(() => {
  es?.close();
  window.clearInterval(clockTimer);
  if (pollTimer) window.clearInterval(pollTimer);
});
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <h1>MINI ENVOY</h1>
        <span>ZERO-COPY GATEWAY CONSOLE</span>
      </div>
      <div class="live">
        <span class="dot" :class="live"></span>
        <span v-if="live === 'live'">LIVE</span>
        <span v-else-if="live === 'stale'">数据已过期 / 正在重连</span>
        <span v-else>数据面不可达</span>
        <span class="mono">{{ clock }}</span>
      </div>
    </header>

    <main class="workspace">
      <section class="kpis">
        <article class="panel kpi">
          <div class="label">并发连接</div>
          <div class="value mono">{{ snap?.overview.conns ?? 0 }}</div>
        </article>
        <article class="panel kpi">
          <div class="label">入站吞吐</div>
          <div class="value mono">{{ inB.value }}<span class="unit">{{ inB.unit }}</span></div>
        </article>
        <article class="panel kpi">
          <div class="label">出站吞吐 / PPS</div>
          <div class="value mono">{{ outB.value }}<span class="unit">{{ outB.unit }}</span></div>
          <div class="label">IN {{ (snap?.overview.in_pps || 0).toFixed(1) }} · OUT {{ (snap?.overview.out_pps || 0).toFixed(1) }}</div>
        </article>
        <article class="panel kpi">
          <div class="label">错误率 · 缓冲占用</div>
          <div class="value mono">{{ (snap?.overview.error_rate || 0).toFixed(2) }}<span class="unit">err/s</span></div>
          <div class="label">
            零额外复制环 {{ ((snap?.overview.ring_ratio || 0) * 100).toFixed(1) }}%
            · Reactor {{ snap?.overview.reactors ?? 0 }}
          </div>
        </article>
      </section>

      <section class="grid-2">
        <article class="panel">
          <h2>吞吐示波器 · 5 分钟窗口</h2>
          <ScopeChart :points="series" />
        </article>
        <article class="panel">
          <h2>连接池健康仪</h2>
          <HealthGauge :items="snap?.upstreams || []" />
        </article>
      </section>

      <article class="panel">
        <h2>动态代理拓扑</h2>
        <TopologyView :data="snap?.topology || null" :selected="selected" @select="selected = $event" />
        <p v-if="selectedNode" class="label">
          选中 {{ selectedNode.label }} · {{ selectedNode.status }}
          <template v-if="selectedNode.reason"> · {{ selectedNode.reason }}</template>
        </p>
      </article>

      <article class="panel">
        <h2>上游池</h2>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th><th>地址</th><th>权重</th><th>状态</th><th>活动</th><th>空闲</th><th>排队</th><th>失败</th><th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in snap?.upstreams || []" :key="u.id">
                <td class="mono">{{ u.id }}</td>
                <td class="mono">{{ u.host }}:{{ u.port }}</td>
                <td>{{ u.weight }}</td>
                <td><span class="badge" :class="u.state">{{ u.state }}</span></td>
                <td>{{ u.active }}</td>
                <td>{{ u.idle }}</td>
                <td>{{ u.queued }}</td>
                <td>{{ u.fails }}</td>
                <td>
                  <button
                    v-if="u.state !== 'ejected'"
                    class="icon-btn"
                    type="button"
                    aria-label="摘除上游"
                    @click="dialog = { kind: 'eject', u }"
                  >摘除</button>
                  <button
                    v-else
                    class="icon-btn"
                    type="button"
                    aria-label="恢复上游"
                    @click="dialog = { kind: 'restore', u }"
                  >恢复</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>
    </main>

    <ConfirmDialog
      v-if="dialog"
      :title="dialog.kind === 'eject' ? '确认摘除上游' : '确认恢复上游'"
      :message="dialog.kind === 'eject'
        ? `立即停止向 ${dialog.u.id} 分配新请求，已有请求按宽限期排空。`
        : `${dialog.u.id} 将进入 probing，探测成功后恢复 healthy。`"
      :danger="dialog.kind === 'eject'"
      :confirm-text="dialog.kind === 'eject' ? '确认摘除' : '确认恢复'"
      @cancel="dialog = null"
      @confirm="confirm"
    />

    <div class="toasts">
      <div v-for="t in toasts" :key="t.id" class="toast">
        <span>{{ t.text }}</span>
        <button type="button" aria-label="关闭提示" @click="toasts = toasts.filter(x => x.id !== t.id)">×</button>
      </div>
    </div>
  </div>
</template>
