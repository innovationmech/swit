# GitHub Pages 首次部署指南

## 前置条件检查

在进行首次部署之前，确保以下条件已满足：

### 1. 仓库权限设置
确保你的GitHub仓库有以下权限：
- 在仓库的 Settings > Actions > General 中，确保 "Workflow permissions" 设置为 "Read and write permissions"
- 确保 "Allow GitHub Actions to create and approve pull requests" 已启用
- **重要**：如果遇到 `Permission denied` 错误，请确保：
  - Workflow 文件中的 `permissions.contents` 设置为 `write`
  - 仓库设置中允许 Actions 写入仓库内容

### 2. GitHub Pages 设置
1. 进入仓库的 `Settings` > `Pages`
2. 在 "Source" 部分选择 `GitHub Actions`
3. 不要选择传统的 "Deploy from a branch" 选项

## 首次部署步骤

### 方法1：通过合并PR触发（推荐）
1. 将当前的 `fix/pages-ci` 分支合并到 `master` 分支
2. 合并后会自动触发 `docs-deploy.yml` workflow
3. 在 Actions 页面监控部署进度

### 方法2：手动触发部署
1. 进入仓库的 `Actions` 页面
2. 选择 "📚 Deploy Documentation" workflow
3. 点击 "Run workflow" 按钮
4. 选择 branch: `master`
5. 设置 environment: `production`
6. 选择 force_sync: `true`（首次部署建议启用）
7. 点击 "Run workflow"

## 监控部署过程

### 在Actions页面检查以下步骤：
1. ✅ `detect-changes` - 检测文档变更
2. ⏩ `sync-content` - 同步内容（可能被跳过）
3. ✅ `build` - 构建文档站点
4. ✅ `deploy` - 部署到GitHub Pages
5. ✅ `performance-test` - 性能测试
6. ✅ `post-deploy-verification` - 部署后验证

### 如果deploy步骤失败：
检查是否是权限问题：
1. 确保仓库的 Pages 设置正确
2. 确保 Workflow permissions 设置正确
3. 检查是否是第一次创建 github-pages environment

## 部署成功后

1. 访问 `https://<username>.github.io/<repository-name>/` 查看站点
2. 在仓库的 Settings > Pages 页面可以看到部署的URL
3. 后续的文档更新会自动触发重新部署

## 常见问题解决

### 问题1：Permission denied 错误
```
remote: Permission to innovationmech/swit.git denied to github-actions[bot].
fatal: unable to access 'https://github.com/innovationmech/swit/': The requested URL returned error: 403
```

**解决方案**：
1. 确保 workflow 文件中设置了正确的权限：
   ```yaml
   permissions:
     contents: write  # 需要写权限
     pages: write
     id-token: write
   ```

2. 检查仓库设置：
   - 进入 Settings > Actions > General
   - 将 "Workflow permissions" 设置为 "Read and write permissions"
   - 启用 "Allow GitHub Actions to create and approve pull requests"

3. 如果问题仍然存在，可能需要：
   - 重新触发 workflow
   - 检查是否有分支保护规则阻止 Actions 推送

### 问题2：deploy步骤被跳过
- 原因：只有在 `master` 分支才会部署
- 解决：确保代码已合并到 `master` 分支

### 问题3：第一次部署失败
- 可能需要手动创建 `github-pages` environment
- 在仓库 Settings > Environments 中创建名为 `github-pages` 的环境

### 问题4：页面显示404
- 检查 `dist/index.html` 是否存在
- 检查 GitHub Pages 设置是否正确
- 等待几分钟让DNS传播

## 验证部署

部署成功后，可以通过以下方式验证：

1. **检查部署状态**：
   ```bash
   curl -I https://<username>.github.io/<repository-name>/
   ```

2. **检查关键页面**：
   - 首页：`/`
   - 英文文档：`/en/`
   - 中文文档：`/zh/`
   - API文档：`/en/api/` 和 `/zh/api/`

3. **性能检查**：
   - Lighthouse CI 会自动运行
   - 检查 Actions 页面的性能测试结果

## 后续维护

- 文档内容更新会自动触发重新部署
- 每天凌晨2点会自动同步内容
- 可以随时通过 workflow_dispatch 手动触发部署
