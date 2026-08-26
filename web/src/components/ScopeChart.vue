<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue";

const props = defineProps<{ points: number[] }>();
const canvas = ref<HTMLCanvasElement | null>(null);
let raf = 0;

function draw() {
  const el = canvas.value;
  if (!el) return;
  const dpr = window.devicePixelRatio || 1;
  const w = el.clientWidth;
  const h = el.clientHeight;
  el.width = Math.floor(w * dpr);
  el.height = Math.floor(h * dpr);
  const ctx = el.getContext("2d");
  if (!ctx) return;
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, w, h);
  ctx.strokeStyle = "rgba(62,224,200,0.12)";
  ctx.lineWidth = 1;
  for (let x = 0; x < w; x += 28) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, h);
    ctx.stroke();
  }
  for (let y = 0; y < h; y += 28) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
    ctx.stroke();
  }
  const data = props.points;
  const max = Math.max(1, ...data);
  ctx.beginPath();
  ctx.strokeStyle = "#3ee0c8";
  ctx.lineWidth = 1.6;
  data.forEach((v, i) => {
    const x = (i / Math.max(data.length - 1, 1)) * w;
    const y = h - (v / max) * (h - 12) - 6;
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.stroke();
  const scan = (Date.now() / 12) % w;
  ctx.fillStyle = "rgba(62,224,200,0.08)";
  ctx.fillRect(scan, 0, 18, h);
}

function loop() {
  draw();
  raf = requestAnimationFrame(loop);
}

onMounted(() => {
  loop();
});
onUnmounted(() => cancelAnimationFrame(raf));
watch(() => props.points, draw, { deep: true });
</script>

<template>
  <div class="scope">
    <canvas ref="canvas" role="img" aria-label="五分钟吞吐示波器"></canvas>
  </div>
</template>
