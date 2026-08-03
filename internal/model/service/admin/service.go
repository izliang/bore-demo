// Package admin 是管理员业务逻辑层（Service Layer）。
//
// 职责：
//   - 登录校验（bcrypt + 状态检查）
//   - 管理员 CRUD 与角色分配
//   - 保护规则：不能删除自己、至少保留一个启用的超级管理员
//
// 遵循仓库约定：面向 dao.AdminDao / dao.RoleDao 接口编程，便于 mock 测试。
package admin

import (
	"errors"
	"time"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore-demo/internal/security"
	"github.com/izliang/bore/e"
	"gorm.io/gorm"
)

// 业务错误码（模块 admin）。
var (
	ErrNotFound         = e.New("A0001", "管理员不存在").Build()
	ErrUsernameTaken    = e.New("A0002", "用户名已存在").Build()
	ErrInvalidCreds     = e.New("A0003", "用户名或密码错误").Build()
	ErrDisabled         = e.New("A0004", "账号已禁用").Build()
	ErrDeleteSelf       = e.New("A0005", "不能删除当前登录账号").Build()
	ErrLastSuperAdmin   = e.New("A0006", "至少保留一个启用的超级管理员").Build()
	ErrPasswordTooShort = e.New("A0010", "密码长度不能少于 6 位").Build()
)

// minPasswordLen 密码最小长度。
const minPasswordLen = 6

// Service 定义管理员业务逻辑的契约接口。
type Service interface {
	// Login 校验用户名密码并记录登录时间，成功返回管理员领域模型。
	Login(username, password string) (*model.Admin, e.ServiceError)
	// GetAdmin 查询管理员详情（含角色）。
	GetAdmin(id uint64) (*model.Admin, e.ServiceError)
	// ListAdmins 关键字分页查询（含角色）。
	ListAdmins(param *model.AdminQueryParam) ([]*model.Admin, int64, e.ServiceError)
	// CreateAdmin 新增管理员（bcrypt 加密 + 角色分配）。
	CreateAdmin(param *model.CreateAdminParam) (*model.Admin, e.ServiceError)
	// UpdateAdmin 更新管理员（密码为空表示不修改，支持角色重分配）。
	UpdateAdmin(param *model.UpdateAdminParam) (*model.Admin, e.ServiceError)
	// DeleteAdmin 删除管理员，带自身/最后超管保护。
	DeleteAdmin(id, operatorID uint64) e.ServiceError
	// BatchDeleteAdmins 批量删除，任一失败即返回错误。
	BatchDeleteAdmins(ids []uint64, operatorID uint64) e.ServiceError
	// CountAdmins 统计管理员总数（仪表盘用）。
	CountAdmins() (int64, e.ServiceError)
}

// service 是 Service 接口的私有实现。
type service struct {
	adminDao dao.AdminDao
	roleDao  dao.RoleDao
}

// New 创建管理员 Service 实例。
func New(adminDao dao.AdminDao, roleDao dao.RoleDao) Service {
	return &service{adminDao: adminDao, roleDao: roleDao}
}

func (s *service) Login(username, password string) (*model.Admin, e.ServiceError) {
	a, err := s.adminDao.GetAdminByUsername(username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 统一返回"凭证错误"，避免暴露用户名是否存在。
		return nil, ErrInvalidCreds
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	if !security.CheckPassword(a.PasswordHash, password) {
		return nil, ErrInvalidCreds
	}
	if a.Status != int8(model.AdminStatusEnabled) {
		return nil, ErrDisabled
	}

	now := time.Now()
	if err := s.adminDao.TouchLastLogin(a.ID, now); err != nil {
		return nil, e.WrapInternal(err)
	}
	a.LastLoginAt = &now
	return toDomain(a, nil), nil
}

func (s *service) GetAdmin(id uint64) (*model.Admin, e.ServiceError) {
	a, err := s.adminDao.GetAdmin(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	roles, svcErr := s.loadRoles(a.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toDomain(a, roles), nil
}

func (s *service) ListAdmins(param *model.AdminQueryParam) ([]*model.Admin, int64, e.ServiceError) {
	page, perPage := normalizePage(param.Page, param.PerPage)
	list, total, err := s.adminDao.ListAdmins(param.Keyword, (page-1)*perPage, perPage)
	if err != nil {
		return nil, 0, e.WrapInternal(err)
	}

	out := make([]*model.Admin, 0, len(list))
	for _, a := range list {
		roles, svcErr := s.loadRoles(a.ID)
		if svcErr != nil {
			return nil, 0, svcErr
		}
		out = append(out, toDomain(a, roles))
	}
	return out, total, nil
}

func (s *service) CreateAdmin(param *model.CreateAdminParam) (*model.Admin, e.ServiceError) {
	if len(param.Password) < minPasswordLen {
		return nil, ErrPasswordTooShort
	}
	if err := s.checkUsernameAvailable(param.Username, 0); err != nil {
		return nil, err
	}

	hash, err := security.HashPassword(param.Password)
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	a := &dao.Admin{
		Username:     param.Username,
		PasswordHash: hash,
		Nickname:     param.Nickname,
		Avatar:       param.Avatar,
		Email:        param.Email,
		Phone:        param.Phone,
		Status:       int8(param.Status),
	}
	if err := s.adminDao.CreateAdmin(a); err != nil {
		return nil, e.WrapInternal(err)
	}
	if err := s.adminDao.ReplaceAdminRoles(a.ID, param.RoleIDs); err != nil {
		return nil, e.WrapInternal(err)
	}

	roles, svcErr := s.loadRoles(a.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toDomain(a, roles), nil
}

func (s *service) UpdateAdmin(param *model.UpdateAdminParam) (*model.Admin, e.ServiceError) {
	a, err := s.adminDao.GetAdmin(param.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	if param.Username != a.Username {
		if err := s.checkUsernameAvailable(param.Username, param.ID); err != nil {
			return nil, err
		}
	}

	// 禁用前保护：目标账号仍是超管且是最后一个启用超管时禁止禁用。
	if model.AdminStatus(a.Status) == model.AdminStatusEnabled && param.Status == model.AdminStatusDisabled {
		super, svcErr := s.isSuperAdmin(a.ID)
		if svcErr != nil {
			return nil, svcErr
		}
		if super {
			if err := s.ensureOtherSuperAdmin(a.ID); err != nil {
				return nil, err
			}
		}
	}

	if param.Password != "" {
		if len(param.Password) < minPasswordLen {
			return nil, ErrPasswordTooShort
		}
		hash, err := security.HashPassword(param.Password)
		if err != nil {
			return nil, e.WrapInternal(err)
		}
		a.PasswordHash = hash
	}

	a.Username = param.Username
	a.Nickname = param.Nickname
	a.Avatar = param.Avatar
	a.Email = param.Email
	a.Phone = param.Phone
	a.Status = int8(param.Status)
	if err := s.adminDao.UpdateAdmin(a); err != nil {
		return nil, e.WrapInternal(err)
	}
	if err := s.adminDao.ReplaceAdminRoles(a.ID, param.RoleIDs); err != nil {
		return nil, e.WrapInternal(err)
	}

	roles, svcErr := s.loadRoles(a.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toDomain(a, roles), nil
}

func (s *service) DeleteAdmin(id, operatorID uint64) e.ServiceError {
	if id == operatorID {
		return ErrDeleteSelf
	}
	a, err := s.adminDao.GetAdmin(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return e.WrapInternal(err)
	}

	super, svcErr := s.isSuperAdmin(a.ID)
	if svcErr != nil {
		return svcErr
	}
	if super {
		if err := s.ensureOtherSuperAdmin(a.ID); err != nil {
			return err
		}
	}
	if err := s.adminDao.DeleteAdmin(a.ID); err != nil {
		return e.WrapInternal(err)
	}
	return nil
}

func (s *service) BatchDeleteAdmins(ids []uint64, operatorID uint64) e.ServiceError {
	for _, id := range ids {
		if err := s.DeleteAdmin(id, operatorID); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) CountAdmins() (int64, e.ServiceError) {
	n, err := s.adminDao.CountAdmins()
	if err != nil {
		return 0, e.WrapInternal(err)
	}
	return n, nil
}

// ============================================================
// 私有辅助
// ============================================================

// checkUsernameAvailable 校验用户名唯一（excludeID 用于更新时排除自身）。
func (s *service) checkUsernameAvailable(username string, excludeID uint64) e.ServiceError {
	existing, err := s.adminDao.GetAdminByUsername(username)
	if err == nil && existing.ID != excludeID {
		return ErrUsernameTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return e.WrapInternal(err)
	}
	return nil
}

// isSuperAdmin 判断管理员是否绑定启用的 super_admin 角色。
func (s *service) isSuperAdmin(adminID uint64) (bool, e.ServiceError) {
	keys, err := s.roleDao.GetRoleKeysByAdminID(adminID)
	if err != nil {
		return false, e.WrapInternal(err)
	}
	for _, k := range keys {
		if k == model.RoleKeySuperAdmin {
			return true, nil
		}
	}
	return false, nil
}

// ensureOtherSuperAdmin 确保除 excludeID 外仍存在启用超管，否则返回保护错误。
func (s *service) ensureOtherSuperAdmin(excludeID uint64) e.ServiceError {
	n, err := s.adminDao.CountOtherSuperAdmins(excludeID)
	if err != nil {
		return e.WrapInternal(err)
	}
	if n == 0 {
		return ErrLastSuperAdmin
	}
	return nil
}

// loadRoles 加载管理员绑定的角色（跳过已删除的角色）。
func (s *service) loadRoles(adminID uint64) ([]model.Role, e.ServiceError) {
	roleIDs, err := s.adminDao.GetRoleIDsByAdminID(adminID)
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	roles := make([]model.Role, 0, len(roleIDs))
	for _, rid := range roleIDs {
		r, err := s.roleDao.GetRole(rid)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, e.WrapInternal(err)
		}
		roles = append(roles, toDomainRole(r))
	}
	return roles, nil
}

// toDomain 将 DAO 实体转换为领域模型。
func toDomain(a *dao.Admin, roles []model.Role) *model.Admin {
	return &model.Admin{
		ID:           a.ID,
		Username:     a.Username,
		PasswordHash: a.PasswordHash,
		Nickname:     a.Nickname,
		Avatar:       a.Avatar,
		Email:        a.Email,
		Phone:        a.Phone,
		Status:       model.AdminStatus(a.Status),
		LastLoginAt:  a.LastLoginAt,
		Roles:        roles,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// toDomainRole 将角色 DAO 实体转换为领域模型。
func toDomainRole(r *dao.Role) model.Role {
	return model.Role{
		ID:        r.ID,
		RoleName:  r.RoleName,
		RoleKey:   r.RoleKey,
		RoleSort:  r.RoleSort,
		DataScope: r.DataScope,
		Status:    r.Status,
		Remark:    r.Remark,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// normalizePage 归一化分页参数（非法值回退到默认）。
func normalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
