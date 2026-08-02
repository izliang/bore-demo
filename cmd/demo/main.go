// Package main 是程序的入口点。
//
// 启动流程：
//  1. bootstrap.Init() —— 按顺序初始化所有基础设施组件
//     （环境变量 → 配置 → 日志 → 数据库 → 消息队列 → 国际化 → 指标 → 缓存 → 就绪标记）
//  2. [TODO] 启动 HTTP Server 并注册路由
//  3. [TODO] 监听系统退出信号（SIGINT/SIGTERM）
//  4. bootstrap.Shutdown() —— 执行优雅关闭
//
// 分层架构概览（自上而下）：
//
//	cmd/demo          → 程序入口 + 启动编排
//	internal/bootstrap → 基础设施初始化（DB、MQ、Logger、Metrics 等）
//	internal/api/router → HTTP 路由注册 + 中间件挂载
//	internal/api/controller → 控制器层：DTO↔Domain 映射 + 调用 Service
//	internal/model/service → 业务逻辑层：纯领域逻辑，不依赖 HTTP
//	internal/model/dao → 数据访问层：GORM 操作，面向接口
//	internal/resource → 全局单例资源池（DB、Logger、Config、Queue 等）
//	internal/model → 领域模型定义（入参、实体，无 ORM 标签）
//	internal/config → 配置结构体定义（映射 YAML 配置）
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore-demo/internal/api/router"
	"github.com/izliang/bore-demo/internal/bootstrap"
	"github.com/izliang/bore-demo/internal/resource"
)

func main() {
	// 创建后台 context，用于贯穿整个应用生命周期。
	ctx := context.Background()

	// ============================================================
	// 阶段一：基础设施初始化
	// 按依赖顺序初始化：环境变量 → 配置 → 日志 → DB → MQ → i18n → Metrics → Cache
	// 任何初始化失败都会 panic，因为缺少基础设施服务无法运行。
	// ============================================================
	bootstrap.Init(ctx)

	// ============================================================
	// 阶段二：启动 HTTP Server + 注册路由
	ginEngine := gin.New()
	router.SetupRoutes(ginEngine)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", resource.AppConfig.Get().Service.Port),
		Handler: ginEngine,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	// ============================================================

	// ============================================================
	// 阶段三：监听操作系统退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	// ============================================================

	// ============================================================
	// 阶段四：优雅关闭
	// 释放连接池、刷新缓冲区、等待进行中的请求完成等。
	// ============================================================
	bootstrap.Shutdown(ctx)
}
