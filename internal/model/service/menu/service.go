// Package menu 是菜单/按钮权限业务逻辑层（Service Layer）。
//
// 职责：
//   - 菜单树组装（parent_id + order_num）
//   - 菜单/按钮 CRUD
//   - 保护规则：存在子菜单时禁止删除
//
// 对应若依 sys_menu 的管理能力（M 目录 / C 菜单 / F 按钮）。
package menu

import (
	"errors"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore/e"
	"gorm.io/gorm"
)

// 业务错误码（模块 menu）。
var (
	ErrNotFound       = e.New("M0001", "菜单不存在").Build()
	ErrHasChildren    = e.New("M0002", "存在子菜单，无法删除").Build()
	ErrInvalidType    = e.New("M0003", "菜单类型不合法（M/C/F）").Build()
	ErrParentNotFound = e.New("M0004", "父菜单不存在").Build()
)

// validMenuTypes 合法的菜单类型集合。
var validMenuTypes = map[string]bool{
	model.MenuTypeDir:    true,
	model.MenuTypeMenu:   true,
	model.MenuTypeButton: true,
}

// Service 定义菜单业务逻辑的契约接口。
type Service interface {
	// Tree 返回完整菜单树（角色分配权限、菜单管理页用）。
	Tree() ([]*model.Menu, e.ServiceError)
	// ListMenus 返回扁平菜单列表。
	ListMenus() ([]*model.Menu, e.ServiceError)
	// GetMenu 查询菜单详情。
	GetMenu(id uint64) (*model.Menu, e.ServiceError)
	// CreateMenu 新增菜单/按钮。
	CreateMenu(param *model.CreateMenuParam) (*model.Menu, e.ServiceError)
	// UpdateMenu 更新菜单/按钮。
	UpdateMenu(param *model.UpdateMenuParam) (*model.Menu, e.ServiceError)
	// DeleteMenu 删除菜单/按钮（含子节点保护）。
	DeleteMenu(id uint64) e.ServiceError
}

// service 是 Service 接口的私有实现。
type service struct {
	menuDao dao.MenuDao
}

// New 创建菜单 Service 实例。
func New(menuDao dao.MenuDao) Service {
	return &service{menuDao: menuDao}
}

func (s *service) Tree() ([]*model.Menu, e.ServiceError) {
	list, err := s.menuDao.ListMenus()
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	return buildTree(list), nil
}

func (s *service) ListMenus() ([]*model.Menu, e.ServiceError) {
	list, err := s.menuDao.ListMenus()
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	out := make([]*model.Menu, 0, len(list))
	for _, m := range list {
		out = append(out, toDomain(m))
	}
	return out, nil
}

func (s *service) GetMenu(id uint64) (*model.Menu, e.ServiceError) {
	m, err := s.menuDao.GetMenu(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	return toDomain(m), nil
}

func (s *service) CreateMenu(param *model.CreateMenuParam) (*model.Menu, e.ServiceError) {
	if !validMenuTypes[param.MenuType] {
		return nil, ErrInvalidType
	}
	if param.ParentID != 0 {
		if err := s.checkParent(param.ParentID); err != nil {
			return nil, err
		}
	}
	m := &dao.Menu{
		ParentID: param.ParentID,
		MenuName: param.MenuName,
		MenuType: param.MenuType,
		Path:     param.Path,
		Perms:    param.Perms,
		Icon:     param.Icon,
		OrderNum: param.OrderNum,
		Visible:  param.Visible,
	}
	if err := s.menuDao.CreateMenu(m); err != nil {
		return nil, e.WrapInternal(err)
	}
	return toDomain(m), nil
}

func (s *service) UpdateMenu(param *model.UpdateMenuParam) (*model.Menu, e.ServiceError) {
	m, err := s.menuDao.GetMenu(param.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, e.WrapInternal(err)
	}
	if !validMenuTypes[param.MenuType] {
		return nil, ErrInvalidType
	}
	if param.ParentID != 0 {
		if err := s.checkParent(param.ParentID); err != nil {
			return nil, err
		}
		// 防止把节点挂到自己或自己的子树下造成环。
		if param.ParentID == m.ID || s.isDescendant(param.ParentID, m.ID) {
			return nil, e.New("M0005", "不能将菜单挂到自身或其子节点下").Build()
		}
	}

	m.ParentID = param.ParentID
	m.MenuName = param.MenuName
	m.MenuType = param.MenuType
	m.Path = param.Path
	m.Perms = param.Perms
	m.Icon = param.Icon
	m.OrderNum = param.OrderNum
	m.Visible = param.Visible
	if err := s.menuDao.UpdateMenu(m); err != nil {
		return nil, e.WrapInternal(err)
	}
	return toDomain(m), nil
}

func (s *service) DeleteMenu(id uint64) e.ServiceError {
	if _, err := s.menuDao.GetMenu(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return e.WrapInternal(err)
	}
	children, err := s.menuDao.ListChildren(id)
	if err != nil {
		return e.WrapInternal(err)
	}
	if len(children) > 0 {
		return ErrHasChildren
	}
	if err := s.menuDao.DeleteMenu(id); err != nil {
		return e.WrapInternal(err)
	}
	return nil
}

// checkParent 校验父菜单存在且不是按钮类型（按钮不能挂子节点）。
func (s *service) checkParent(parentID uint64) e.ServiceError {
	parent, err := s.menuDao.GetMenu(parentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrParentNotFound
	}
	if err != nil {
		return e.WrapInternal(err)
	}
	if parent.MenuType == model.MenuTypeButton {
		return e.New("M0006", "按钮节点不能作为父菜单").Build()
	}
	return nil
}

// isDescendant 判断 targetID 是否为 nodeID 的后代节点。
func (s *service) isDescendant(targetID, nodeID uint64) bool {
	all, err := s.menuDao.ListMenus()
	if err != nil {
		return false
	}
	children := make(map[uint64][]uint64)
	for _, m := range all {
		children[m.ParentID] = append(children[m.ParentID], m.ID)
	}
	stack := children[nodeID]
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == targetID {
			return true
		}
		stack = append(stack, children[cur]...)
	}
	return false
}

// buildTree 将扁平菜单列表组装为树（按 order_num 升序）。
func buildTree(list []*dao.Menu) []*model.Menu {
	nodes := make(map[uint64]*model.Menu, len(list))
	roots := make([]*model.Menu, 0)

	for _, m := range list {
		nodes[m.ID] = toDomain(m)
	}
	for _, m := range list {
		node := nodes[m.ID]
		if parent, ok := nodes[m.ParentID]; ok {
			parent.Children = append(parent.Children, node)
			continue
		}
		roots = append(roots, node)
	}
	return sortByOrder(roots)
}

// sortByOrder 按 order_num 升序排序（children 已在组装时保持 list 顺序，这里对根节点排序）。
func sortByOrder(list []*model.Menu) []*model.Menu {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].OrderNum < list[j-1].OrderNum; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
	return list
}

// toDomain 将菜单 DAO 实体转换为领域模型。
func toDomain(m *dao.Menu) *model.Menu {
	return &model.Menu{
		ID:        m.ID,
		ParentID:  m.ParentID,
		MenuName:  m.MenuName,
		MenuType:  m.MenuType,
		Path:      m.Path,
		Perms:     m.Perms,
		Icon:      m.Icon,
		OrderNum:  m.OrderNum,
		Visible:   m.Visible,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
