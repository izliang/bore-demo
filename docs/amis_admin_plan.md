# bore-demo AMIS 管理后台 —— 整体方案（含 RBAC 权限设计）

> 目标：基于 bore 的 `web/amis` SDK（JSON Schema → 管理后台），为 bore-demo 生成一套
> 完整可运行的管理后台，并内置**管理员（Admin）模型 + 若依风格 RBAC 权限体系**。
>
> 方案状态：**已实施完成**（实施记录见文末第 15 节；方案主体为实施前评审稿，个别细节以代码为准）

---

## 1. 背景与现状

- 仓库（正确路径）：
  - bore：`/Users/izliang/Documents/GitHub/bore`（模块 `github.com/izliang/bore`）
  - bore-demo：`/Users/izliang/Documents/GitHub/bore-demo`（模块 `github.com/izliang/bore-demo`）
  - 开发工作区入口：`/Users/izliang/Projects/GitHubProjects/bore-wedding-workspace`（软链指向上述两仓库）
- bore 提供 `web/amis` 包：注册 JSON Schema 页面 + 内置登录鉴权（`SetupAuth` / `RequireAuth`），
  模板内置 `adaptAmisResponse` 自动适配 Envelope 响应（`{"code":"000000","message":"success","data":{...}}`）。
- bore 自带完整示例：`bore/web/amis/example/`（login / dashboard / users / models / settings），
  是本次后台的页面与代码蓝本。
- RBAC 参照：若依（RuoYi）经典模型，已核对官方最新建表 SQL（`ry_20260319.sql`）：
  5 张核心表 `sys_user / sys_role / sys_menu / sys_user_role / sys_role_menu`、
  权限标识 `module:resource:action`（如 `system:user:list`）、菜单类型 `M目录/C菜单/F按钮`、
  角色 `data_scope` 数据范围、超管角色 `role_key=admin` 全部放行。
- bore-demo 现状：
  - 分层架构已就绪：`model → dao → service → controller → router`，仅有一个 User 读示例。
  - **存在两个已确认的启动缺陷（实施时必须先修复）**：
    1. `internal/bootstrap/bootstrap.go` 调用 `config.WithTemplate(...)`，该 API 在 bore 中不存在
       （应为 `cfg.SetTemplates(map[string]string{...})` + `LoadRootFile(path)`）。
       已在 `/tmp/bore-demo-build-check` 副本验证：修复后 `go build ./...` 与 `go vet ./...` 均通过。
    2. 本地开发需要 `replace github.com/izliang/bore => ../bore`（发布版 `v0.0.1` 同样存在 API 不匹配）。
  - 启动时强制要求环境变量 `SIGN_ONE_HUNDRED_APP` / `SIGN_ONE_HUNDRED_DOMAIN`（`initEnv` 缺省 panic），
    本地运行验证需先导出，例如 `SIGN_ONE_HUNDRED_APP=bore-demo SIGN_ONE_HUNDRED_DOMAIN=localhost`。

## 2. 总体架构

```
浏览器 (amis SDK)
   │  /admin/* 页面 + /api/login + /api/v1/admin|role|menu|auth/* API
   ▼
Gin Engine
   ├── amis Engine (bore web/amis)
   │     ├── /login            登录页（SetupAuth 自动注册）
   │     ├── /api/login        登录 API（Authenticator → Admin Service 校验）
   │     ├── /api/logout       登出 API
   │     └── /admin/*          受保护页面（RequireAuth + BindGin）
   ├── cyberhuma API v1
   │     └── /api/v1/*         业务 API（统一挂 RequireAuth，接口级再挂 RequirePerms）
   └── 现有 /api/v1/user/*、/health 保持不变

分层职责（沿用 demo 约定）：
  internal/model/dao/{admin,role,menu}.go        GORM Entity + Dao 接口/实现
  internal/model/service/{admin,role,menu}/      RBAC 业务逻辑
  internal/api/admin/                            amis Engine 装配 + 页面 JSON + 鉴权/权限中间件
  internal/api/controller/{admin,role,menu}.go   DTO ↔ Domain 转换 + 调用 Service
  internal/api/router/router.go                  路由注册入口
```

## 3. RBAC 权限设计（若依风格）

### 3.1 数据模型（5 张表）

```
admins      (id, username, password_hash, nickname, avatar, email, phone,
             status, last_login_at, created_at, updated_at)          -- 不存角色字段，走关联
roles       (id, role_name, role_key, role_sort, data_scope, status,
             remark, created_at, updated_at)                         -- data_scope 预留（本期仅 '1' 全部）
menus       (id, parent_id, menu_name, menu_type, path, perms, icon,
             order_num, visible, created_at, updated_at)             -- menu_type: M目录/C菜单/F按钮
admin_roles (admin_id, role_id)  PK(admin_id, role_id)               -- 管理员 1-N 角色
role_menus  (role_id, menu_id)   PK(role_id, menu_id)                -- 角色 1-N 菜单/按钮权限
```

> 与若依的映射：`admins→sys_user`、`roles→sys_role`、`menus→sys_menu`、
> `admin_roles→sys_user_role`、`role_menus→sys_role_menu`。
> demo 无部门概念，`dept`/`post` 表与 `data_scope` 自定义范围暂不实现（字段预留）。

### 3.2 权限标识规范（若依风格）

- 命名：`模块:资源:操作`
- 预置权限（菜单表种子数据）：

| 模块 | 权限标识 | 说明 |
|---|---|---|
| 仪表盘 | `dashboard:view` | 查看仪表盘 |
| 管理员 | `admin:list` `admin:query` `admin:add` `admin:edit` `admin:remove` | 管理员 CRUD |
| 角色 | `role:list` `role:query` `role:add` `role:edit` `role:remove` `role:assign` | 角色 CRUD + 分配菜单权限 |
| 菜单 | `menu:list` `menu:query` `menu:add` `menu:edit` `menu:remove` | 菜单/权限管理 |

- 超级管理员：种子角色 `role_key=super_admin`（对应若依 `admin`），所有接口直接放行（等价 `*:*:*`）。
- 普通角色种子：`role_key=admin`，仅基础权限（`dashboard:view` + `admin:list/query`）。

### 3.3 校验流程

```
登录（POST /api/login）
  → Admin Service.Login（bcrypt + 状态校验）
  → 生成 token "admin-{id}" 写 Cookie
  → 加载权限缓存：admin_id → {roles[], perms[]}（demo 用内存 map + TTL 5min）

请求（/api/v1/**）
  → RequireAuth 中间件：解析 token → admin_id（无/无效 → 401）
  → RequirePerms("admin:add") 中间件（按路由挂载）：
      admin_id → 查角色（status 正常）→ 聚合 perms
      → 命中 perms 或超管 → 放行；否则 403 {code:A0007, message:无权限}

前端（amis）
  → GET /api/v1/auth/info 返回 {roles:[], perms:[]}
  → 页面 service 组件把 perms 注入页面 data，按钮 visibleOn 控制显隐
  → 前端隐藏仅为体验，后端 RequirePerms 是安全底线
```

### 3.4 权限缓存

- 一期：`internal/api/admin/session.go` 内存缓存（`sync.Map` + TTL 5 分钟），
  角色/菜单变更时主动失效对应 admin 的缓存（或全量失效，demo 可接受）。
- 生产演进：换 bore `cache` 抽象（Redis provider 已具备），缓存键 `rbac:perms:{adminID}`。

### 3.5 amis 按钮级权限

- 登录后页面通过隐藏 `service` 组件请求 `/api/v1/auth/info`，将结果写入页面 `data.perms`。
- 按钮显隐示例（amis 表达式）：

```json
{ "type": "button", "label": "新增管理员", "visibleOn": "this.perms.indexOf('admin:add') > -1" }
```

- 菜单/导航动态过滤（是否显示「角色管理」入口）：
  `visibleOn: "this.perms.indexOf('role:list') > -1"`。
- 一期：固定导航 + 按钮级控制；二期可升级为按 perms 动态生成导航。

### 3.6 种子数据

- 角色：`super_admin`（全权限）、`admin`（基础权限）。
- 菜单：仪表盘（`dashboard:view`）、管理员管理（目录 + 5 按钮）、角色管理（目录 + 6 按钮）、
  菜单管理（目录 + 5 按钮）。
- 默认管理员 `admin / admin123` 绑定 `super_admin`（启动日志提示，后续配置化）。
- 种子幂等：仅当表为空/无超管时写入。

## 4. Admin / Role / Menu 模型设计

### 4.1 领域模型 `internal/model/admin.go`（admin 为主聚合）

```go
type Admin struct {
    ID           uint64
    Username     string      // 唯一
    PasswordHash string      // bcrypt，对外绝不输出
    Nickname     string
    Avatar       string
    Email        string
    Phone        string
    Status       AdminStatus // 1 启用 / 0 禁用
    LastLoginAt  *time.Time
    Roles        []Role      // 关联角色（列表/详情时填充）
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
type Role struct {
    ID        uint64
    RoleName  string
    RoleKey   string // super_admin / admin / 自定义
    DataScope string // 预留：'1' 全部数据
    Status    int8
    MenuIDs   []uint64 // 分配权限时使用
}
type Menu struct {
    ID        uint64
    ParentID  uint64
    MenuName  string
    MenuType  string // M 目录 / C 菜单 / F 按钮
    Path      string // 页面路由（目录/菜单）
    Perms     string // 权限标识（按钮/菜单）
    Icon      string
    OrderNum  int
    Visible   int8
}
```

- 领域模型无 ORM/JSON 标签；DAO Entity 在 `internal/model/dao/{admin,role,menu}.go` 带 gorm tag。
- `menus` 树形结构由 Service 组装（parent_id + 排序）。

### 4.2 DAO 接口

```go
// AdminDao（admin.go）
GetAdmin(id) / GetAdminByUsername(username) / CreateAdmin(a) / UpdateAdmin(a) /
DeleteAdmin(id) / ListAdmins(keyword, offset, limit) / CountAdmins()
// 角色关联
ReplaceAdminRoles(adminID, roleIDs []uint64) / GetRoleIDsByAdminID(adminID)

// RoleDao（role.go）
GetRole(id) / GetRoleByKey(key) / CreateRole(r) / UpdateRole(r) / DeleteRole(id) /
ListRoles(keyword, offset, limit) / CountRoles() / ReplaceRoleMenus(roleID, menuIDs) /
GetMenuIDsByRoleID(roleID)

// MenuDao（menu.go）
GetMenu(id) / CreateMenu(m) / UpdateMenu(m) / DeleteMenu(id) /
ListMenus() / GetPermsByAdminID(adminID) // JOIN 聚合权限标识（供鉴权）
```

### 4.3 Service

- `service/admin`：Login（bcrypt + 状态 + 记录 LastLoginAt）、CRUD、分配角色、
  保护规则（不能删除自己、不能删除最后一个 super_admin、不能禁用自己的超管身份）。
- `service/role`：CRUD、分配菜单权限、删除前校验（被 admin 引用时禁止删除/或级联清理）。
- `service/menu`：CRUD、菜单树组装。
- `service/auth`（或并入 admin）：`GetPerms(adminID)` 聚合 roles+perms（供缓存与 /auth/info）。
- 错误码（模块 `admin`/`rbac` 注册到 `e`）：
  `A0001 管理员不存在`、`A0002 用户名已存在`、`A0003 用户名或密码错误`、
  `A0004 账号已禁用`、`A0005 不能删除自身`、`A0006 至少保留一个超级管理员`、
  `A0007 无权限`、`A0008 角色不存在`、`A0009 角色被引用不可删除`。

## 5. 数据库迁移与种子数据

- 新增 `internal/model/dao/migrate.go`：

```go
func Migrate(atx appctx.AppContext) error {
    // AutoMigrate: Admin / Role / Menu / AdminRole / RoleMenu
}
func Seed(atx appctx.AppContext) error {
    // 角色种子（super_admin / admin）→ 菜单种子 → 默认管理员（admin/admin123，绑定 super_admin）
}
```

- 挂载点：`bootstrap.Init` 中 `initDB` 之后新增 `initAdminDB(ctx)`（幂等）。
- 注意：dev 配置连接 `192.168.160.66:3306/bore_demo`，验证前需保证 MySQL 可达。

## 6. 管理后台 API 设计

统一 Envelope 响应（`{"code":"000000","message":"success","data":{...}}`），amis 模板适配器自动转换。
所有 `/api/v1/**` 先过 `RequireAuth`，再按路由挂 `RequirePerms`。

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | `/api/login` | 公开 | 登录（amis SetupAuth 自动注册） |
| GET/POST | `/api/logout` | 登录态 | 登出 |
| GET | `/api/v1/auth/info` | 登录态 | 当前管理员信息 + roles + perms（前端按钮显隐） |
| GET | `/api/v1/admin/list` | `admin:list` | 管理员分页列表（含角色） |
| GET | `/api/v1/admin/get` | `admin:query` | 管理员详情（含角色/角色菜单） |
| POST | `/api/v1/admin/create` | `admin:add` | 新增（含角色分配 role_ids） |
| PUT | `/api/v1/admin/update` | `admin:edit` | 更新/重置密码/改角色 |
| DELETE | `/api/v1/admin/delete` | `admin:remove` | 删除单个 |
| POST | `/api/v1/admin/batch-delete` | `admin:remove` | 批量删除 |
| GET | `/api/v1/role/list` | `role:list` | 角色分页列表 |
| POST | `/api/v1/role/create` | `role:add` | 新增角色 |
| PUT | `/api/v1/role/update` | `role:edit` | 更新角色 + 分配菜单 menu_ids |
| DELETE | `/api/v1/role/delete` | `role:remove` | 删除角色 |
| GET | `/api/v1/menu/tree` | `role:assign` / `menu:list` | 菜单树（角色分配权限用） |
| GET | `/api/v1/menu/list` | `menu:list` | 菜单/权限列表 |
| POST | `/api/v1/menu/create` | `menu:add` | 新增菜单/按钮 |
| PUT | `/api/v1/menu/update` | `menu:edit` | 更新菜单/按钮 |
| DELETE | `/api/v1/menu/delete` | `menu:remove` | 删除菜单/按钮 |
| GET | `/api/v1/admin/stats` | 登录态 | 仪表盘统计（管理员数、角色数） |

- 控制器沿用 `controller.GetUser` 的 `appctx.New → dao.New → service.New → DTO↔Domain` 模式。
- DTO 定义在 `internal/resource/api/v1/admin_*.go / role_*.go / menu_*.go / auth_*.go`。
- 响应 DTO 一律不包含 `password_hash`。

## 7. AMIS 页面设计

页面目录：`internal/api/admin/pages/*.json`（路径用 `config.GetProjectRootDir()` 拼绝对路径，避免 cwd 依赖）。

| 页面 | 路由 | 权限 | 内容 |
|---|---|---|---|
| 登录页 | `/login` | 公开 | 自定义登录页（参考 bore example login.json，品牌化） |
| 仪表盘 | `/admin/dashboard` | `dashboard:view` | 统计卡片（`/api/v1/admin/stats`）+ 欢迎区 |
| 管理员管理 | `/admin/admins` | `admin:list` | CRUD 表格：搜索、新增/编辑弹窗（含角色多选）、删除/批量删除、状态/角色 mapping 标签；按钮按 perms 显隐 |
| 角色管理 | `/admin/roles` | `role:list` | 角色 CRUD + 菜单权限树勾选分配（amis tree + 弹窗） |
| 菜单管理 | `/admin/menus` | `menu:list` | 菜单/按钮权限维护（树形表格，简版） |
| 系统设置 | `/admin/settings` | 登录态 | 静态设置表单（后续接配置） |

- 页面共享左侧导航 aside（按 perms 显隐：`dashboard / admins / roles / menus / settings`）。
- 顶部 toolbar：当前登录用户名 + 登出按钮。
- 品牌与暗黑模式样式通过 `WithExtraCSS` 注入（复制 bore 示例 extraCSS 并微调）。
- 页面文案全部使用 `$t.*` key，由引擎服务端注入翻译。

## 8. 国际化（i18n）

- 扩充 `locales/zh.json` 与 `locales/en.json`（两文件 key 必须一致，bootstrap StrictMode 启动校验）。
- 新增 key 分组：`auth.*`、`common.*`、`dashboard.*`、`admins.*`、`roles.*`、`menus.*`、`settings.*`、`label.*`。
- amis Engine 使用 `amis.WithI18n(I18nConfig{Bundle: bundle, StrictMode: true})`。
- 当前 `initI18n` 只把 `Localizer` 存入 `resource.Localizer`；需补充 `resource.I18nBundle = bundle`
  供 amis Engine 使用。
- 路由挂载 `middleware.LanguageMiddleware()`（bore `web/middleware/language.go` 已提供，demo 尚未挂载）。

## 9. 路由与鉴权集成

`internal/api/router/router.go` 增加（或新增 `internal/api/admin/setup.go` 被其调用）：

```go
// 1. 装配 amis Engine
engine := amis.New(amis.WithCDN("6.13.0", amis.ThemeCXD),
    amis.WithTitle("bore-demo Admin"),
    amis.WithExtraCSS(adminExtraCSS),
    amis.WithI18n(amis.I18nConfig{Bundle: resource.I18nBundle, StrictMode: true}))
engine.RegisterJSONFile("/login", pagesDir+"/login.json")
engine.RegisterJSONFile("/admin/dashboard", pagesDir+"/dashboard.json")
engine.RegisterJSONFile("/admin/admins", pagesDir+"/admins.json")
engine.RegisterJSONFile("/admin/roles", pagesDir+"/roles.json")
engine.RegisterJSONFile("/admin/menus", pagesDir+"/menus.json")
engine.RegisterJSONFile("/admin/settings", pagesDir+"/settings.json")

// 2. 鉴权 + 权限
cfg := amis.DefaultAuthConfig()
cfg.HomePath = "/admin/dashboard"
cfg.Messages = amis.AuthMessagesZH()
engine.SetupAuth(r, cfg, adminAuthenticator) // → Admin Service.Login + 权限缓存预热

protected := r.Group("/admin", cfg.RequireAuth())
engine.BindGin(protected)

// 3. API 组：登录态 + 权限中间件
apiV1 := apiRouter.Group("/api/v1")
adminApi := apiV1.Group("/admin", authMW.RequireAuth(), authMW.RequirePerms("admin:list"))
// ... 各路由挂对应 RequirePerms
```

- `adminAuthenticator` 位于 `internal/api/admin/auth.go`：构建请求级 `appctx`，调
  `admin.New(dao.New(atx)).Login(...)`；成功返回 token `admin-{id}`（demo 用固定会话 token，
  Cookie 由 `LoginHandler` 写入；真实项目建议接 Redis 会话 / 签名 JWT）。
- 权限中间件 `internal/api/admin/perm.go`：`RequireAuth()`（登录态）+ `RequirePerms(perm)`（RBAC 校验，
  超管放行），配合 `session.go` 内存缓存。
- 注意 `SetupAuth` 不安装鉴权中间件，`/admin` 组手动挂 `cfg.RequireAuth()`（与 bore 示例一致）。

## 10. 依赖与构建

- **必须修复的两个前置问题**（否则 `go build ./...` 失败）：
  1. `bootstrap.go`：`config.WithTemplate(...)` → `SetTemplates(map)` + `LoadRootFile(path)`。
  2. `go.mod` 增加本地替换：`replace github.com/izliang/bore => ../bore`
     （`/Users/izliang/Documents/GitHub/bore`；发布前移除并确认已发布版本 API 匹配）。
- 新增依赖：`golang.org/x/crypto`（bcrypt；当前在 go.sum 为 indirect，转正为直接依赖）。
- 依赖树无需大改：gin、gorm、cyberhuma 等已在 go.mod 中。

## 11. 测试计划

- DAO（sqlmock，沿用 `user_test.go` table-driven + `appctx.NewSQLMock` 风格）：
  `admin_test.go`（GetAdminByUsername / ListAdmins / ReplaceAdminRoles）、
  `role_test.go`、`menu_test.go`（GetPermsByAdminID JOIN 聚合）。
- Service（mock DAO）：`admin_test.go`（Login 成功/密码错误/账号禁用、唯一性、删除保护）、
  `role_test.go`（删除被引用角色保护、分配菜单）。
- 权限中间件：`perm_test.go`（无 token / 无权限 403 / 超管放行 / 缓存命中）。
- 手工验证：导出 `SIGN_ONE_HUNDRED_APP` / `SIGN_ONE_HUNDRED_DOMAIN` 后
  `go run ./cmd/demo` → 浏览器访问 `http://localhost:8000/admin/dashboard`，
  未登录 302 到 `/login`，`admin / admin123` 登录后可增删改查管理员；
  新建一个 `admin` 角色管理员验证按钮显隐与接口 403。

## 12. 实施步骤（评审通过后执行）

1. 修复前置构建问题（bootstrap `WithTemplate` → `SetTemplates`；go.mod 加本地 replace），
   确保 `go build ./...` 与 `go vet ./...` 通过。
2. RBAC 领域模型 + DAO Entity（Admin / Role / Menu / 关联表）+ DAO 接口/实现。
3. Service 层（admin / role / menu / auth-perms，含 bcrypt 与保护规则）。
4. 迁移与种子（AutoMigrate 5 表 + 角色/菜单/默认管理员种子，接入 bootstrap `initAdminDB`）。
5. 权限中间件（RequireAuth / RequirePerms / 内存缓存）+ `/api/v1/auth/info`。
6. RBAC API（admin / role / menu CRUD + DTO + controller + cyberhuma 路由，挂权限）。
7. amis Engine 装配 + 鉴权（`internal/api/admin/`：engine / auth / perm / session / extraCSS）。
8. AMIS 页面 JSON（login / dashboard / admins / roles / menus / settings）+ 共享导航 + 按钮显隐。
9. i18n 词条（zh/en）扩充 + `resource.I18nBundle` + 语言中间件挂载。
10. 单元测试（dao + service + 权限中间件）。
11. 构建、运行、浏览器全流程验证（登录 → 权限显隐 → CRUD → 角色分配 → 登出）。

## 13. 分期与扩展

- 一期（本次实施）：5 表 RBAC + 权限中间件 + 管理员/角色/菜单 API + amis 页面 + 按钮级显隐。
- 二期（预留）：
  - `data_scope` 数据权限（部门表 + 自定义数据范围，若依 data_scope 1-5）。
  - 动态导航（按 perms 生成 aside 菜单）。
  - 会话升级（Redis 会话 / JWT）+ 权限缓存换 bore `cache`（Redis）。
  - 操作日志（若依 sys_oper_log / sys_logininfor 思路）。

## 14. 风险与决策点

| 风险/决策 | 说明 | 决策建议 |
|---|---|---|
| bore 依赖 API 不匹配 | demo 引用了不存在的 `config.WithTemplate`；发布版 v0.0.1 同样不匹配 | 实施第 1 步先修复并本地 replace |
| 本地运行环境变量 | `initEnv` 缺 `SIGN_ONE_HUNDRED_APP/DOMAIN` 会 panic | 验证脚本/说明中导出，或后续放宽校验 |
| 数据库可达性 | dev 指向内网 MySQL，无库则后台无法登录 | 验证前准备可达 MySQL（建 `bore_demo` 库） |
| cyberhuma 分页参数 | amis 用 `page/perPage`，与 Huma 默认参数兼容性待验证 | 不兼容则 admin API 改用原生 gin + amis 原生响应 |
| 会话安全 | demo 用固定 token 存 Cookie | 仅限 demo；真实项目换 Redis 会话/JWT |
| 权限缓存一致性 | 角色/菜单变更后缓存需失效 | 变更接口主动清缓存；TTL 5min 兜底 |
| 默认管理员密码 | `admin/admin123` 硬编码种子 | 启动日志提示 + 配置化预留 |
| amis CDN 版本 | 固定 `6.13.0` 与 bore 示例一致 | 内网部署可换 `WithAssets` 自定义 CDN |

## 15. 实施完成记录（2026-08-04）

- 前置修复：`bootstrap.go` 改用 `SetTemplates(map)` + `LoadRootFile(path)`；`go.mod` 增加
  `replace github.com/izliang/bore => ../bore`。`go build ./...`、`go vet ./...` 通过。
- RBAC 数据层：`internal/model/{admin,role,menu}.go` + `internal/model/dao/`（5 表 AutoMigrate、
  幂等种子：2 个内置角色、18 个菜单节点、默认 `admin/admin123`）。
- Service 层：`internal/model/service/{admin,role,menu,auth}/`，保护规则齐备
  （不能删自己/最后一个超管、内置角色不可改删、被引用角色不可删、有子菜单不可删、菜单防环）。
- API 层：`internal/api/controller/{admin,role,menu}.go`（cyberhuma）+ DTO；
  每个入口调用 `adminapi.CheckPerm(ctx, "xxx")` 做接口级权限点校验。
- 运行时权限：`internal/api/admin/`（Session 内存缓存 TTL 5min、`admin-{id}` Cookie token、
  `RequireLogin` + `CheckPerm`、amis Engine 装配、自定义登录页）。
- 页面与 i18n：`internal/api/admin/pages/*.json` 6 个页面、`locales/{zh,en}.json` 各 66 条词条，
  `resource.I18nBundle` + 语言中间件接入。
- 单元测试（全部通过）：dao sqlmock（admin/role/menu）、service mock DAO（admin/role/menu/auth）、
  权限中间件/session/ctx（gin + 内存缓存）、security bcrypt。
- 待办（需外部环境）：连接 MySQL 后手工全流程验证（登录 → 权限显隐 → CRUD → 角色分配 → 登出），
  见第 11 节；本地运行需先导出 `SIGN_ONE_HUNDRED_APP` / `SIGN_ONE_HUNDRED_DOMAIN`。
