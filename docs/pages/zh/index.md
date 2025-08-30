---
layout: home
title: Swit Go 微服务框架
titleTemplate: Go 微服务开发框架

hero:
  name: "Swit"
  text: "Go 微服务框架"
  tagline: 生产就绪的微服务开发基础设施
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 查看 API
      link: /zh/api/
    - theme: alt
      text: English Docs
      link: /en/guide/getting-started

features:
  - icon: 🚀
    title: 统一的服务器框架
    details: 完整的服务器生命周期管理，包括传输协调和健康监控
  - icon: 🔄
    title: 多传输层支持
    details: 无缝的 HTTP 和 gRPC 传输协调，支持可插拔架构
  - icon: 🔍
    title: 错误监控与可观测性
    details: 全面的 Sentry 集成，支持错误跟踪、性能监控和实时告警
  - icon: 🛠️
    title: CLI 开发工具
    details: 强大的 switctl CLI 工具，支持脚手架、代码生成、质量检查和模板管理
  - icon: 📦
    title: 依赖注入系统
    details: 基于工厂的依赖容器，支持自动生命周期管理
  - icon: 🔒
    title: 生产级安全
    details: 内置安全最佳实践、漏洞扫描和安全开发工作流
---

## 快速开始

开始使用 Swit 框架构建您的微服务：

<div style="display: flex; gap: 1rem; margin: 2rem 0;">
  <a href="/zh/guide/getting-started" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>📖 开发指南</h3>
    <p>完整的中文开发指南和教程</p>
  </a>
  <a href="/zh/examples/" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>💡 代码示例</h3>
    <p>实际项目示例和最佳实践</p>
  </a>
</div>

## 项目信息

<div class="stats-grid">
  <div class="stat-card">
    <div class="stat-number">MIT</div>
    <div class="stat-label">开源许可证</div>
  </div>
  <div class="stat-card">
    <div class="stat-number">Go 1.23.12+</div>
    <div class="stat-label">运行要求</div>
  </div>
  <div class="stat-card">
    <div class="stat-number">生产级</div>
    <div class="stat-label">稳定性</div>
  </div>
</div>

<style>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 1rem;
  margin: 2rem 0;
}

.stat-card {
  text-align: center;
  padding: 1rem;
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  background: var(--vp-c-bg-soft);
}

.stat-number {
  font-size: 1.5rem;
  font-weight: bold;
  color: var(--vp-c-brand-1);
}

.stat-label {
  font-size: 0.9rem;
  color: var(--vp-c-text-2);
  margin-top: 0.5rem;
}
</style>
