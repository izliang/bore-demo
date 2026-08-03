// Package role 是角色业务逻辑层（Service Layer）。
//
// 职责：
//   - 角色 CRUD（role_key 创建后不可修改）
//   - 角色-菜单权限分配（ReplaceRoleMenus）
//   - 保护规则：内置角色（super_admin / admin）不可改删；被管理员绑定的角色不可删除
package role

import (
	"errors"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore/e"
	"gorm.io/gorm"
)

// 业务错误码（模块 role）。
var (
	ErrNotFound  = e.New("R0001", "角色不存在").Build()
	ErrKeyTaken  = e.New("R0002", "角色标识已存在").Build()
	ErrBuiltin   = e.New("R0003", "内置角色不允许修改或删除").Build()
	ErrRoleInUse = e.New("R0004", "角色已被管理员绑定，无法删除").Build()
)

// builtinRoleKeys 内置角色标识集合（受保护，不可改删）。
var builtinRoleKeys = map[string]bool{
	model.RoleKeySuperAdmin: true,
	model.RoleKeyAdmin:      true,
}

// Service 定义角色业务逻辑的契约接口。
type Service interface {
	// GetRole 查询角色详情（含关联菜单 ID）。
	GetRole(id uint64) (*model.Role, e.ServiceError)
	// ListRoles 关键字分页查询。
	ListRoles(param *model.RoleQueryParam) ([]*model.Role, int64, e.ServiceError)
	// ListOptions 查询全部启用角色（下拉选择用，不含菜单关联）。
	ListOptions() ([]*model.Role, e.ServiceError)
	// CreateRole 新增角色并分配菜单权限。
	CreateRole(param *model.CreateRoleParam) (*model.Role, e.ServiceError)
	// UpdateRole 更新角色并重分配菜单权限。
	UpdateRole(param *model.UpdateRoleParam) (*model.Role, e.ServiceError)
	// DeleteRole 删除角色（内置/被引用保护）。
	DeleteRole(id uint64) e.ServiceError
	// CountRoles 统计角色总数（仪表盘用）。
	CountRoles() (int64, e.ServiceError)
}

// service 是 Service 接口的私有实现。
type service struct {
	roleDao dao.RoleDao
}

// New 创建角色 Service 实例。
func New(roleDao dao.RoleDao) Service {
	return &service{roleDao: roleDao}
}

func (s *service) GetRole(id uint64) (*model.Role, e.ServiceError) {
	r, err := s.roleDao.GetRole(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	menuIDs, err := s.roleDao.GetMenuIDsByRoleID(r.ID)
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	role := toDomain(r)
	role.MenuIDs = menuIDs
	return role, nil
}

func (s *service) ListRoles(param *model.RoleQueryParam) ([]*model.Role, int64, e.ServiceError) {
	page, perPage := normalizePage(param.Page, param.PerPage)
	list, total, err := s.roleDao.ListRoles(param.Keyword, (page-1)*perPage, perPage)
	if err != nil {
		return nil, 0, e.WrapInternal(err)
	}
	out := make([]*model.Role, 0, len(list))
	for _, r := range list {
		out = append(out, toDomain(r))
	}
	return out, total, nil
}

func (s *service) ListOptions() ([]*model.Role, e.ServiceError) {
	list, err := s.roleDao.ListEnabledRoles()
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	out := make([]*model.Role, 0, len(list))
	for _, r := range list {
		out = append(out, toDomain(r))
	}
	return out, nil
}

func (s *service) CreateRole(param *model.CreateRoleParam) (*model.Role, e.ServiceError) {
	if err := s.checkKeyAvailable(param.RoleKey, 0); err != nil {
		return nil, err
	}
	r := &dao.Role{
		RoleName:  param.RoleName,
		RoleKey:   param.RoleKey,
		RoleSort:  param.RoleSort,
		DataScope: "1",
		Status:    param.Status,
		Remark:    param.Remark,
	}
	if err := s.roleDao.CreateRole(r); err != nil {
		return nil, e.WrapInternal(err)
	}
	if err := s.roleDao.ReplaceRoleMenus(r.ID, param.MenuIDs); err != nil {
		return nil, e.WrapInternal(err)
	}
	role := toDomain(r)
	role.MenuIDs = param.MenuIDs
	return role, nil
}

func (s *service) UpdateRole(param *model.UpdateRoleParam) (*model.Role, e.ServiceError) {
	r, err := s.roleDao.GetRole(param.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	if builtinRoleKeys[r.RoleKey] {
		return nil, ErrBuiltin
	}

	r.RoleName = param.RoleName
	r.RoleSort = param.RoleSort
	r.Status = param.Status
	r.Remark = param.Remark
	if err := s.roleDao.UpdateRole(r); err != nil {
		return nil, e.WrapInternal(err)
	}
	if err := s.roleDao.ReplaceRoleMenus(r.ID, param.MenuIDs); err != nil {
		return nil, e.WrapInternal(err)
	}

	role := toDomain(r)
	role.MenuIDs = param.MenuIDs
	return role, nil
}

func (s *service) DeleteRole(id uint64) e.ServiceError {
	r, err := s.roleDao.GetRole(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return e.WrapInternal(err)
	}
	if builtinRoleKeys[r.RoleKey] {
		return ErrBuiltin
	}
	refs, err := s.roleDao.CountAdminRefs(r.ID)
	if err != nil {
		return e.WrapInternal(err)
	}
	if refs > 0 {
		return ErrRoleInUse
	}
	if err := s.roleDao.DeleteRole(r.ID); err != nil {
		return e.WrapInternal(err)
	}
	return nil
}

func (s *service) CountRoles() (int64, e.ServiceError) {
	n, err := s.roleDao.CountRoles()
	if err != nil {
		return 0, e.WrapInternal(err)
	}
	return n, nil
}

// checkKeyAvailable 校验角色标识唯一（excludeID 用于更新时排除自身）。
func (s *service) checkKeyAvailable(key string, excludeID uint64) e.ServiceError {
	existing, err := s.roleDao.GetRoleByKey(key)
	if err == nil && existing.ID != excludeID {
		return ErrKeyTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return e.WrapInternal(err)
	}
	return nil
}

// toDomain 将角色 DAO 实体转换为领域模型。
func toDomain(r *dao.Role) *model.Role {
	return &model.Role{
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

// normalizePage 归一化分页参数。
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
