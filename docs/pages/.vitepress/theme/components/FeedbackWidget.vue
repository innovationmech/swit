<template>
  <div class="feedback-widget">
    <!-- 反馈触发按钮 -->
    <button 
      v-if="!showWidget" 
      @click="openWidget" 
      class="feedback-trigger"
      :title="isZh ? '反馈建议' : 'Feedback'"
    >
      <Icon name="message-circle" />
      <span class="trigger-text">{{ isZh ? '反馈' : 'Feedback' }}</span>
    </button>

    <!-- 反馈小组件 -->
    <Transition name="widget">
      <div v-if="showWidget" class="feedback-panel">
        <div class="feedback-header">
          <h4>{{ isZh ? '用户反馈' : 'User Feedback' }}</h4>
          <button @click="closeWidget" class="close-btn">
            <Icon name="x" />
          </button>
        </div>

        <!-- 反馈类型选择 -->
        <div v-if="currentStep === 'type'" class="feedback-step">
          <p class="step-description">
            {{ isZh 
              ? '请选择反馈类型：' 
              : 'Please select feedback type:' 
            }}
          </p>
          
          <div class="feedback-types">
            <button 
              v-for="type in feedbackTypes" 
              :key="type.value"
              @click="selectFeedbackType(type.value)"
              class="feedback-type-btn"
            >
              <Icon :name="type.icon" class="type-icon" />
              <div class="type-content">
                <div class="type-title">{{ isZh ? type.titleZh : type.title }}</div>
                <div class="type-description">{{ isZh ? type.descriptionZh : type.description }}</div>
              </div>
            </button>
          </div>
        </div>

        <!-- 反馈内容填写 -->
        <div v-if="currentStep === 'content'" class="feedback-step">
          <div class="step-header">
            <button @click="goBack" class="back-btn">
              <Icon name="arrow-left" />
            </button>
            <h5>{{ getSelectedTypeTitle() }}</h5>
          </div>

          <form @submit.prevent="submitFeedback" class="feedback-form">
            <!-- 页面信息显示 -->
            <div class="page-info">
              <label>{{ isZh ? '当前页面：' : 'Current page:' }}</label>
              <input 
                v-model="feedback.page" 
                type="text" 
                readonly 
                class="page-input"
              >
            </div>

            <!-- 反馈内容 -->
            <div class="form-group">
              <label for="feedback-content">
                {{ isZh ? '详细描述：' : 'Description:' }}
                <span class="required">*</span>
              </label>
              <textarea
                id="feedback-content"
                v-model="feedback.content"
                :placeholder="getContentPlaceholder()"
                required
                rows="4"
                maxlength="1000"
                class="feedback-textarea"
              ></textarea>
              <div class="char-count">{{ feedback.content.length }}/1000</div>
            </div>

            <!-- 联系方式（可选） -->
            <div class="form-group">
              <label for="feedback-contact">
                {{ isZh ? '联系方式（可选）：' : 'Contact (optional):' }}
              </label>
              <input
                id="feedback-contact"
                v-model="feedback.contact"
                type="email"
                :placeholder="isZh ? '邮箱地址' : 'Email address'"
                class="feedback-input"
              >
              <div class="contact-note">
                {{ isZh 
                  ? '如需回复，请留下邮箱地址。我们承诺保护您的隐私。' 
                  : 'Leave your email if you want a response. We promise to protect your privacy.' 
                }}
              </div>
            </div>

            <!-- 评分（对于满意度反馈） -->
            <div v-if="feedback.type === 'satisfaction'" class="form-group">
              <label>{{ isZh ? '整体评分：' : 'Overall rating:' }}</label>
              <div class="rating-stars">
                <button
                  v-for="star in 5"
                  :key="star"
                  type="button"
                  @click="setRating(star)"
                  class="star-btn"
                  :class="{ active: star <= feedback.rating }"
                >
                  <Icon name="star" />
                </button>
              </div>
            </div>

            <!-- 提交按钮 -->
            <div class="form-actions">
              <button 
                type="submit" 
                :disabled="!feedback.content.trim() || isSubmitting"
                class="submit-btn"
              >
                <Icon v-if="isSubmitting" name="loader" class="spinner" />
                {{ isSubmitting 
                  ? (isZh ? '提交中...' : 'Submitting...') 
                  : (isZh ? '提交反馈' : 'Submit Feedback') 
                }}
              </button>
            </div>
          </form>
        </div>

        <!-- 提交成功 -->
        <div v-if="currentStep === 'success'" class="feedback-step success-step">
          <div class="success-icon">
            <Icon name="check-circle" />
          </div>
          <h5>{{ isZh ? '反馈提交成功！' : 'Feedback submitted successfully!' }}</h5>
          <p>
            {{ isZh 
              ? '感谢您的反馈，我们会认真考虑您的建议。' 
              : 'Thank you for your feedback. We will carefully consider your suggestions.' 
            }}
          </p>
          <button @click="resetWidget" class="new-feedback-btn">
            {{ isZh ? '提交新反馈' : 'Submit new feedback' }}
          </button>
        </div>

        <!-- 提交失败 -->
        <div v-if="currentStep === 'error'" class="feedback-step error-step">
          <div class="error-icon">
            <Icon name="alert-circle" />
          </div>
          <h5>{{ isZh ? '提交失败' : 'Submission failed' }}</h5>
          <p>{{ errorMessage }}</p>
          <div class="error-actions">
            <button @click="retrySubmission" class="retry-btn">
              {{ isZh ? '重试' : 'Retry' }}
            </button>
            <button @click="goBack" class="back-btn">
              {{ isZh ? '返回' : 'Back' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 背景遮罩 -->
    <Transition name="overlay">
      <div v-if="showWidget" @click="closeWidget" class="feedback-overlay"></div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useData, useRoute } from 'vitepress'

interface FeedbackData {
  type: string
  page: string
  content: string
  contact: string
  rating: number
  timestamp: number
  userAgent: string
  language: string
}

interface FeedbackType {
  value: string
  title: string
  titleZh: string
  description: string
  descriptionZh: string
  icon: string
}

// 组件状态
const showWidget = ref(false)
const currentStep = ref<'type' | 'content' | 'success' | 'error'>('type')
const isSubmitting = ref(false)
const errorMessage = ref('')

// 反馈数据
const feedback = reactive<FeedbackData>({
  type: '',
  page: '',
  content: '',
  contact: '',
  rating: 0,
  timestamp: 0,
  userAgent: '',
  language: ''
})

// VitePress 数据
const { lang } = useData()
const route = useRoute()

// 计算属性
const isZh = computed(() => lang.value === 'zh')

// 反馈类型配置
const feedbackTypes: FeedbackType[] = [
  {
    value: 'bug',
    title: 'Bug Report',
    titleZh: '错误报告',
    description: 'Report a bug or issue',
    descriptionZh: '报告错误或问题',
    icon: 'bug'
  },
  {
    value: 'feature',
    title: 'Feature Request',
    titleZh: '功能建议',
    description: 'Suggest a new feature',
    descriptionZh: '建议新功能',
    icon: 'lightbulb'
  },
  {
    value: 'content',
    title: 'Content Feedback',
    titleZh: '内容反馈',
    description: 'Feedback on documentation',
    descriptionZh: '文档内容反馈',
    icon: 'file-text'
  },
  {
    value: 'satisfaction',
    title: 'General Feedback',
    titleZh: '满意度评价',
    description: 'Overall experience feedback',
    descriptionZh: '整体体验反馈',
    icon: 'heart'
  }
]

// 小组件控制
function openWidget() {
  showWidget.value = true
  resetForm()
}

function closeWidget() {
  showWidget.value = false
  currentStep.value = 'type'
}

function resetWidget() {
  currentStep.value = 'type'
  resetForm()
}

function resetForm() {
  feedback.type = ''
  feedback.page = getPageInfo()
  feedback.content = ''
  feedback.contact = ''
  feedback.rating = 0
  feedback.timestamp = Date.now()
  feedback.userAgent = navigator.userAgent
  feedback.language = lang.value
}

function goBack() {
  currentStep.value = 'type'
}

// 反馈流程
function selectFeedbackType(type: string) {
  feedback.type = type
  currentStep.value = 'content'
}

function setRating(rating: number) {
  feedback.rating = rating
}

async function submitFeedback() {
  if (!feedback.content.trim()) return
  
  isSubmitting.value = true
  
  try {
    await sendFeedback()
    currentStep.value = 'success'
    
    // 追踪反馈提交
    trackFeedbackSubmission()
    
  } catch (error) {
    currentStep.value = 'error'
    errorMessage.value = isZh.value 
      ? '提交失败，请稍后重试。' 
      : 'Submission failed, please try again later.'
    console.error('Feedback submission error:', error)
  } finally {
    isSubmitting.value = false
  }
}

function retrySubmission() {
  currentStep.value = 'content'
  errorMessage.value = ''
}

async function sendFeedback() {
  // 这里可以集成实际的反馈服务
  // 例如：GitHub Issues API, Notion API, 或自建服务
  
  const feedbackData = {
    ...feedback,
    timestamp: new Date().toISOString(),
    url: typeof window !== 'undefined' ? window.location.href : '',
    referrer: typeof document !== 'undefined' ? document.referrer : ''
  }
  
  // 模拟 API 调用
  await new Promise((resolve, reject) => {
    setTimeout(() => {
      // 90% 成功率模拟
      if (Math.random() > 0.1) {
        resolve(feedbackData)
      } else {
        reject(new Error('Network error'))
      }
    }, 1500)
  })
  
  // 实际实现示例：
  // 1. 发送到 GitHub Issues
  // await createGitHubIssue(feedbackData)
  
  // 2. 发送到 Notion 数据库
  // await sendToNotion(feedbackData)
  
  // 3. 发送到自建 API
  // await fetch('/api/feedback', {
  //   method: 'POST',
  //   headers: { 'Content-Type': 'application/json' },
  //   body: JSON.stringify(feedbackData)
  // })
}

// GitHub Issues 集成示例
async function createGitHubIssue(feedbackData: FeedbackData) {
  const owner = 'your-org'
  const repo = 'your-repo'
  const token = 'your-github-token' // 应该从环境变量获取
  
  const title = `[${feedbackData.type.toUpperCase()}] ${feedbackData.content.substring(0, 50)}...`
  const body = `
## Feedback Type
${feedbackData.type}

## Page
${feedbackData.page}

## Description
${feedbackData.content}

## Additional Information
- **Language**: ${feedbackData.language}
- **User Agent**: ${feedbackData.userAgent}
- **Timestamp**: ${feedbackData.timestamp}
- **Contact**: ${feedbackData.contact || 'Not provided'}
${feedbackData.rating ? `- **Rating**: ${feedbackData.rating}/5` : ''}
`

  const response = await fetch(`https://api.github.com/repos/${owner}/${repo}/issues`, {
    method: 'POST',
    headers: {
      'Authorization': `token ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      title,
      body,
      labels: [`feedback-${feedbackData.type}`, 'user-feedback']
    })
  })
  
  if (!response.ok) {
    throw new Error(`GitHub API error: ${response.status}`)
  }
  
  return response.json()
}

// 工具函数
function getPageInfo(): string {
  return `${route.path} - ${document.title}`
}

function getSelectedTypeTitle(): string {
  const type = feedbackTypes.find(t => t.value === feedback.type)
  return type ? (isZh.value ? type.titleZh : type.title) : ''
}

function getContentPlaceholder(): string {
  const placeholders = {
    bug: {
      en: 'Please describe the bug you encountered, including steps to reproduce...',
      zh: '请描述您遇到的错误，包括重现步骤...'
    },
    feature: {
      en: 'Please describe the feature you would like to see...',
      zh: '请描述您希望看到的功能...'
    },
    content: {
      en: 'Please provide feedback on the content, including suggestions for improvement...',
      zh: '请对内容提供反馈，包括改进建议...'
    },
    satisfaction: {
      en: 'Please share your overall experience and suggestions...',
      zh: '请分享您的整体体验和建议...'
    }
  }
  
  const placeholder = placeholders[feedback.type as keyof typeof placeholders]
  return placeholder ? (isZh.value ? placeholder.zh : placeholder.en) : ''
}

function trackFeedbackSubmission() {
  // Analytics tracking removed for simplicity
  console.log('Feedback submitted:', feedback.type)
}
</script>

<style scoped>
.feedback-widget {
  position: relative;
}

.feedback-trigger {
  position: fixed;
  bottom: 80px;
  right: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--vp-c-brand);
  color: var(--vp-c-white);
  border: none;
  border-radius: 25px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  z-index: 998;
  transition: all 0.3s ease;
}

.feedback-trigger:hover {
  background: var(--vp-c-brand-dark);
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

.trigger-text {
  margin: 0;
}

.feedback-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 999;
}

.feedback-panel {
  position: fixed;
  bottom: 20px;
  right: 20px;
  width: 400px;
  max-height: 600px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.feedback-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--vp-c-bg-soft);
  border-bottom: 1px solid var(--vp-c-divider);
}

.feedback-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--vp-c-text-2);
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.feedback-step {
  padding: 20px;
  flex: 1;
  overflow-y: auto;
}

.step-description {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: var(--vp-c-text-2);
}

.feedback-types {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.feedback-type-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  transition: all 0.2s ease;
}

.feedback-type-btn:hover {
  background: var(--vp-c-bg-soft);
  border-color: var(--vp-c-brand);
}

.type-icon {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  color: var(--vp-c-brand);
}

.type-content {
  flex: 1;
}

.type-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin-bottom: 2px;
}

.type-description {
  font-size: 12px;
  color: var(--vp-c-text-2);
}

.step-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.back-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--vp-c-text-2);
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.back-btn:hover {
  background: var(--vp-c-gray-soft);
  color: var(--vp-c-text-1);
}

.step-header h5 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.feedback-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: var(--vp-c-text-1);
}

.required {
  color: var(--vp-c-red);
}

.page-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.page-input {
  padding: 8px 12px;
  background: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  font-size: 13px;
  color: var(--vp-c-text-2);
  cursor: not-allowed;
}

.feedback-input,
.feedback-textarea {
  padding: 10px 12px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  border-radius: 6px;
  font-size: 14px;
  color: var(--vp-c-text-1);
  transition: border-color 0.2s ease;
  resize: vertical;
}

.feedback-input:focus,
.feedback-textarea:focus {
  outline: none;
  border-color: var(--vp-c-brand);
}

.char-count {
  align-self: flex-end;
  font-size: 11px;
  color: var(--vp-c-text-3);
}

.contact-note {
  font-size: 11px;
  color: var(--vp-c-text-3);
  line-height: 1.4;
}

.rating-stars {
  display: flex;
  gap: 4px;
}

.star-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--vp-c-text-3);
  padding: 4px;
  transition: color 0.2s ease;
}

.star-btn:hover,
.star-btn.active {
  color: #fbbf24;
}

.form-actions {
  margin-top: 8px;
}

.submit-btn {
  width: 100%;
  padding: 12px;
  background: var(--vp-c-brand);
  color: var(--vp-c-white);
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.submit-btn:hover:not(:disabled) {
  background: var(--vp-c-brand-dark);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.success-step,
.error-step {
  text-align: center;
}

.success-icon,
.error-icon {
  margin-bottom: 16px;
}

.success-icon .icon {
  width: 48px;
  height: 48px;
  color: var(--vp-c-green);
}

.error-icon .icon {
  width: 48px;
  height: 48px;
  color: var(--vp-c-red);
}

.success-step h5,
.error-step h5 {
  margin: 0 0 12px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.success-step p,
.error-step p {
  margin: 0 0 20px 0;
  font-size: 14px;
  color: var(--vp-c-text-2);
  line-height: 1.5;
}

.new-feedback-btn {
  padding: 10px 20px;
  background: var(--vp-c-brand);
  color: var(--vp-c-white);
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.new-feedback-btn:hover {
  background: var(--vp-c-brand-dark);
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.retry-btn {
  padding: 8px 16px;
  background: var(--vp-c-brand);
  color: var(--vp-c-white);
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.retry-btn:hover {
  background: var(--vp-c-brand-dark);
}

/* 过渡动画 */
.widget-enter-active,
.widget-leave-active {
  transition: all 0.3s ease;
}

.widget-enter-from,
.widget-leave-to {
  opacity: 0;
  transform: translateY(20px) scale(0.95);
}

.overlay-enter-active,
.overlay-leave-active {
  transition: opacity 0.3s ease;
}

.overlay-enter-from,
.overlay-leave-to {
  opacity: 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .feedback-trigger {
    bottom: 20px;
    right: 20px;
  }
  
  .feedback-panel {
    bottom: 10px;
    right: 10px;
    left: 10px;
    width: auto;
    max-height: calc(100vh - 20px);
  }
}

/* 滚动条样式 */
.feedback-step::-webkit-scrollbar {
  width: 4px;
}

.feedback-step::-webkit-scrollbar-thumb {
  background: var(--vp-c-divider);
  border-radius: 2px;
}

.feedback-step::-webkit-scrollbar-track {
  background: transparent;
}
</style>