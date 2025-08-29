---
layout: home
title: Swit Framework
description: Modern Go Microservice Development Framework
titleTemplate: Production-ready microservice foundation
# Last updated: 2025-08-24
head:
  - - script
    - type: text/javascript
    - |
      // Browser language detection and auto-redirect
      (function() {
        // Only run on the root page
        if (window.location.pathname !== '/' && !window.location.pathname.startsWith('/swit/')) return;
        
        // Don't redirect if user has manually selected a language
        if (localStorage.getItem('vitepress-preferred-locale')) return;
        
        // Detect user's preferred language
        const userLang = navigator.language || navigator.userLanguage;
        const langCode = userLang.toLowerCase();
        
        let targetLang = 'en'; // default to English
        
        // Check if user prefers Chinese
        if (langCode.startsWith('zh')) {
          targetLang = 'zh';
        }
        
        // Get current path and redirect to appropriate language version
        const currentPath = window.location.pathname;
        const basePath = '/swit/';
        
        // If we're on the root page, redirect to language-specific page
        if (currentPath === '/' || currentPath === basePath) {
          window.location.href = basePath + targetLang + '/';
        }
      })();

hero:
  name: "Swit"
  text: "Go Microservice Framework"
  tagline: Production-ready microservice development foundation
  image:
    src: /images/logo.svg
    alt: Swit Framework
  actions:
    - theme: brand
      text: Get Started
      link: /en/guide/getting-started
    - theme: alt
      text: 查看中文文档
      link: /zh/guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/innovationmech/swit

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

Choose your preferred language to get started with Swit framework:

<div style="display: flex; gap: 1rem; margin: 2rem 0;">
  <a href="/en/guide/getting-started" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>🇺🇸 English Documentation</h3>
    <p>Complete guide and API reference in English</p>
  </a>
  <a href="/zh/guide/getting-started" style="flex: 1; padding: 1rem; border: 1px solid var(--vp-c-border); border-radius: 8px; text-decoration: none;">
    <h3>🇨🇳 中文文档</h3>
    <p>完整的中文指南和 API 参考</p>
  </a>
</div>

## Project Statistics

<div class="stats-grid">
  <div class="stat-card">
    <div class="stat-number">MIT</div>
    <div class="stat-label">License</div>
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
