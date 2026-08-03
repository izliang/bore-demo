package controller

import (
	"context"

	adminapi "github.com/izliang/bore-demo/internal/api/admin"
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	rolesvc "github.com/izliang/bore-demo/internal/model/service/role"
	v1 "github.com/izliang/bore-demo/internal/resource/api/v1"
	"github.com/izliang/bore/e"
)

// ListRoles 处理 GET /api/v1/role/list（权限：role:list）。
func ListRoles(ctx context.Context, req *v1.ListRolesRequest) (*v1.RoleListResponse, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "role:list"); err != nil {
		return nil, err
	}

	list, total, svcErr := roleSvc(ctx).ListRoles(&model.RoleQueryParam{
		Keyword: req.Keyword,
		Page:    req.Page,
		PerPage: req.PerPage,
	})
	if svcErr != nil {
		return nil, svcErr
	}

	items := make([]*v1.RoleItem, 0, len(list))
	for _, r := range list {
		items = append(items, toRoleItem(r, nil))
	}
	return &v1.RoleListResponse{Items: items, Total: total}, nil
}

// GetRole 处理 GET /api/v1/role/get（权限：role:query）。
// 返回含 MenuIDs，供权限分配表单回显。
func GetRole(ctx context.Context, req *v1.GetRoleRequest) (*v1.RoleItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "role:query"); err != nil {
		return nil, err
	}

	r, svcErr := roleSvc(ctx).GetRole(req.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toRoleItem(r, r.MenuIDs), nil
}

// ListRoleOptions 处理 GET /api/v1/role/options（登录态即可访问）。
// 返回启用角色数组（data 为数组），供 amis select 组件直接渲染。
func ListRoleOptions(ctx context.Context, _ *struct{}) (*v1.RoleOptionsResponse, e.ServiceError) {
	list, svcErr := roleSvc(ctx).ListOptions()
	if svcErr != nil {
		return nil, svcErr
	}

	options := make(v1.RoleOptionsResponse, 0, len(list))
	for _, r := range list {
		options = append(options, &v1.RoleOption{
			ID:       r.ID,
			RoleName: r.RoleName,
			RoleKey:  r.RoleKey,
		})
	}
	return &options, nil
}

// CreateRole 处理 POST /api/v1/role/create（权限：role:add）。
func CreateRole(ctx context.Context, req *v1.CreateRoleRequest) (*v1.RoleItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "role:add"); err != nil {
		return nil, err
	}

	r, svcErr := roleSvc(ctx).CreateRole(&model.CreateRoleParam{
		RoleName: req.RoleName,
		RoleKey:  req.RoleKey,
		RoleSort: req.RoleSort,
		Status:   req.Status,
		Remark:   req.Remark,
		MenuIDs:  req.MenuIDs,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	// 新角色可能影响其他管理员权限（暂无绑定，无需失效；保留调用保持语义一致）。
	adminapi.InvalidateAllPerms()
	return toRoleItem(r, r.MenuIDs), nil
}

// UpdateRole 处理 PUT /api/v1/role/update（权限：role:edit）。
func UpdateRole(ctx context.Context, req *v1.UpdateRoleRequest) (*v1.RoleItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "role:edit"); err != nil {
		return nil, err
	}

	r, svcErr := roleSvc(ctx).UpdateRole(&model.UpdateRoleParam{
		ID:       req.ID,
		RoleName: req.RoleName,
		RoleSort: req.RoleSort,
		Status:   req.Status,
		Remark:   req.Remark,
		MenuIDs:  req.MenuIDs,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	// 角色权限变更影响所有绑定管理员，全量失效缓存。
	adminapi.InvalidateAllPerms()
	return toRoleItem(r, r.MenuIDs), nil
}

// DeleteRole 处理 DELETE /api/v1/role/delete（权限：role:remove）。
func DeleteRole(ctx context.Context, req *v1.DeleteRoleRequest) (*struct{}, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "role:remove"); err != nil {
		return nil, err
	}

	if svcErr := roleSvc(ctx).DeleteRole(req.ID); svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidateAllPerms()
	return &struct{}{}, nil
}

// ============================================================
// 私有辅助
// ============================================================

// roleSvc 构建角色 Service。
func roleSvc(ctx context.Context) rolesvc.Service {
	return rolesvc.New(dao.NewRoleDao(newATX(ctx)))
}

// toRoleItem 将领域模型转换为响应 DTO。
func toRoleItem(r *model.Role, menuIDs []uint64) *v1.RoleItem {
	return &v1.RoleItem{
		ID:        r.ID,
		RoleName:  r.RoleName,
		RoleKey:   r.RoleKey,
		RoleSort:  r.RoleSort,
		Status:    r.Status,
		Remark:    r.Remark,
		MenuIDs:   menuIDs,
		CreatedAt: r.CreatedAt,
	}
}
