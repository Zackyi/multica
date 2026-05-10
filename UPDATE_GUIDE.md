# Multica 更新操作手册

本文档供 Claude AI 执行使用，用于在更新 multica 上游代码时，自动保留本地部署所需的修改。

## 执行环境信息

- **项目路径**：`/Users/fh/fhdev/multica`
- **Tailscale 地址**：`fhs-mac-mini.tail626c72.ts.net`
- **Tailscale IP**：`100.94.134.78`
- **Docker Compose 文件**：`docker-compose.selfhost.yml`

---

## 第一步：检查当前状态

```bash
cd /Users/fh/fhdev/multica
git status
git branch -a
```

**预期输出**：
- 当前分支应该是 `main` 或 `local-deploy`
- 工作目录应该是干净的（或显示已知的本地修改）

**如果当前分支不是 main**：
```bash
git checkout main
```

---

## 第二步：备份当前状态

```bash
# 创建备份分支
git branch backup-before-update-$(date +%Y%m%d)

# 提交当前所有修改
git add -A
git commit -m "backup before upstream update - $(date +%Y%m%d)"
```

---

## 第三步：获取上游更新

```bash
# 确认上游远程仓库
git remote -v | grep upstream

# 如果没有 upstream，添加官方仓库
git remote add upstream https://github.com/multica-ai/multica.git

# 获取最新代码
git fetch upstream main
```

---

## 第四步：检查待合并的变更

```bash
# 查看本地与上游的差异
git log HEAD..upstream/main --oneline | head -20

# 查看影响的文件
git diff --name-only HEAD upstream/main
```

**重点关注以下文件是否会被修改**：
- `server/cmd/server/router.go`
- `packages/core/platform/auth-initializer.tsx`
- `apps/web/next.config.ts`
- `Dockerfile.web`
- `docker-compose.selfhost.yml`
- `server/internal/middleware/`

---

## 第五步：合并上游代码

```bash
# 合并上游 main 分支
git merge upstream/main -m "Merge upstream/main - $(date +%Y%m%d)"
```

**如果有冲突**，按以下优先级处理：

### 冲突处理规则

| 优先级 | 规则 | 说明 |
|--------|------|------|
| 1 | **保留本地修改** | 对于以下关键文件的本地修改必须保留 |
| 2 | **接受上游变更** | 对于未修改的文件，接受上游的新代码 |
| 3 | **手动合并** | 对于既有本地修改又有上游变更的文件，手动合并 |

---

## 第六步：验证和恢复本地修改

### 6.1 检查关键文件是否存在

#### 检查 local_auth.go
```bash
ls -la server/internal/middleware/local_auth.go
```

**如果文件不存在**，需要重新创建。这是新增文件，上游不会有，但如果被意外删除需恢复：

```bash
# 从备份恢复
git show backup-before-update-*:server/internal/middleware/local_auth.go > server/internal/middleware/local_auth.go
```

#### 检查 router.go
```bash
grep -n "LocalAuth\|LocalDaemonAuth\|LocalModeInfo" server/cmd/server/router.go
```

**如果输出为空**，说明上游覆盖了本地修改，需要重新应用。

#### 检查 auth-initializer.tsx
```bash
grep -n "fetch(\"/health\")\|LOCAL_MODE_USER" packages/core/platform/auth-initializer.tsx
```

**如果输出为空**，说明上游覆盖了本地修改，需要重新应用。

#### 检查 next.config.ts
```bash
grep -n 'source: "/health"' apps/web/next.config.ts
```

**如果输出为空**，说明上游覆盖了本地修改，需要重新应用。

#### 检查 Dockerfile.web
```bash
grep -n "NEXT_PUBLIC_API_URL\|NEXT_PUBLIC_WS_URL" Dockerfile.web
```

**如果输出为空**，说明上游覆盖了本地修改，需要重新应用。

#### 检查 docker-compose.selfhost.yml
```bash
grep -A2 "build:" docker-compose.selfhost.yml | grep NEXT_PUBLIC
```

**如果输出为空**，说明上游覆盖了本地修改，需要重新应用。

---

## 第七步：恢复本地修改（如需要）

如果第六步发现任何关键文件的本地修改被覆盖，执行以下恢复操作：

### 7.1 恢复 router.go

检查 `server/cmd/server/router.go` 中的以下关键修改：

**必须存在的修改点 1**：使用 LocalAuth 中间件
```go
// 查找这一行，应该是：
r.Use(middleware.LocalAuth(queries))
// 而不是：
// r.Use(middleware.Auth(queries))
```

**必须存在的修改点 2**：使用 LocalDaemonAuth 中间件
```go
// /api/daemon 路由应该使用：
r.Use(middleware.LocalDaemonAuth(queries))
// 而不是：
// r.Use(middleware.DaemonAuth(queries))
```

**必须存在的修改点 3**：健康检查返回本地模式信息
```go
// /health 端点应该包含：
info := middleware.LocalModeInfo()
info["status"] = "ok"
```

**如果以上修改不存在**，从备份恢复或手动添加：

```bash
# 查看备份中的正确版本
git show backup-before-update-*:server/cmd/server/router.go > /tmp/router_backup.go
# 对比并手动合并
diff /tmp/router_backup.go server/cmd/server/router.go
```

### 7.2 恢复 auth-initializer.tsx

从备份恢复完整的本地版本：

```bash
git show backup-before-update-*:packages/core/platform/auth-initializer.tsx > packages/core/platform/auth-initializer.tsx
```

### 7.3 恢复 next.config.ts

检查 `/health` rewrite 规则是否存在：

```typescript
async rewrites() {
  return [
    {
      source: "/health",
      destination: `${remoteApiUrl}/health`,
    },
    // ... 其他规则
  ];
}
```

**如果不存在**，从备份恢复或手动添加：

```bash
git show backup-before-update-*:apps/web/next.config.ts > /tmp/next_config_backup.ts
# 查看差异
diff /tmp/next_config_backup.ts apps/web/next.config.ts
```

### 7.4 恢复 Dockerfile.web

检查构建参数是否存在：

```dockerfile
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ARG NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws

ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
```

**如果不存在**，从备份恢复或手动添加：

```bash
git show backup-before-update-*:Dockerfile.web > /tmp/dockerfile_backup.web
# 查看差异
diff /tmp/dockerfile_backup.web Dockerfile.web
```

### 7.5 恢复 docker-compose.selfhost.yml

检查构建参数是否存在：

```yaml
frontend:
  build:
    args:
      NEXT_PUBLIC_API_URL: ${NEXT_PUBLIC_API_URL:-http://localhost:8080}
      NEXT_PUBLIC_WS_URL: ${NEXT_PUBLIC_WS_URL:-ws://localhost:8080/ws}
```

**如果不存在**，从备份恢复或手动添加：

```bash
git show backup-before-update-*:docker-compose.selfhost.yml > /tmp/compose_backup.yml
# 查看差异
diff /tmp/compose_backup.yml docker-compose.selfhost.yml
```

---

## 第八步：验证环境变量配置

检查 `.env` 文件是否包含必要的配置：

```bash
grep -E "MULTICA_LOCAL_MODE|NEXT_PUBLIC_API_URL|NEXT_PUBLIC_WS_URL|FRONTEND_ORIGIN|CORS_ALLOWED_ORIGINS" .env
```

**必须存在的配置**：
```bash
MULTICA_LOCAL_MODE=true
NEXT_PUBLIC_API_URL=http://fhs-mac-mini.tail626c72.ts.net:8080
NEXT_PUBLIC_WS_URL=ws://fhs-mac-mini.tail626c72.ts.net:8080/ws
FRONTEND_ORIGIN=http://fhs-mac-mini.tail626c72.ts.net:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://fhs-mac-mini.tail626c72.ts.net:3000,http://100.94.134.78:3000
```

**如果配置不存在或被重置**，重新添加：

```bash
cat >> .env << 'EOF'

# Local Mode - skip authentication
MULTICA_LOCAL_MODE=true

# Tailscale access
NEXT_PUBLIC_API_URL=http://fhs-mac-mini.tail626c72.ts.net:8080
NEXT_PUBLIC_WS_URL=ws://fhs-mac-mini.tail626c72.ts.net:8080/ws
FRONTEND_ORIGIN=http://fhs-mac-mini.tail626c72.ts.net:3000

# CORS for Tailscale
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://fhs-mac-mini.tail626c72.ts.net:3000,http://100.94.134.78:3000
EOF
```

---

## 第九步：停止并清理旧容器

```bash
cd /Users/fh/fhdev/multica
docker compose -f docker-compose.selfhost.yml down
```

**预期输出**：
```
Container multica-postgres-1  Stopped
Container multica-backend-1    Stopped
Container multica-frontend-1   Stopped
```

---

## 第十步：重新构建

```bash
# 清理构建缓存（确保使用新代码）
docker builder prune -f

# 重新构建所有服务
docker compose -f docker-compose.selfhost.yml build --no-cache
```

**构建时间约 5-10 分钟**，请等待完成。

---

## 第十一步：启动服务

```bash
docker compose -f docker-compose.selfhost.yml up -d
```

**预期输出**：
```
Container multica-postgres-1  Started
Container multica-backend-1    Started
Container multica-frontend-1   Started
```

**等待服务启动**（约 10 秒）：
```bash
sleep 10
```

---

## 第十二步：验证服务状态

### 12.1 检查容器状态

```bash
docker compose -f docker-compose.selfhost.yml ps
```

**预期输出**（所有服务 Status 应为 Up）：
```
NAME                 STATUS
multica-backend-1    Up X minutes
multica-frontend-1   Up X minutes
multica-postgres-1   Up X minutes (healthy)
```

### 12.2 检查后端健康状态

```bash
curl -s http://localhost:8080/health | jq
```

**预期输出**：
```json
{
  "default_user": "local@multica.local",
  "local_mode": "enabled",
  "status": "ok"
}
```

### 12.3 检查前端状态

```bash
curl -s http://localhost:3000/health | jq
```

**预期输出**：
```json
{
  "default_user": "local@multica.local",
  "local_mode": "enabled",
  "status": "ok"
}
```

### 12.4 检查工作空间 API

```bash
curl -s http://localhost:8080/api/workspaces | jq
```

**预期输出**：
```json
[
  {
    "id": "...",
    "name": "Local Workspace",
    "slug": "local-workspace"
  }
]
```

### 12.5 检查 Tailscale 访问

```bash
curl -s http://fhs-mac-mini.tail626c72.ts.net:3000/health | jq
```

**预期输出**：
```json
{
  "default_user": "local@multica.local",
  "local_mode": "enabled",
  "status": "ok"
}
```

---

## 第十三步：浏览器验证

### Mac 本地验证

1. 打开 Safari
2. 访问 `http://localhost:3000`
3. **预期**：直接进入 dashboard，显示 "Dashboard" 按钮，无登录页面

### iPhone Tailscale 验证

1. 确认 iPhone Tailscale 状态为绿灯
2. 打开 iPhone Safari
3. 访问 `http://fhs-mac-mini.tail626c72.ts.net:3000`
4. **预期**：直接进入 dashboard，显示 "Dashboard" 按钮，无登录页面

---

## 故障排查

### 问题 1：容器无法启动

**症状**：`docker ps` 显示容器未运行

**诊断**：
```bash
# 查看容器日志
docker logs multica-backend-1
docker logs multica-frontend-1
```

**常见原因**：
- 端口被占用：检查 3000 和 8080 端口
- 配置错误：检查 `.env` 文件格式

### 问题 2：本地模式未启用

**症状**：访问时显示登录页面

**诊断**：
```bash
# 检查后端环境变量
docker exec multica-backend-1 env | grep MULTICA_LOCAL_MODE

# 检查健康响应
curl http://localhost:8080/health | jq
```

**解决**：
- 确认 `.env` 中 `MULTICA_LOCAL_MODE=true`
- 重启后端：`docker compose -f docker-compose.selfhost.yml restart backend`

### 问题 3：iPhone 无法访问

**症状**：Tailscale 地址无法打开

**诊断**：
```bash
# 检查 Tailscale 状态
tailscale status

# 测试本地 Tailscale 地址
curl http://fhs-mac-mini.tail626c72.ts.net:3000/health
```

**解决**：
- 确认 Mac 和 iPhone 的 Tailscale 都是绿灯
- 检查 macOS 防火墙设置
- 尝试使用 IP 地址：`http://100.94.134.78:3000`

### 问题 4：CORS 错误

**症状**：浏览器控制台显示 CORS 错误

**诊断**：
```bash
# 检查 CORS 响应头
curl -v -H "Origin: http://fhs-mac-mini.tail626c72.ts.net:3000" http://fhs-mac-mini.tail626c72.ts.net:8080/health 2>&1 | grep Access-Control
```

**解决**：
- 确认 `.env` 中 `CORS_ALLOWED_ORIGINS` 包含 Tailscale 地址
- 重启后端：`docker compose -f docker-compose.selfhost.yml restart backend`

---

## 完成确认

当以下所有检查通过时，更新完成：

- [ ] 容器状态：所有服务 Up
- [ ] 后端健康：`local_mode: enabled`
- [ ] 前端健康：`local_mode: enabled`
- [ ] Mac 浏览器：无需登录，直接进入 dashboard
- [ ] iPhone 浏览器：无需登录，直接进入 dashboard

---

## 提交更新

验证通过后，提交更新：

```bash
git add -A
git commit -m "Update from upstream - $(date +%Y%m%d)

- Merged upstream/main
- Preserved local mode modifications
- Verified Tailscale access working
"
```

---

## 附录：完整文件参考

### A. server/internal/middleware/local_auth.go

完整文件内容见项目根目录下的 `LOCAL_MODIFICATIONS.md`。

### B. packages/core/platform/auth-initializer.tsx

完整文件内容见项目根目录下的 `LOCAL_MODIFICATIONS.md`。

---

## 执行检查清单

在执行更新时，请按以下顺序确认：

- [ ] 已备份当前状态
- [ ] 已获取上游代码
- [ ] 已合并到本地
- [ ] 已检查所有关键文件
- [ ] 已恢复被覆盖的本地修改
- [ ] 已验证 .env 配置
- [ ] 已停止旧容器
- [ ] 已重新构建
- [ ] 已启动新容器
- [ ] 已验证后端健康
- [ ] 已验证前端健康
- [ ] 已验证 Tailscale 访问
- [ ] 已验证 Mac 浏览器
- [ ] 已验证 iPhone 浏览器
- [ ] 已提交更新
