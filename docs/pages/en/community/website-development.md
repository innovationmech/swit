---
title: Website Development Guide
description: Comprehensive guide for developing and maintaining the Swit project website
---

# Website Development Guide

This document provides comprehensive guidance for developing and maintaining the Swit project's GitHub Pages website built with VitePress.

## Overview

The Swit project website is built using:
- **VitePress** - Static site generator
- **Vue 3** - Component framework
- **TypeScript** - Type safety
- **Tailwind CSS** - Styling framework
- **GitHub Actions** - CI/CD pipeline

## Project Structure

```
docs/pages/
├── .vitepress/           # VitePress configuration
│   ├── config/          # Multi-language configurations
│   └── theme/           # Custom theme components
├── en/                  # English content
├── zh/                  # Chinese content
├── public/              # Static assets
├── scripts/             # Build and automation scripts
└── tests/               # Testing suite
```

## Development Setup

### Prerequisites

- Node.js 18+
- npm or yarn
- Git

### Installation

```bash
cd docs/pages
npm install
```

### Development Commands

```bash
# Start development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Run tests
npm test

# Run linting
npm run lint

# Type checking
npm run type-check
```

## Website Architecture

### VitePress Configuration

The website uses a multi-language setup with shared configurations:

- `config/index.mjs` - Main configuration entry
- `config/shared.mjs` - Shared settings
- `config/en.mjs` - English-specific configuration
- `config/zh.mjs` - Chinese-specific configuration

### Theme Customization

Custom theme components are located in `.vitepress/theme/`:

- `index.js` - Theme entry point
- `components/` - Vue components
- `styles/` - CSS styles and variables

### Content Structure

Content is organized by language and category:

```
├── en/zh/
│   ├── index.md          # Homepage
│   ├── guide/            # User guides
│   ├── api/              # API documentation
│   ├── examples/         # Code examples
│   └── community/        # Community resources
```

## Component Development

### Creating New Components

1. Create Vue component in `.vitepress/theme/components/`
2. Import and register in theme `index.js`
3. Use in Markdown files with proper props

Example component structure:

```vue
<template>
  <div class="my-component">
    <!-- Component content -->
  </div>
</template>

<script setup lang="ts">
// TypeScript setup
</script>

<style scoped>
/* Component styles */
</style>
```

### Key Components

- **HomePage.vue** - Landing page hero and features
- **FeatureCard.vue** - Feature showcase cards
- **CodeExample.vue** - Syntax-highlighted code examples
- **ApiDocViewer.vue** - API documentation display
- **SearchBox.vue** - Enhanced search functionality

### Styling Guidelines

- Use Tailwind CSS classes for consistency
- Follow responsive design principles
- Support both light and dark themes
- Maintain accessibility standards

## Content Management

### Writing Documentation

1. Use Markdown with VitePress extensions
2. Include frontmatter with title and description
3. Follow established content structure
4. Add appropriate navigation links

### Multi-language Support

- Keep content structure consistent between languages
- Use the same file names and paths
- Update both language versions simultaneously
- Test language switching functionality

### Content Automation

The website includes scripts for automated content management:

- `sync-docs.js` - Synchronizes content from main project
- `api-docs-generator.js` - Generates API documentation
- `content-validator.js` - Validates content consistency

## API Documentation Integration

### Automatic Generation

API documentation is automatically generated from:
- OpenAPI/Swagger specifications
- Protocol Buffer definitions
- Code comments and annotations

### Manual Updates

For manual API documentation updates:

1. Edit source files in the main project
2. Run `npm run sync:api` to regenerate
3. Review and commit changes

## Performance Optimization

### Build Optimization

- Code splitting is configured automatically
- Static assets are optimized during build
- CSS is purged and minimized

### Runtime Performance

- Images use lazy loading
- Critical resources are preloaded
- Service Worker provides offline caching

### Monitoring

Performance is monitored using:
- Lighthouse CI in GitHub Actions
- Core Web Vitals tracking
- Bundle size analysis

## Testing Strategy

### Unit Tests

```bash
# Run component tests
npm run test:unit

# Watch mode
npm run test:unit:watch

# Coverage report
npm run test:coverage
```

### E2E Tests

```bash
# Run end-to-end tests
npm run test:e2e

# Run in headless mode
npm run test:e2e:headless
```

### Accessibility Tests

```bash
# Run accessibility audits
npm run test:a11y
```

### Performance Tests

```bash
# Run performance audits
npm run test:performance
```

## Deployment

### GitHub Actions

The website deploys automatically via GitHub Actions:

1. **Build Stage** - Compiles VitePress site
2. **Test Stage** - Runs all test suites
3. **Deploy Stage** - Publishes to GitHub Pages

### Manual Deployment

For manual deployment:

```bash
# Build production site
npm run build

# Deploy to GitHub Pages
npm run deploy
```

## Maintenance Tasks

### Regular Updates

- Update dependencies monthly
- Review and update content quarterly
- Monitor performance metrics weekly
- Check broken links monthly

### Content Synchronization

```bash
# Sync with main project documentation
npm run sync:docs

# Update API documentation
npm run sync:api

# Validate content consistency
npm run validate:content
```

### Security Updates

- Monitor security advisories
- Update dependencies promptly
- Run security audits regularly

## Troubleshooting

### Common Issues

**Build Failures**
- Check Node.js version compatibility
- Clear node_modules and reinstall
- Verify configuration files syntax

**Development Server Issues**
- Check port availability (default 5173)
- Clear VitePress cache
- Verify file permissions

**Content Display Issues**
- Check Markdown syntax
- Verify frontmatter format
- Test component imports

**Performance Issues**
- Analyze bundle size
- Check image optimization
- Review lazy loading implementation

### Debug Mode

Enable debug mode for detailed logging:

```bash
DEBUG=vitepress:* npm run dev
```

## Contributing Guidelines

### Code Style

- Follow TypeScript best practices
- Use ESLint and Prettier configurations
- Write meaningful commit messages
- Include tests for new features

### Pull Request Process

1. Create feature branch
2. Make changes with tests
3. Run full test suite
4. Submit PR with description
5. Address review feedback

### Documentation Standards

- Write clear, concise documentation
- Include code examples
- Keep both language versions updated
- Follow established patterns

## Version Control

### Branch Strategy

- `master` - Main branch
- `docs/development` - Active development
- `docs/gh-pages` - Deployment branch
- Feature branches for specific changes

### Release Process

1. Update version numbers
2. Run full test suite
3. Build production site
4. Tag release
5. Deploy to production

## Support and Resources

### Getting Help

- Check existing documentation
- Search GitHub issues
- Create detailed bug reports
- Join community discussions

### Useful Resources

- [VitePress Documentation](https://vitepress.dev/)
- [Vue 3 Guide](https://vuejs.org/)
- [Tailwind CSS Docs](https://tailwindcss.com/)
- [GitHub Actions Guide](https://docs.github.com/en/actions)

### Contact

- Create GitHub issues for bugs
- Use discussions for questions
- Follow contribution guidelines
- Respect community standards