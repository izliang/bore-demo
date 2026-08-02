package bootstrap

import (
	"context"
	"time"

	"github.com/izliang/bore-demo/internal/resource"
)

// Shutdown 执行应用的优雅关闭流程。
//
// 典型关闭步骤（按顺序）：
//  1. 将 Accepting 标记设为 false，阻止新的请求进入（通过 K8s Readiness Probe）
//  2. 等待进行中的请求处理完成（设置超时上限）
//  3. 关闭 HTTP Server（先停止接收，再等待现有连接完成）
//  4. 关闭消息队列消费者（停止拉取新消息）
//  5. 关闭数据库连接池
//  6. 刷新日志缓冲区并关闭日志
//  7. 清理临时资源
//
// 每个步骤的超时时间应可配置，避免无限等待导致 K8s Pod 强杀。
//
// 新项目复刻指南：
//   - 根据实际使用的中间件补充对应的关闭逻辑
//   - 使用 resource.Accepting.Store(false) 提前拒绝新流量
func Shutdown(ctx context.Context) {
	// ============================================================
	// 实现优雅关闭逻辑
	//
	//
	// 1. 标记不再接受新请求
	resource.Accepting.Store(false)

	// 2. 等待一段时间让负载均衡器感知状态变化
	time.Sleep(5 * time.Second)

	// 3. 关闭全局函数
	for _, downFunc := range resource.ShutDownFuncs {
		downFunc()
	}
}
