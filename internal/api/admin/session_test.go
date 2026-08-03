// Package admin 权限会话缓存单元测试。
package admin

import (
	"testing"
	"time"

	"github.com/izliang/bore-demo/internal/model"
)

func TestSession_SetAndGet(t *testing.T) {
	s := NewSession(time.Minute)

	perms := model.AdminPerms{
		Roles:   []string{"admin"},
		Perms:   []string{"user:list", "user:add"},
		IsSuper: false,
	}
	s.Set(1, perms)

	got, ok := s.Get(1)
	if !ok {
		t.Fatal("期望缓存命中，实际未命中")
	}
	if len(got.Roles) != 1 || len(got.Perms) != 2 || got.IsSuper {
		t.Errorf("缓存读取结果异常: %+v", got)
	}

	// 未写入的管理员 ID 未命中。
	if _, ok := s.Get(2); ok {
		t.Error("期望未命中的 ID，实际命中")
	}
}

func TestSession_Expired(t *testing.T) {
	s := NewSession(1) // 1ns，立即过期
	s.Set(1, model.AdminPerms{Roles: []string{"admin"}})

	if _, ok := s.Get(1); ok {
		t.Error("期望过期条目未命中，实际命中")
	}
	// 过期条目应被清理。
	if len(s.entries) != 0 {
		t.Errorf("期望过期条目被清理，实际残留 %d 条", len(s.entries))
	}
}

func TestSession_Invalidate(t *testing.T) {
	s := NewSession(time.Minute)
	s.Set(1, model.AdminPerms{})
	s.Set(2, model.AdminPerms{})

	s.Invalidate(1)
	if _, ok := s.Get(1); ok {
		t.Error("期望管理员 1 缓存失效，实际仍命中")
	}
	if _, ok := s.Get(2); !ok {
		t.Error("期望管理员 2 缓存保留，实际失效")
	}
}

func TestSession_InvalidateAll(t *testing.T) {
	s := NewSession(time.Minute)
	s.Set(1, model.AdminPerms{})
	s.Set(2, model.AdminPerms{})

	s.InvalidateAll()
	if len(s.entries) != 0 {
		t.Errorf("期望全部缓存清空，实际残留 %d 条", len(s.entries))
	}
}

func TestNewSession_DefaultTTL(t *testing.T) {
	s := NewSession(0)
	if s.ttl != sessionTTL {
		t.Errorf("期望使用默认 TTL %v，实际 %v", sessionTTL, s.ttl)
	}
}
