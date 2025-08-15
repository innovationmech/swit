<template>
  <svg 
    :width="size" 
    :height="size" 
    :viewBox="icons[name]?.viewBox || '0 0 24 24'" 
    fill="none" 
    xmlns="http://www.w3.org/2000/svg"
    :class="$attrs.class"
  >
    <path 
      v-if="icons[name]"
      :d="icons[name].path" 
      :stroke="icons[name].stroke ? 'currentColor' : undefined"
      :fill="icons[name].fill ? 'currentColor' : 'none'"
      :stroke-width="icons[name].strokeWidth || 2"
      :stroke-linecap="icons[name].strokeLinecap || 'round'"
      :stroke-linejoin="icons[name].strokeLinejoin || 'round'"
    />
  </svg>
</template>

<script setup>
defineProps({
  name: {
    type: String,
    required: true
  },
  size: {
    type: [String, Number],
    default: 24
  }
})

// 图标库
const icons = {
  // 基础图标
  'x': {
    viewBox: '0 0 24 24',
    path: 'M18 6L6 18M6 6l12 12',
    stroke: true
  },
  'check': {
    viewBox: '0 0 24 24',
    path: 'M20 6L9 17l-5-5',
    stroke: true
  },
  'chevron-up': {
    viewBox: '0 0 24 24',
    path: 'M18 15l-6-6-6 6',
    stroke: true
  },
  'chevron-down': {
    viewBox: '0 0 24 24',
    path: 'M6 9l6 6 6-6',
    stroke: true
  },
  'arrow-left': {
    viewBox: '0 0 24 24',
    path: 'M19 12H5M12 19l-7-7 7-7',
    stroke: true
  },
  
  // 功能图标
  'activity': {
    viewBox: '0 0 24 24',
    path: 'M22 12h-4l-3 9L9 3l-3 9H2',
    stroke: true
  },
  'refresh': {
    viewBox: '0 0 24 24',
    path: 'M23 4v6h-6M1 20v-6h6M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4a9 9 0 0 1-14.85 4.36L23 14',
    stroke: true
  },
  'shield': {
    viewBox: '0 0 24 24',
    path: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
    stroke: true
  },
  'message-circle': {
    viewBox: '0 0 24 24',
    path: 'M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z',
    stroke: true
  },
  'star': {
    viewBox: '0 0 24 24',
    path: 'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z',
    stroke: true
  },
  'loader': {
    viewBox: '0 0 24 24',
    path: 'M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83',
    stroke: true
  },
  
  // 状态图标
  'check-circle': {
    viewBox: '0 0 24 24',
    path: 'M22 11.08V12a10 10 0 1 1-5.93-9.14',
    stroke: true
  },
  'alert-circle': {
    viewBox: '0 0 24 24',
    path: 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z',
    stroke: true
  },
  'alert-triangle': {
    viewBox: '0 0 24 24',
    path: 'M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z',
    stroke: true
  },
  'info': {
    viewBox: '0 0 24 24',
    path: 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z',
    stroke: true
  },
  'help-circle': {
    viewBox: '0 0 24 24',
    path: 'M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z',
    stroke: true
  },
  
  // 反馈类型图标
  'bug': {
    viewBox: '0 0 24 24',
    path: 'M8 2v4m8-4v4M9 9h.01M15 9h.01M7 13h10M7 17h10M8 21l4-7 4 7M12 9V2',
    stroke: true
  },
  'lightbulb': {
    viewBox: '0 0 24 24',
    path: 'M9 21h6M12 17V9a5 5 0 1 0-5 5h10a5 5 0 1 0-5-5v8',
    stroke: true
  },
  'file-text': {
    viewBox: '0 0 24 24',
    path: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z',
    stroke: true
  },
  'heart': {
    viewBox: '0 0 24 24',
    path: 'M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z',
    stroke: true
  }
}
</script>