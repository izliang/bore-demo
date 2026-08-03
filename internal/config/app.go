// Package config 定义应用配置的结构体。
//
// 配置结构体通过 yaml tag 与 conf/ 目录下的配置文件字段一一对应。
// 使用 bore/config.Cfg[T] 泛型容器加载，支持模板变量渲染。
//
// 配置加载流程：
//
//	config_dev.yaml → config.LoadRootFile() → AppConfig.Get() → 业务代码使用
//
// 新项目复刻指南：
//   - 按需增减字段，yaml tag 必须与配置文件中的 key 完全一致
//   - 敏感信息（密码、密钥）不要硬编码在配置文件中，应从环境变量或 K8s Secret 注入
//   - 新增配置块时，在 AppConfig 中添加新的顶级字段
package config

import "github.com/izliang/bore/logit"

// AppConfig 应用的顶级配置结构体。
// 对应 conf/config_*.yaml 的根节点。
type AppConfig struct {
	// Service 服务基础配置（名称、端口、日志级别）。
	Service Service `yaml:"service"`

	// RabbitMQ 消息队列连接配置。
	// 如果项目不需要 MQ，删除此字段。
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`

	// Database 数据库连接配置。
	Database Database `yaml:"database"`

	// Trace trace信息
	Trace Trace `yaml:"trace"`
}

// Service 服务基础配置。
type Service struct {
	// Name 服务名称，用于日志文件命名和指标标签。
	Name string `yaml:"name"`

	// Port HTTP 服务监听端口。
	Port int `yaml:"port"`

	// LogLevel 日志级别（debug / info / warn / error）。
	LogLevel logit.Level `yaml:"log_level"`
}

// RabbitMQConfig RabbitMQ 连接参数。
//
// 新项目复刻指南：
//   - 如果不需要 MQ，删除整个结构体
//   - 如果使用 Kafka，替换为对应的 Kafka 配置结构体
type RabbitMQConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Vhost        string `yaml:"vhost"`
	DeclareVHost bool   `yaml:"declare_vhost"`
	SSL          bool   `yaml:"ssl"`
}

// Database 数据库连接配置。
type Database struct {
	// DSN 数据库连接字符串（格式：user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True）
	DSN string `yaml:"dsn"`
}

type Trace struct {
	Enable           bool   `yaml:"enable"`
	GRPCOtlpEndpoint string `yaml:"grpc_otlp_endpoint"`
}
