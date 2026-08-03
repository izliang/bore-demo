// Package auth 是鉴权相关业务逻辑层（Service Layer）。
//
// 职责：
//   - 聚合管理员的角色标识与权限标识（供权限中间件与 /auth/info 使用）
//   - 超级管理员返回全量权限标识
package auth

import (
	"errors"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore/e"
	"gorm.io/gorm"
)

// ErrUnauthorized 登录态无效（管理员不存在或已禁用）。
var ErrUnauthorized = e.New("A0007", "登录状态无效或已过期").Build()

// Service 定义鉴权业务逻辑的契约接口。
type Service interface {
	// GetPerms 聚合管理员（启用角色）的角色标识与权限标识。
	// 超级管理员（super_admin）返回全量权限标识。
	GetPerms(adminID uint64) (*model.AdminPerms, e.ServiceError)
}

// service 是 Service 接口的私有实现。
type service struct {
	adminDao dao.AdminDao
	roleDao  dao.RoleDao
	menuDao  dao.MenuDao
}

// New 创建鉴权 Service 实例。
func New(adminDao dao.AdminDao, roleDao dao.RoleDao, menuDao dao.MenuDao) Service {
	return &service{adminDao: adminDao, roleDao: roleDao, menuDao: menuDao}
}

func (s *service) GetPerms(adminID uint64) (*model.AdminPerms, e.ServiceError) {
	a, err := s.adminDao.GetAdmin(adminID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	if a.Status != int8(model.AdminStatusEnabled) {
		return nil, ErrUnauthorized
	}

	roleKeys, err := s.roleDao.GetRoleKeysByAdminID(adminID)
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	isSuper := false
	for _, k := range roleKeys {
		if k == model.RoleKeySuperAdmin {
			isSuper = true
			break
		}
	}

	var perms []string
	if isSuper {
		// 超管全量权限，保证前端按钮/导航全部可见。
		perms, err = s.menuDao.ListAllPerms()
	} else {
		perms, err = s.menuDao.GetPermsByAdminID(adminID)
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}

	return &model.AdminPerms{
		Roles:   roleKeys,
		Perms:   perms,
		IsSuper: isSuper,
	}, nil
}
