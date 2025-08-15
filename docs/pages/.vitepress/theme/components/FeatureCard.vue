<template>
  <div class="feature-card">
    <div v-if="feature.icon" class="feature-icon">
      {{ feature.icon }}
    </div>
    
    <h3 class="feature-title">
      <a v-if="feature.link" :href="feature.link" class="feature-link">
        {{ feature.title }}
      </a>
      <span v-else>{{ feature.title }}</span>
    </h3>
    
    <p class="feature-description">{{ feature.details }}</p>
    
    <a v-if="feature.link" :href="feature.link" class="learn-more-link">
      Learn More →
    </a>
  </div>
</template>

<script setup>
const props = defineProps({
  feature: {
    type: Object,
    required: true,
    validator(value) {
      return value.title && value.details
    }
  }
})
</script>

<style scoped>
.feature-card {
  padding: 2rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  transition: all 0.3s ease;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.feature-card:hover {
  border-color: var(--vp-c-brand-1);
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
}

.feature-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 4rem;
  height: 4rem;
  border-radius: 50%;
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

.feature-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--vp-c-text-1);
}

.feature-link {
  color: inherit;
  text-decoration: none;
  transition: color 0.2s;
}

.feature-link:hover {
  color: var(--vp-c-brand-1);
}

.feature-description {
  color: var(--vp-c-text-2);
  line-height: 1.6;
  margin-bottom: auto;
  flex: 1;
}

.learn-more-link {
  display: inline-flex;
  align-items: center;
  color: var(--vp-c-brand-1);
  text-decoration: none;
  font-weight: 500;
  margin-top: 1rem;
  transition: color 0.2s;
}

.learn-more-link:hover {
  color: var(--vp-c-brand-2);
}
</style>