// Package admin 提供 amis 管理后台的装配与 RBAC 运行时能力。
//
// 包含：
//   - amis Engine 装配（页面注册、登录鉴权、受保护路由）
//   - 登录回调（Admin Service 校验）
//   - 权限会话缓存与请求上下文（RequireLogin / CheckPerm）
package admin

import (
	"sync"
	"time"

	"github.com/izliang/bore-demo/internal/model"
)

// sessionTTL 权限缓存的默认有效期。
// 角色/菜单变更会主动失效（InvalidateAll），TTL 仅作兜底。
const sessionTTL = 5 * time.Minute

// sessionEntry 单个管理员的权限缓存条目。
type sessionEntry struct {
	roles     []string
	perms     []string
	isSuper   bool
	expiresAt time.Time
}

func (e *sessionEntry) expired() bool {
	return time.Now().After(e.expiresAt)
}

// Session 管理员权限会话缓存（进程内内存实现）。
//
// demo 场景使用内存缓存即可；生产可替换为 bore cache（Redis provider）。
type Session struct {
	mu      sync.Mutex
	entries map[uint64]*sessionEntry
	ttl     time.Duration
}

// NewSession 创建权限会话缓存。
func NewSession(ttl time.Duration) *Session {
	if ttl <= 0 {
		ttl = sessionTTL
	}
	return &Session{
		entries: make(map[uint64]*sessionEntry),
		ttl:     ttl,
	}
}

// Get 返回管理员的权限缓存，未命中或已过期时返回 false。
func (s *Session) Get(adminID uint64) (model.AdminPerms, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.entries[adminID]
	if !ok || e.expired() {
		delete(s.entries, adminID)
		return model.AdminPerms{}, false
	}
	return model.AdminPerms{Roles: e.roles, Perms: e.perms, IsSuper: e.isSuper}, true
}

// Set 写入管理员的权限缓存。
func (s *Session) Set(adminID uint64, p model.AdminPerms) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[adminID] = &sessionEntry{
		roles:     p.Roles,
		perms:     p.Perms,
		isSuper:   p.IsSuper,
		expiresAt: time.Now().Add(s.ttl),
	}
}

// Invalidate 使单个管理员的缓存失效（管理员变更后调用）。
func (s *Session) Invalidate(adminID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, adminID)
}

// InvalidateAll 使全部缓存失效（角色/菜单权限变更后调用）。
func (s *Session) InvalidateAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[uint64]*sessionEntry)
}

// permSession 是全局权限会话缓存，由 Setup 初始化。
var permSession = NewSession(sessionTTL)

// InvalidatePerms 使指定管理员权限缓存失效。
// 供 controller 在管理员变更后调用。
func InvalidatePerms(adminID uint64) {
	permSession.Invalidate(adminID)
}

// InvalidateAllPerms 使全部权限缓存失效。
// 供 controller 在角色/菜单权限变更后调用。
func InvalidateAllPerms() {
	permSession.InvalidateAll()
}
