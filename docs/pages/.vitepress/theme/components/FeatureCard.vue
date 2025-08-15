<template>
  <div 
    class="feature-card"
    :class="{ 'feature-card-hover': isHovered }"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
  >
    <div class="feature-icon">
      <span v-if="typeof feature.icon === 'string'" class="icon-emoji">
        {{ feature.icon }}
      </span>
      <component v-else :is="feature.icon" class="icon-component" />
    </div>
    <h3 class="feature-title">
      {{ feature.titleKey }}
    </h3>
    <p class="feature-description">
      {{ feature.descriptionKey }}
    </p>
    <a 
      v-if="feature.link" 
      :href="feature.link" 
      class="feature-link"
      @click.stop
    >
      {{ linkText }} →
    </a>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useData } from 'vitepress'

export interface Feature {
  id: string
  icon: string | any
  titleKey: string
  descriptionKey: string
  link?: string
}

const props = defineProps<{
  feature: Feature
}>()

const { lang } = useData()
const isHovered = ref(false)

const linkText = computed(() => {
  return lang.value === 'zh-CN' ? '了解更多' : 'Learn More'
})
</script>

<style scoped>
.feature-card {
  position: relative;
  padding: 1.5rem;
  border-radius: var(--swit-border-radius);
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-border);
  transition: var(--swit-transition);
  cursor: default;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.feature-card-hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.1);
  border-color: var(--vp-c-brand-1);
}

.dark .feature-card-hover {
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.3);
}

.feature-icon {
  width: 56px;
  height: 56px;
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--vp-c-brand-soft);
  border-radius: 12px;
  transition: var(--swit-transition);
}

.feature-card-hover .feature-icon {
  background: var(--vp-c-brand-1);
  transform: scale(1.1);
}

.icon-emoji {
  font-size: 28px;
  line-height: 1;
}

.feature-card-hover .icon-emoji {
  filter: grayscale(0) brightness(1.2);
}

.icon-component {
  width: 32px;
  height: 32px;
  color: var(--vp-c-brand-1);
}

.feature-card-hover .icon-component {
  color: white;
}

.feature-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: var(--vp-c-text-1);
  transition: color 0.3s;
}

.feature-card-hover .feature-title {
  color: var(--vp-c-brand-1);
}

.feature-description {
  color: var(--vp-c-text-2);
  line-height: 1.7;
  flex-grow: 1;
  margin-bottom: 1rem;
}

.feature-link {
  display: inline-flex;
  align-items: center;
  color: var(--vp-c-brand-1);
  text-decoration: none;
  font-weight: 500;
  transition: var(--swit-transition);
  margin-top: auto;
}

.feature-link:hover {
  color: var(--vp-c-brand-2);
  transform: translateX(4px);
}

/* Responsive adjustments */
@media (max-width: 640px) {
  .feature-card {
    padding: 1.25rem;
  }
  
  .feature-icon {
    width: 48px;
    height: 48px;
  }
  
  .icon-emoji {
    font-size: 24px;
  }
  
  .feature-title {
    font-size: 1.125rem;
  }
}

/* Animation for hover effect */
@keyframes pulse {
  0% {
    box-shadow: 0 0 0 0 var(--vp-c-brand-soft);
  }
  70% {
    box-shadow: 0 0 0 10px transparent;
  }
  100% {
    box-shadow: 0 0 0 0 transparent;
  }
}

.feature-card-hover .feature-icon {
  animation: pulse 1.5s infinite;
}
</style>