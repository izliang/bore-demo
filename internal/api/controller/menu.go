package controller

import (
	"context"

	adminapi "github.com/izliang/bore-demo/internal/api/admin"
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	menusvc "github.com/izliang/bore-demo/internal/model/service/menu"
	v1 "github.com/izliang/bore-demo/internal/resource/api/v1"
	"github.com/izliang/bore/e"
)

// MenuTree 处理 GET /api/v1/menu/tree（权限：menu:list / role:assign / role:edit 任一）。
// 返回完整菜单树（data 为数组），供菜单管理页与角色权限分配树使用。
func MenuTree(ctx context.Context, _ *struct{}) (*v1.MenuTreeResponse, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "menu:list", "role:assign", "role:edit"); err != nil {
		return nil, err
	}

	tree, svcErr := menuSvc(ctx).Tree()
	if svcErr != nil {
		return nil, svcErr
	}
	nodes := v1.MenuTreeResponse(toMenuNodes(tree))
	return &nodes, nil
}

// GetMenu 处理 GET /api/v1/menu/get（权限：menu:query）。
func GetMenu(ctx context.Context, req *v1.GetMenuRequest) (*v1.MenuNode, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "menu:query"); err != nil {
		return nil, err
	}

	m, svcErr := menuSvc(ctx).GetMenu(req.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toMenuNode(m), nil
}

// CreateMenu 处理 POST /api/v1/menu/create（权限：menu:add）。
func CreateMenu(ctx context.Context, req *v1.CreateMenuRequest) (*v1.MenuNode, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "menu:add"); err != nil {
		return nil, err
	}

	m, svcErr := menuSvc(ctx).CreateMenu(&model.CreateMenuParam{
		ParentID: req.ParentID,
		MenuName: req.MenuName,
		MenuType: req.MenuType,
		Path:     req.Path,
		Perms:    req.Perms,
		Icon:     req.Icon,
		OrderNum: req.OrderNum,
		Visible:  req.Visible,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidateAllPerms()
	return toMenuNode(m), nil
}

// UpdateMenu 处理 PUT /api/v1/menu/update（权限：menu:edit）。
func UpdateMenu(ctx context.Context, req *v1.UpdateMenuRequest) (*v1.MenuNode, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "menu:edit"); err != nil {
		return nil, err
	}

	m, svcErr := menuSvc(ctx).UpdateMenu(&model.UpdateMenuParam{
		ID:       req.ID,
		ParentID: req.ParentID,
		MenuName: req.MenuName,
		MenuType: req.MenuType,
		Path:     req.Path,
		Perms:    req.Perms,
		Icon:     req.Icon,
		OrderNum: req.OrderNum,
		Visible:  req.Visible,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidateAllPerms()
	return toMenuNode(m), nil
}

// DeleteMenu 处理 DELETE /api/v1/menu/delete（权限：menu:remove）。
func DeleteMenu(ctx context.Context, req *v1.DeleteMenuRequest) (*struct{}, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "menu:remove"); err != nil {
		return nil, err
	}

	if svcErr := menuSvc(ctx).DeleteMenu(req.ID); svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidateAllPerms()
	return &struct{}{}, nil
}

// ============================================================
// 私有辅助
// ============================================================

// menuSvc 构建菜单 Service。
func menuSvc(ctx context.Context) menusvc.Service {
	return menusvc.New(dao.NewMenuDao(newATX(ctx)))
}

// toMenuNodes 将领域树转换为响应树。
func toMenuNodes(list []*model.Menu) []*v1.MenuNode {
	nodes := make([]*v1.MenuNode, 0, len(list))
	for _, m := range list {
		nodes = append(nodes, toMenuNode(m))
	}
	return nodes
}

// toMenuNode 将单个领域节点转换为响应节点（含子树）。
func toMenuNode(m *model.Menu) *v1.MenuNode {
	node := &v1.MenuNode{
		ID:       m.ID,
		ParentID: m.ParentID,
		MenuName: m.MenuName,
		MenuType: m.MenuType,
		Path:     m.Path,
		Perms:    m.Perms,
		Icon:     m.Icon,
		OrderNum: m.OrderNum,
		Visible:  m.Visible,
	}
	if len(m.Children) > 0 {
		node.Children = toMenuNodes(m.Children)
	}
	return node
}
