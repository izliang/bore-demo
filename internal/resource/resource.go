// Package resource 管理应用的全局单例资源。
//
// 设计原则：
//   - 所有基础设施组件（DB、Logger、MQ、Config 等）以包级变量形式存储
//   - 在 bootstrap.Init() 中按依赖顺序初始化并赋值
//   - 业务代码通过 resource.XXX 直接访问（如 resource.DB、resource.Logger）
//
// 为什么使用全局变量而不是依赖注入容器？
//
//	对于中小型 Go 项目，包级变量是最简洁实用的方案：
//	  - 避免到处传递 *App 或 *Container
//	  - 测试时可以通过替换变量值来注入 mock
//	  - 清晰可见：所有全局状态集中在一个文件中
//
// 分层归属说明：
//   - resource 包属于基础设施层，不包含业务逻辑
//   - resource/api/v1/ 存放 HTTP API 的请求/响应 DTO（表示层）
//   - 命名上 resource/api 容易引发困惑，但这是框架约定，遵循现有规范
//
// 新项目复刻指南：
//   - 根据实际使用的中间件增减变量
//   - 变量命名遵循 "大写驼峰 + 简短英文" 风格
//   - 每个变量必须有注释说明用途
//   - 如果引入了 Redis，在此处新增 var Redis *redis.Client
package resource

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore/cfgstore"
	"github.com/izliang/bore/config"
	"github.com/izliang/bore/i18n"
	"github.com/izliang/bore/logit"
	"github.com/izliang/bore/queue"
	"github.com/izliang/bore/request/propagator"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

	// MC 是项目配置的别名，方便与框架配置区分。
	MC "github.com/izliang/bore-demo/internal/config"
)

// ============================================================
// 应用基础信息（来自环境变量）
// ============================================================

// SignOneHundredApp 平台注入的应用别名。
// 来源：环境变量 SIGN_ONE_HUNDRED_APP。
var SignOneHundredApp string

// SignOneHundredDomain 平台注入的域名。
// 来源：环境变量 SIGN_ONE_HUNDRED_DOMAIN。
var SignOneHundredDomain string

// PodName 当前 Pod 名称（K8s 环境）。
// 来源：环境变量 POD_NAME。
// 用途：日志标识、指标标签、分布式追踪中的实例标识。
var PodName string

// AppName 应用名称，从配置文件的 service.name 读取。
var AppName string

// Env 当前部署环境（dev / pretest / prod / saas / sig / sig_gray）。
// 来源：环境变量 SERVICE_ENV。
var Env string

// ============================================================
// 基础设施组件
// ============================================================

// Logger 全局结构化日志实例（基于 Zap）。
// 在 initLogger 中初始化，依赖 initConfig 先执行。
var Logger logit.Logger

// DB 全局数据库连接实例（基于 GORM）。
// 在 initDB 中初始化，依赖 initLogger 先执行。
var DB *gorm.DB

// Queue 全局消息队列连接实例（RabbitMQ）。
// 在 initRabbitMQ 中初始化。
// 如果项目不需要消息队列，此变量为 nil。
var Queue queue.Queue

// ============================================================
// 配置
// ============================================================

// AppConfig 应用配置的泛型容器。
// 通过 config.LoadRootFile() 加载 YAML 配置，通过 AppConfig.Get() 获取当前值。
// 支持配置热更新（配合 cfgstore）。
var AppConfig *config.Cfg[MC.AppConfig]

// CfgstoreCached 配置热加载缓存。
// 从数据库加载动态配置，支持周期性增量/全量刷新。
var CfgstoreCached *cfgstore.Config

// ============================================================
// 可观测性
// ============================================================

// Metrics Prometheus Registry 实例。
// 在 initMetrics 中初始化，注册了 Go 运行时、HTTP、DB 指标。
var Metrics *prometheus.Registry

// ============================================================
// 国际化
// ============================================================

// Localizer i18n 翻译器实例。
// 在 initI18n 中初始化，根据 LANG 环境变量选择语言。
var Localizer i18n.Localizer

// I18nBundle i18n 翻译包（*i18n.Bundle）。
// 供 amis Engine（amis.WithI18n）等需要完整 Bundle 的组件使用。
var I18nBundle *i18n.Bundle

// ============================================================
// 服务状态
// ============================================================

// Accepting 标记服务是否接受新请求（原子操作，线程安全）。
//   - 启动完成后设为 true（initAccepting）
//   - 优雅关闭时设为 false（Shutdown）
//   - 用于 K8s Readiness Probe 和负载均衡健康检查
var Accepting atomic.Bool

// ============================================================
// 中间件 & 传播器
// ============================================================

// SREPropagator 链路追踪传播器。
// 用于在微服务之间传递 TraceID、SpanID 等上下文信息。
var SREPropagator propagator.Propagator

// GinMiddlewares 全局 Gin 中间件链。
// 在 bootstrap 各 init 方法中按顺序追加：
//  1. Access 日志中间件（initLogger）
//  2. HTTP 指标中间件（initMetrics）
//
// 在 router.SetupRoutes 中被注册到 Gin Engine。
//
// 注意：使用 make 初始化而非 nil，确保后续 append 安全。
var GinMiddlewares []gin.HandlerFunc = make([]gin.HandlerFunc, 0)

var Trace trace.Tracer

var ShutDownFuncs []func() = make([]func(), 0)
