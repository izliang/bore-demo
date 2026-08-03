// Package bootstrap 负责应用启动时的基础设施初始化。
//
// 初始化按依赖顺序严格编排，确保每个组件初始化时其上游依赖已就绪：
//
//	initEnv        → 读取必需的环境变量
//	initConfig     → 加载 YAML 配置文件（依赖 env）
//	initLogger     → 初始化结构化日志（依赖 config）
//	initDB         → 建立数据库连接池（依赖 config + logger）
//	initRabbitMQ   → 建立消息队列连接（依赖 config + logger）
//	initI18n       → 加载国际化资源（依赖 env + logger）
//	initMetrics    → 注册 Prometheus 指标（依赖 DB）
//	initCachedStore→ 初始化配置热加载缓存（依赖 DB + logger）
//	initAccepting  → 标记服务就绪
//
// 新项目复刻指南：
//   - 如果不需要 RabbitMQ，删除 initRabbitMQ 及相关配置
//   - 如果不需要 i18n，删除 initI18n 及相关配置
//   - 如果不需要 Metrics，删除 initMetrics 及相关配置
//   - 如果需要 Redis，在此处新增 initRedis
//   - 初始化顺序取决于依赖关系，严禁随意调整
package bootstrap

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	MC "github.com/izliang/bore-demo/internal/config"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore-demo/internal/resource"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/config"
	"github.com/izliang/bore/db"
	"github.com/izliang/bore/i18n"
	"github.com/izliang/bore/logit"
	"github.com/izliang/bore/metrics"
	"github.com/izliang/bore/metrics/provider"
	"github.com/izliang/bore/queue/rabbitmq"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	itelresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Init 按依赖顺序执行所有基础设施组件的初始化。
// 任意初始化步骤失败会直接 panic，因为在缺少基础设施的情况下服务无法运行。
func Init(ctx context.Context) {
	// 【步骤1】环境变量 —— 最底层依赖，必须在所有组件之前初始化。
	initEnv(ctx)

	// 【步骤2】配置文件 —— 依赖环境变量中指定的 SERVICE_ENV 来决定加载哪个配置。
	initConfig(ctx)

	// 【步骤3】日志 —— 依赖配置中的日志级别设置。
	// 日志初始化后，后续组件可以使用结构化日志记录启动过程。
	initLogger(ctx)

	// 【步骤4】数据库 —— 依赖配置中的 DSN 和 Logger。
	initDB(ctx)

	// 【步骤5】RBAC 表结构迁移与种子数据 —— 依赖 DB。
	initAdminDB(ctx)

	// 【步骤6】消息队列 —— 依赖配置中的 RabbitMQ 连接参数和 Logger。
	initRabbitMQ(ctx)

	// 【步骤7】国际化 —— 依赖环境变量 LANG 和 Logger。
	initI18n(ctx)

	// 【步骤8】指标 —— 依赖 DB 实例来采集连接池指标。
	initMetrics(ctx)

	// 【步骤9】配置热加载 —— 依赖 DB 和 Logger，周期性刷新缓存配置。
	initCachedStore(ctx)

	// 【步骤10】就绪标记 —— 所有组件初始化完毕后，标记服务可接受请求。
	initAccepting(ctx)

	initTrace(ctx)
}

// initEnv 读取平台注入的必需环境变量。
//
// 环境变量清单：
//   - SIGN_ONE_HUNDRED_APP：应用别名，用于配置模板渲染
//   - SIGN_ONE_HUNDRED_DOMAIN：平台域名，用于配置模板渲染
//   - POD_NAME：Pod 名称，用于日志追踪和指标标签
func initEnv(_ context.Context) {
	// SIGN_ONE_HUNDRED_APP 是平台注入的应用别名，配置文件中通过模板变量引用。
	signOneHundredApp := "app"
	//os.Getenv("SIGN_ONE_HUNDRED_APP")
	if signOneHundredApp == "" {
		panic("env SIGN_ONE_HUNDRED_APP must be set")
	}
	resource.SignOneHundredApp = signOneHundredApp

	// POD_NAME 用于日志中标识 Pod 实例，容器环境由 K8s Downward API 注入。
	podName := "app-pod"
	//os.Getenv("POD_NAME")
	resource.PodName = podName

	// SIGN_ONE_HUNDRED_DOMAIN 是平台域名，配置文件中通过模板变量引用。
	signOneHundredDomain := "domain"
	//os.Getenv("SIGN_ONE_HUNDRED_DOMAIN")
	if signOneHundredDomain == "" {
		panic("env SIGN_ONE_HUNDRED_DOMAIN must be set")
	}
	resource.SignOneHundredDomain = signOneHundredDomain
}

// initConfig 根据 SERVICE_ENV 环境变量选择对应的 YAML 配置文件并加载。
//
// 环境 → 配置文件映射：
//
//	pretest → config_pretest.yaml    # 预发环境
//	prod    → config_prod.yaml       # 生产环境
//	saas    → config_saas.yaml       # SaaS 环境
//	sig     → config_sig.yaml        # 签名环境
//	sig_gray→ config_sig_gray.yaml   # 签名灰度环境
//	(其他)  → config_dev.yaml        # 本地开发环境（默认）
//
// 配置文件通过模板变量渲染，支持将环境变量注入到配置值中。
func initConfig(_ context.Context) {
	// 读取部署环境标识。
	env := os.Getenv("SERVICE_ENV")

	// 根据环境选择对应的配置文件。
	var configFile string
	switch env {
	case "pretest":
		configFile = "config_pretest.yaml"
	case "prod":
		configFile = "config_prod.yaml"
	case "saas":
		configFile = "config_saas.yaml"
	case "sig":
		configFile = "config_sig.yaml"
	case "sig_gray":
		configFile = "config_sig_gray.yaml"
	default:
		configFile = "config_dev.yaml"
	}

	// 加载配置文件并用环境变量渲染模板占位符。
	configPath := filepath.Join("conf", configFile)
	resource.AppConfig = config.New[MC.AppConfig]()
	resource.AppConfig.SetTemplates(map[string]string{
		"SignOneHundredApp":    resource.SignOneHundredApp,
		"SignOneHundredDomain": resource.SignOneHundredDomain,
	})
	if err := resource.AppConfig.LoadRootFile(configPath); err != nil {
		panic(fmt.Errorf("failed to load config from %s: %w", configPath, err))
	}

	// 将环境和应用名存储到全局资源中供后续使用。
	resource.Env = env
	resource.AppName = resource.AppConfig.Get().Service.Name
}

// initLogger 初始化结构化日志（基于 Zap）。
//
// 日志配置：
//   - 同时输出到标准输出和文件（OutputStdFile 模式）
//   - 文件滚动策略：单文件最大 100MB，保留 30 天，压缩归档
//   - 自动注入 TraceID 便于链路追踪
//   - 非 dev 环境自动切换 Gin 为 Release 模式
//   - 将 Gin 内部日志重定向到结构化日志
//   - 注册 Access 日志中间件（跳过健康检查等路径）
func initLogger(_ context.Context) {
	// 构建 Zap 日志配置。
	conf := logit.ZapConfig{
		Config: logit.Config{
			Level:   resource.AppConfig.Get().Service.LogLevel,
			PodName: resource.PodName,
			TraceID: &logit.DefaultTraceID{},
		},
		Mode:       logit.OutputStdFile,
		AppName:    resource.AppName,
		FileName:   fmt.Sprintf("%s.log", resource.AppName),
		MaxSizeMB:  100,
		MaxAgeDays: 30,
		MaxBackups: 1,
		Compress:   true,
	}

	// 先用简单 logger 输出初始化过程中的日志（Zap 初始化之前）。
	tmpLog := logit.NewSimple(os.Stdout)
	tmpLog.Info(context.Background(), "init logger config", logit.JSONAbsField("config", conf))

	// 创建 Zap logger 并存储到全局资源。
	//resource.Logger = logit.NewZapWithConfig(conf)
	resource.Logger = logit.NewSimple(os.Stdout)

	// 非开发环境关闭 Gin 的调试输出（避免启动时打印大量 [GIN-debug] 路由信息）。
	if resource.Env != "" && resource.Env != "dev" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 将 Gin 内部日志（[GIN-debug]、[WARNING] 等）重定向到结构化日志。
	gw := logit.NewGinLogWriter(resource.Logger)
	gin.DefaultWriter = gw
	gin.DefaultErrorWriter = gw

	// 注册 Access 日志中间件：记录每个 HTTP 请求的耗时、状态码等信息。
	// 跳过健康检查、指标和探活路径，避免日志噪音。
	resource.GinMiddlewares = append(resource.GinMiddlewares, logit.AccessMiddleware(resource.Logger,
		&logit.GinAccessConfig{
			SkipPaths: []string{
				"/health",
				"/metrics",
				"/ping",
			},
		},
	))
}

// initAdminDB 初始化 RBAC 数据表并写入种子数据。
//
// 依赖 initDB 完成数据库连接；幂等，可安全重复启动。
func initAdminDB(ctx context.Context) {
	atx := appctx.New(
		appctx.WithContext(ctx),
		appctx.WithLogger(resource.Logger),
		appctx.WithGORM(resource.DB),
	)
	if err := dao.Migrate(atx); err != nil {
		panic(fmt.Errorf("init admin db migrate: %w", err))
	}
	if err := dao.Seed(atx); err != nil {
		panic(fmt.Errorf("init admin db seed: %w", err))
	}
	resource.Logger.Info(ctx, "init admin db (migrate + seed) done")
}

// initRabbitMQ 初始化 RabbitMQ 消息队列连接。
//
// 连接参数从配置文件读取，支持 TLS 加密连接。
// 生产连接池大小为 10，断线后自动重连（1s 起步，最大 30s 间隔）。
//
// 新项目复刻指南：
//   - 如果项目不需要消息队列，可以删除此方法
//   - 如果使用其他 MQ（如 Kafka），替换为对应的初始化逻辑
func initRabbitMQ(_ context.Context) {
	// 构建 RabbitMQ 连接参数。
	opt := rabbitmq.Options{
		Host:             resource.AppConfig.Get().RabbitMQ.Host,
		Port:             resource.AppConfig.Get().RabbitMQ.Port,
		Username:         resource.AppConfig.Get().RabbitMQ.Username,
		Password:         resource.AppConfig.Get().RabbitMQ.Password,
		Vhost:            resource.AppConfig.Get().RabbitMQ.Vhost,
		ProducerPoolSize: 10,
		ReconnectInitial: 1 * time.Second,
		ReconnectMax:     30 * time.Second,
		Logger:           resource.Logger,
		Propagator:       resource.SREPropagator,
		DeclareVhost:     resource.AppConfig.Get().RabbitMQ.DeclareVHost,
	}

	// 如果配置开启了 SSL，设置 TLS 配置。
	if resource.AppConfig.Get().RabbitMQ.SSL {
		opt.TLSConfig = &tls.Config{}
	}

	// 建立连接并存储到全局资源。
	q, err := rabbitmq.New(opt)
	if err != nil {
		panic(fmt.Errorf("init bore rabbitmq: %w", err))
	}
	resource.Queue = q
}

// initDB 初始化数据库连接（基于 GORM）。
//
// DSN 从配置文件读取，Logger 注入到 GORM 以便记录 SQL 日志。
// 注意：此方法必须在 initLogger 之后调用，因为需要 resource.Logger 已就绪。
//
// 新项目复刻指南：
//   - 项目需要改名为你自己的数据库名
//   - 如果有多数据源，新增 initDB2、initDBReadOnly 等方法
func initDB(_ context.Context) {
	cfg := db.DefaultConfig()
	cfg.DSN = resource.AppConfig.Get().Database.DSN

	// 防御性检查：Logger 必须在 DB 之前初始化。
	if resource.Logger == nil {
		panic("resource.Logger is nil. initDB init must before init logger")
	}
	cfg.Logger = resource.Logger

	// 打开数据库连接并存储到全局资源。
	gorm := db.MustOpen(cfg)
	resource.DB = gorm

	resource.ShutDownFuncs = append(resource.ShutDownFuncs, func() {
		if sqlDB, err := resource.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

// initI18n 初始化国际化资源。
//
// 根据 LANG 环境变量选择语言（默认 zh），从 locales 目录加载翻译文件。
// 严格模式：缺少翻译 key 时会 panic，避免线上出现未翻译的文案。
//
// 新项目复刻指南：
//   - 项目不需要多语言的话，删除此方法
//   - locales 目录需要手动创建并放入对应语言的翻译文件
func initI18n(ctx context.Context) {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = "zh"
	}

	// 获取项目根目录，用于定位 locales 目录。
	rootDir, err := config.GetProjectRootDir()
	if err != nil {
		panic(fmt.Errorf("get project root dir: %w", err))
	}
	path := filepath.Join(rootDir, "locales")

	// 加载翻译资源包。
	bundle := i18n.MustLoad(i18n.Config{
		Dir:         path,
		DefaultLang: lang,
		StrictMode:  true,
	})

	// 保存完整 Bundle 供 amis Engine 等组件使用。
	resource.I18nBundle = bundle

	// 根据当前语言创建 Localizer 实例。
	resource.Localizer = bundle.L(lang)
	resource.Logger.Info(ctx, "init i18n localizer", logit.StringField("lang", lang))
}

// initMetrics 初始化 Prometheus 指标采集。
//
// 注册的采集器：
//   - 默认采集器：Go 运行时指标（goroutine、内存、GC 等）
//   - DB 采集器：数据库连接池指标
//   - HTTP 采集器：HTTP 请求指标（QPS、延迟、状态码分布）
//
// 同时注册 HTTP 指标中间件，自动采集每个请求的指标。
func initMetrics(_ context.Context) {
	// 创建新的 Prometheus Registry（避免使用默认的全局 registry）。
	reg := prometheus.NewRegistry()

	// 注册默认采集器（Go 运行时指标）。
	metrics.MustRegisterCollectors(reg, metrics.DefaultCollectors()...)

	// 注册数据库连接池指标。
	dbMetrics := db.NewDBStats(resource.DB, db.WithDBName("cybertron"))

	// 注册 HTTP 指标采集器。
	httpMetrics := provider.NewHTTP()

	// 将采集器注册到 Registry。
	metrics.MustRegister(reg, httpMetrics, dbMetrics)

	// 注册 HTTP 指标中间件（自动采集每个请求的指标）。
	resource.GinMiddlewares = append(resource.GinMiddlewares, httpMetrics.GinMiddleware())

	resource.Metrics = reg
}

// initCachedStore 初始化配置热加载缓存。
//
// 缓存通过 GORM Loader 从数据库加载配置，支持：
//   - 增量刷新：每 1 分钟检查数据库中的变更
//   - 全量刷新：每 24 小时全量重载
//
// 适用于业务开关、灰度配置等需要动态调整的配置项。
func initCachedStore(ctx context.Context) {
	// 默认新项目不开启,这里仅提供默认代码。
	//var err error
	//resource.CfgstoreCached, err = cfgstore.New(ctx, cfgstore.NewGormLoader(resource.DB), resource.Logger,
	//	cfgstore.WithRefreshInterval(1*time.Minute),
	//	cfgstore.WithFullRefreshInterval(24*time.Hour))
	//if err != nil {
	//	panic(fmt.Errorf("failed to init config store, %w", err))
	//}
}

// initAccepting 将服务标记为"可接受请求"状态。
//
// 在 K8s Readiness Probe 中使用：只有当 Accepting 为 true 时才返回健康状态。
// 也可以用于优雅关闭时将 Accepting 设为 false，让负载均衡器停止向该实例转发流量。
func initAccepting(_ context.Context) {
	resource.Accepting.Store(true)
}

func initTrace(ctx context.Context) {

	if !resource.AppConfig.Get().Trace.Enable {
		return
	}

	otlpEndpoint := resource.AppConfig.Get().Trace.GRPCOtlpEndpoint

	conn, err := grpc.NewClient(otlpEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Errorf("failed to init trace client, %w. grpc:%s", err, otlpEndpoint))
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		panic(fmt.Errorf("failed to init trace client, %w. grpc:%s", err, otlpEndpoint))
	}

	res, _ := itelresource.New(context.Background(),
		itelresource.WithAttributes(
			semconv.ServiceNameKey.String(resource.AppName),
			attribute.String("env", resource.Env),
			attribute.String("pod_name", resource.PodName),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	resource.Trace = tp.Tracer(resource.AppName)
	resource.ShutDownFuncs = append(resource.ShutDownFuncs, func() {
		_ = tp.Shutdown(ctx)
	})
}
