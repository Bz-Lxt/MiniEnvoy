<script setup lang="ts">
defineProps<{ title: string; message: string; danger?: boolean; confirmText: string }>();
const emit = defineEmits<{ (e: "cancel"): void; (e: "confirm"): void }>();
</script>

<template>
  <div class="dialog-mask" role="dialog" aria-modal="true" :aria-label="title">
    <div class="dialog">
      <h3>{{ title }}</h3>
      <p>{{ message }}</p>
      <div class="dialog-actions">
        <button class="btn-ghost" type="button" @click="emit('cancel')">取消</button>
        <button :class="danger ? 'btn-danger' : 'btn-ok'" type="button" @click="emit('confirm')">
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>
