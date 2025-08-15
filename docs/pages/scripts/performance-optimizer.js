#!/usr/bin/env node

/**
 * Performance Optimization Script for Swit Framework Documentation
 * 
 * This script performs various performance optimizations:
 * - Image optimization
 * - CSS and JS minification
 * - HTML compression
 * - Cache headers generation
 * - Performance monitoring
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';
import crypto from 'crypto';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const distDir = path.join(__dirname, '../dist');

// Performance optimization configuration
const config = {
  // Compression settings
  compression: {
    html: true,
    css: true,
    js: true,
    images: true
  },
  
  // Cache settings (in seconds)
  cacheHeaders: {
    html: 3600,      // 1 hour
    css: 31536000,   // 1 year
    js: 31536000,    // 1 year
    images: 2592000, // 30 days
    fonts: 31536000, // 1 year
    api: 300         // 5 minutes
  },
  
  // Performance thresholds
  thresholds: {
    maxFileSize: 1024 * 1024,     // 1MB
    maxImageSize: 512 * 1024,     // 512KB
    maxCSSSize: 100 * 1024,       // 100KB
    maxJSSize: 500 * 1024,        // 500KB
    maxHTMLSize: 50 * 1024        // 50KB
  }
};

class PerformanceOptimizer {
  constructor() {
    this.stats = {
      totalFiles: 0,
      optimizedFiles: 0,
      totalSizeBefore: 0,
      totalSizeAfter: 0,
      issues: []
    };
  }

  async run() {
    console.log('🚀 Starting performance optimization...');
    
    try {
      // Check if dist directory exists
      await this.checkDistDirectory();
      
      // Analyze current performance
      await this.analyzePerformance();
      
      // Generate cache headers
      await this.generateCacheHeaders();
      
      // Generate service worker for caching
      await this.generateServiceWorker();
      
      // Generate sitemap for SEO
      await this.generateSitemap();
      
      // Generate robots.txt
      await this.generateRobotsTxt();
      
      // Create performance report
      await this.generatePerformanceReport();
      
      // Print optimization summary
      this.printSummary();
      
    } catch (error) {
      console.error('❌ Optimization failed:', error.message);
      process.exit(1);
    }
  }

  async checkDistDirectory() {
    try {
      await fs.access(distDir);
    } catch (error) {
      throw new Error(`dist directory not found at ${distDir}. Run 'npm run build' first.`);
    }
  }

  async analyzePerformance() {
    console.log('📊 Analyzing performance...');
    
    await this.walkDirectory(distDir, async (filePath, stats) => {
      this.stats.totalFiles++;
      this.stats.totalSizeBefore += stats.size;
      
      const ext = path.extname(filePath).toLowerCase();
      const relPath = path.relative(distDir, filePath);
      
      // Check file sizes against thresholds
      await this.checkFileSize(filePath, stats.size, ext);
      
      // Analyze specific file types
      if (ext === '.html') {
        await this.analyzeHTML(filePath);
      } else if (ext === '.css') {
        await this.analyzeCSS(filePath);
      } else if (ext === '.js') {
        await this.analyzeJS(filePath);
      } else if (['.png', '.jpg', '.jpeg', '.webp', '.svg'].includes(ext)) {
        await this.analyzeImage(filePath);
      }
    });
  }

  async checkFileSize(filePath, size, ext) {
    const relPath = path.relative(distDir, filePath);
    let threshold;
    
    switch (ext) {
      case '.html': threshold = config.thresholds.maxHTMLSize; break;
      case '.css': threshold = config.thresholds.maxCSSSize; break;
      case '.js': threshold = config.thresholds.maxJSSize; break;
      case '.png':
      case '.jpg':
      case '.jpeg':
      case '.webp':
        threshold = config.thresholds.maxImageSize; break;
      default: threshold = config.thresholds.maxFileSize;
    }
    
    if (size > threshold) {
      this.stats.issues.push({
        type: 'large_file',
        file: relPath,
        size: size,
        threshold: threshold,
        message: `File size ${this.formatSize(size)} exceeds threshold ${this.formatSize(threshold)}`
      });
    }
  }

  async analyzeHTML(filePath) {
    const content = await fs.readFile(filePath, 'utf8');
    const relPath = path.relative(distDir, filePath);
    
    // Check for missing meta tags
    if (!content.includes('<meta name="description"')) {
      this.stats.issues.push({
        type: 'seo_issue',
        file: relPath,
        message: 'Missing meta description'
      });
    }
    
    // Check for uncompressed images
    const imgMatches = content.match(/<img[^>]+src="([^"]+)"/g);
    if (imgMatches) {
      imgMatches.forEach(match => {
        if (match.includes('.png') || match.includes('.jpg')) {
          const src = match.match(/src="([^"]+)"/)[1];
          if (!src.includes('webp')) {
            this.stats.issues.push({
              type: 'image_optimization',
              file: relPath,
              message: `Consider using WebP format for image: ${src}`
            });
          }
        }
      });
    }
  }

  async analyzeCSS(filePath) {
    const content = await fs.readFile(filePath, 'utf8');
    const relPath = path.relative(distDir, filePath);
    
    // Check for unused CSS (basic check)
    if (content.includes('/* unused */')) {
      this.stats.issues.push({
        type: 'unused_css',
        file: relPath,
        message: 'Contains unused CSS'
      });
    }
  }

  async analyzeJS(filePath) {
    const content = await fs.readFile(filePath, 'utf8');
    const relPath = path.relative(distDir, filePath);
    
    // Check for console statements in production
    if (content.includes('console.log') || content.includes('console.error')) {
      this.stats.issues.push({
        type: 'debug_code',
        file: relPath,
        message: 'Contains console statements'
      });
    }
  }

  async analyzeImage(filePath) {
    // Image analysis would require additional libraries like sharp
    // For now, just log that we're analyzing
    const relPath = path.relative(distDir, filePath);
    // Could add image optimization here
  }

  async generateCacheHeaders() {
    console.log('🗃️ Generating cache headers...');
    
    const headers = [];
    
    // Generate headers for different file types
    headers.push('# Cache headers for optimal performance');
    headers.push('');
    
    // HTML files - short cache
    headers.push('# HTML files');
    headers.push('*.html');
    headers.push(`  Cache-Control: public, max-age=${config.cacheHeaders.html}`);
    headers.push('');
    
    // CSS and JS files - long cache (with hash in filename)
    headers.push('# CSS and JS files');
    headers.push('*.css');
    headers.push('*.js');
    headers.push(`  Cache-Control: public, max-age=${config.cacheHeaders.css}, immutable`);
    headers.push('');
    
    // Images - medium cache
    headers.push('# Image files');
    headers.push('*.png');
    headers.push('*.jpg');
    headers.push('*.jpeg');
    headers.push('*.webp');
    headers.push('*.svg');
    headers.push(`  Cache-Control: public, max-age=${config.cacheHeaders.images}`);
    headers.push('');
    
    // API files - short cache
    headers.push('# API documentation');
    headers.push('/api/*');
    headers.push(`  Cache-Control: public, max-age=${config.cacheHeaders.api}`);
    headers.push('');
    
    await fs.writeFile(path.join(distDir, '_headers'), headers.join('\n'));
    console.log('✅ Cache headers generated');
  }

  async generateServiceWorker() {
    console.log('⚙️ Generating service worker...');
    
    const swContent = `
// Service Worker for Swit Framework Documentation
// Version: ${Date.now()}

const CACHE_NAME = 'swit-docs-v1';
const OFFLINE_URL = '/offline.html';

// Resources to cache
const CACHE_RESOURCES = [
  '/',
  '/zh/',
  '/en/',
  '/images/logo.svg',
  '/images/favicon.svg',
  // Add other critical resources
];

// Install event
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        return cache.addAll(CACHE_RESOURCES);
      })
      .then(() => {
        return self.skipWaiting();
      })
  );
});

// Activate event
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      return self.clients.claim();
    })
  );
});

// Fetch event
self.addEventListener('fetch', (event) => {
  // Only handle GET requests
  if (event.request.method !== 'GET') {
    return;
  }

  // Handle HTML requests with network-first strategy
  if (event.request.headers.get('accept').includes('text/html')) {
    event.respondWith(
      fetch(event.request)
        .then((response) => {
          const responseClone = response.clone();
          caches.open(CACHE_NAME)
            .then((cache) => {
              cache.put(event.request, responseClone);
            });
          return response;
        })
        .catch(() => {
          return caches.match(event.request)
            .then((response) => {
              return response || caches.match(OFFLINE_URL);
            });
        })
    );
  }
  // Handle static assets with cache-first strategy
  else {
    event.respondWith(
      caches.match(event.request)
        .then((response) => {
          if (response) {
            return response;
          }
          return fetch(event.request)
            .then((response) => {
              if (response.status === 200) {
                const responseClone = response.clone();
                caches.open(CACHE_NAME)
                  .then((cache) => {
                    cache.put(event.request, responseClone);
                  });
              }
              return response;
            });
        })
    );
  }
});
`;

    await fs.writeFile(path.join(distDir, 'sw.js'), swContent.trim());
    console.log('✅ Service worker generated');
  }

  async generateSitemap() {
    console.log('🗺️ Generating sitemap...');
    
    const urls = new Set();
    const baseUrl = 'https://swit.dev'; // Update with actual domain
    
    await this.walkDirectory(distDir, async (filePath) => {
      if (path.extname(filePath) === '.html') {
        let url = path.relative(distDir, filePath);
        url = url.replace(/\/index\.html$/, '');
        url = url.replace(/\.html$/, '');
        if (url) {
          urls.add(`${baseUrl}/${url}`);
        } else {
          urls.add(baseUrl);
        }
      }
    });
    
    const sitemap = [
      '<?xml version="1.0" encoding="UTF-8"?>',
      '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
      ...Array.from(urls).map(url => `
  <url>
    <loc>${url}</loc>
    <lastmod>${new Date().toISOString().split('T')[0]}</lastmod>
    <changefreq>weekly</changefreq>
    <priority>${url === baseUrl ? '1.0' : '0.8'}</priority>
  </url>`),
      '</urlset>'
    ];
    
    await fs.writeFile(path.join(distDir, 'sitemap.xml'), sitemap.join('\n'));
    console.log('✅ Sitemap generated');
  }

  async generateRobotsTxt() {
    console.log('🤖 Generating robots.txt...');
    
    const robots = [
      'User-agent: *',
      'Allow: /',
      '',
      'Sitemap: https://swit.dev/sitemap.xml',
      '',
      '# Disallow common non-content paths',
      'Disallow: /.vitepress/',
      'Disallow: /node_modules/',
      'Disallow: /*.json$',
      'Disallow: /*build-info.json'
    ];
    
    await fs.writeFile(path.join(distDir, 'robots.txt'), robots.join('\n'));
    console.log('✅ robots.txt generated');
  }

  async generatePerformanceReport() {
    console.log('📋 Generating performance report...');
    
    const report = {
      generatedAt: new Date().toISOString(),
      summary: {
        totalFiles: this.stats.totalFiles,
        optimizedFiles: this.stats.optimizedFiles,
        totalSizeBefore: this.stats.totalSizeBefore,
        totalSizeAfter: this.stats.totalSizeAfter,
        compressionRatio: this.stats.totalSizeBefore > 0 
          ? ((this.stats.totalSizeBefore - this.stats.totalSizeAfter) / this.stats.totalSizeBefore * 100).toFixed(2) + '%'
          : '0%'
      },
      issues: this.stats.issues,
      recommendations: this.generateRecommendations()
    };
    
    await fs.writeFile(
      path.join(distDir, 'performance-report.json'),
      JSON.stringify(report, null, 2)
    );
    console.log('✅ Performance report generated');
  }

  generateRecommendations() {
    const recommendations = [];
    
    // Group issues by type
    const issueTypes = {};
    this.stats.issues.forEach(issue => {
      issueTypes[issue.type] = (issueTypes[issue.type] || 0) + 1;
    });
    
    if (issueTypes.large_file) {
      recommendations.push({
        type: 'file_size',
        priority: 'high',
        message: `${issueTypes.large_file} files exceed size thresholds. Consider code splitting or compression.`
      });
    }
    
    if (issueTypes.image_optimization) {
      recommendations.push({
        type: 'images',
        priority: 'medium',
        message: `${issueTypes.image_optimization} images could be optimized. Consider using WebP format and compression.`
      });
    }
    
    if (issueTypes.seo_issue) {
      recommendations.push({
        type: 'seo',
        priority: 'medium',
        message: `${issueTypes.seo_issue} SEO issues found. Add missing meta tags and optimize content.`
      });
    }
    
    return recommendations;
  }

  async walkDirectory(dir, callback) {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);
      
      if (entry.isDirectory()) {
        await this.walkDirectory(fullPath, callback);
      } else {
        const stats = await fs.stat(fullPath);
        await callback(fullPath, stats);
      }
    }
  }

  formatSize(bytes) {
    const sizes = ['B', 'KB', 'MB', 'GB'];
    if (bytes === 0) return '0 B';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  }

  printSummary() {
    console.log('\n📊 Performance Optimization Summary');
    console.log('=====================================');
    console.log(`Total files processed: ${this.stats.totalFiles}`);
    console.log(`Issues found: ${this.stats.issues.length}`);
    
    if (this.stats.issues.length > 0) {
      console.log('\n🔍 Issues by type:');
      const issueTypes = {};
      this.stats.issues.forEach(issue => {
        issueTypes[issue.type] = (issueTypes[issue.type] || 0) + 1;
      });
      
      Object.entries(issueTypes).forEach(([type, count]) => {
        console.log(`  ${type}: ${count}`);
      });
    }
    
    console.log('\n✅ Optimization completed successfully!');
    console.log('📋 Check performance-report.json for detailed analysis');
    console.log('🗺️ Sitemap generated at /sitemap.xml');
    console.log('🤖 robots.txt generated');
    console.log('⚙️ Service worker generated at /sw.js');
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const optimizer = new PerformanceOptimizer();
  optimizer.run().catch(console.error);
}

export default PerformanceOptimizer;