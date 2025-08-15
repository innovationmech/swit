<template>
  <div class="feature-card">
    <div class="feature-icon">
      {{ icon }}
    </div>
    <div class="feature-content">
      <h3 class="feature-title">{{ title }}</h3>
      <p class="feature-description">{{ description }}</p>
      <a v-if="link" :href="link" class="feature-link">
        了解更多 →
      </a>
    </div>
  </div>
</template>

<script>
export default {
  name: 'FeatureCard',
  props: {
    icon: {
      type: String,
      required: true
    },
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      required: true
    },
    link: {
      type: String,
      default: ''
    }
  }
}
</script>

<style scoped>
.feature-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 2rem;
  border: 1px solid var(--vp-c-border);
  border-radius: var(--swit-radius-lg);
  background: var(--vp-c-bg-soft);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.feature-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--swit-primary), var(--swit-secondary));
  transform: scaleX(0);
  transition: transform 0.3s ease;
}

.feature-card:hover {
  border-color: var(--vp-c-brand-1);
  box-shadow: var(--swit-shadow-lg);
  transform: translateY(-4px);
}

.feature-card:hover::before {
  transform: scaleX(1);
}

.feature-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  height: 4rem;
  width: 4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--swit-primary), var(--swit-secondary));
  border-radius: 50%;
  background-size: 200% 200%;
  animation: gradientShift 3s ease infinite;
}

@keyframes gradientShift {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}

.feature-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.feature-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0 0 1rem 0;
}

.feature-description {
  color: var(--vp-c-text-2);
  line-height: 1.6;
  margin: 0 0 1.5rem 0;
  flex: 1;
}

.feature-link {
  color: var(--vp-c-brand-1);
  text-decoration: none;
  font-weight: 600;
  transition: all 0.3s ease;
  position: relative;
}

.feature-link::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  width: 0;
  height: 2px;
  background: var(--vp-c-brand-1);
  transition: width 0.3s ease;
}

.feature-link:hover {
  color: var(--vp-c-brand-2);
}

.feature-link:hover::after {
  width: 100%;
}

/* Dark mode adaptations */
.dark .feature-card {
  border-color: var(--vp-c-border-dark);
  background: var(--vp-c-bg-alt);
}

.dark .feature-card:hover {
  border-color: var(--vp-c-brand-1);
}

/* Responsive design */
@media (max-width: 768px) {
  .feature-card {
    padding: 1.5rem;
  }
  
  .feature-icon {
    font-size: 2.5rem;
    height: 3.5rem;
    width: 3.5rem;
  }
  
  .feature-title {
    font-size: 1.25rem;
  }
}

/* Animation on scroll */
.feature-card {
  opacity: 0;
  transform: translateY(20px);
  animation: fadeInUp 0.6s ease-out forwards;
}

.feature-card:nth-child(2) {
  animation-delay: 0.1s;
}

.feature-card:nth-child(3) {
  animation-delay: 0.2s;
}

.feature-card:nth-child(4) {
  animation-delay: 0.3s;
}

.feature-card:nth-child(5) {
  animation-delay: 0.4s;
}

.feature-card:nth-child(6) {
  animation-delay: 0.5s;
}

@keyframes fadeInUp {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>