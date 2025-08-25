---
layout: home
title: Swit Go Microservice Framework
titleTemplate: Go Microservice Development Framework

hero:
  name: "Swit"
  text: "Go Microservice Framework"
  tagline: Production-ready microservice development foundation
  actions:
    - theme: brand
      text: Get Started
      link: /en/guide/getting-started
    - theme: alt
      text: View API
      link: /en/api/
    - theme: alt
      text: 中文文档
      link: /zh/guide/getting-started

features:
  - icon: 🚀
    title: Unified Server Framework
    details: Complete server lifecycle management with transport coordination and health monitoring
  - icon: 🔄
    title: Multi-Transport Support
    details: Seamless HTTP and gRPC transport coordination with pluggable architecture
  - icon: 🔍
    title: Error Monitoring & Observability
    details: Comprehensive Sentry integration for error tracking, performance monitoring, and real-time alerts
  - icon: 🛠️
    title: CLI Development Tools
    details: Powerful switctl CLI for scaffolding, code generation, quality checks, and template management
  - icon: 📦
    title: Dependency Injection
    details: Factory-based dependency container with automatic lifecycle management
  - icon: 🔒
    title: Production Security
    details: Built-in security best practices, vulnerability scanning, and secure development workflows
---

## Quick Start

Get started building microservices with Swit framework:

<div style="display: flex; gap: 1rem; margin: 2rem 0;">
  <a href="/en/guide/getting-started" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>📖 Developer Guide</h3>
    <p>Complete development guide and tutorials</p>
  </a>
  <a href="/en/examples/" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>💡 Code Examples</h3>
    <p>Real project examples and best practices</p>
  </a>
</div>

## Project Information

<div class="stats-grid">
  <div class="stat-card">
    <div class="stat-number">MIT</div>
    <div class="stat-label">Open Source</div>
  </div>
  <div class="stat-card">
    <div class="stat-number">Go 1.24+</div>
    <div class="stat-label">Requirements</div>
  </div>
  <div class="stat-card">
    <div class="stat-number">Production</div>
    <div class="stat-label">Ready</div>
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
