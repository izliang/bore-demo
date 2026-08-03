// Package controller 是 HTTP 控制器层（Controller Layer）。
//
// 本文件为 RBAC 管理后台控制器：管理员 CRUD、统计、鉴权信息。
// 权限声明位于每个方法入口（CheckPerm），便于审计；
// 登录态由路由组的 RequireLogin 中间件保证。
package controller

import (
	"context"

	adminapi "github.com/izliang/bore-demo/internal/api/admin"
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	adminsvc "github.com/izliang/bore-demo/internal/model/service/admin"
	authsvc "github.com/izliang/bore-demo/internal/model/service/auth"
	menusvc "github.com/izliang/bore-demo/internal/model/service/menu"
	rolesvc "github.com/izliang/bore-demo/internal/model/service/role"
	"github.com/izliang/bore-demo/internal/resource"
	v1 "github.com/izliang/bore-demo/internal/resource/api/v1"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/e"
)

// ListAdmins 处理 GET /api/v1/admin/list（权限：admin:list）。
func ListAdmins(ctx context.Context, req *v1.ListAdminsRequest) (*v1.AdminListResponse, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:list"); err != nil {
		return nil, err
	}

	list, total, svcErr := adminSvc(ctx).ListAdmins(&model.AdminQueryParam{
		Keyword: req.Keyword,
		Page:    req.Page,
		PerPage: req.PerPage,
	})
	if svcErr != nil {
		return nil, svcErr
	}

	items := make([]*v1.AdminItem, 0, len(list))
	for _, a := range list {
		items = append(items, toAdminItem(a))
	}
	return &v1.AdminListResponse{Items: items, Total: total}, nil
}

// GetAdmin 处理 GET /api/v1/admin/get（权限：admin:query）。
func GetAdmin(ctx context.Context, req *v1.GetAdminRequest) (*v1.AdminItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:query"); err != nil {
		return nil, err
	}

	a, svcErr := adminSvc(ctx).GetAdmin(req.ID)
	if svcErr != nil {
		return nil, svcErr
	}
	return toAdminItem(a), nil
}

// CreateAdmin 处理 POST /api/v1/admin/create（权限：admin:add）。
func CreateAdmin(ctx context.Context, req *v1.CreateAdminRequest) (*v1.AdminItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:add"); err != nil {
		return nil, err
	}

	a, svcErr := adminSvc(ctx).CreateAdmin(&model.CreateAdminParam{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   model.AdminStatus(req.Status),
		RoleIDs:  req.RoleIDs,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	return toAdminItem(a), nil
}

// UpdateAdmin 处理 PUT /api/v1/admin/update（权限：admin:edit）。
// 更新后主动失效该管理员权限缓存，使其下次请求按新角色/菜单重新加载。
func UpdateAdmin(ctx context.Context, req *v1.UpdateAdminRequest) (*v1.AdminItem, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:edit"); err != nil {
		return nil, err
	}

	a, svcErr := adminSvc(ctx).UpdateAdmin(&model.UpdateAdminParam{
		ID:       req.ID,
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   model.AdminStatus(req.Status),
		RoleIDs:  req.RoleIDs,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidatePerms(req.ID)
	return toAdminItem(a), nil
}

// DeleteAdmin 处理 DELETE /api/v1/admin/delete（权限：admin:remove）。
func DeleteAdmin(ctx context.Context, req *v1.DeleteAdminRequest) (*struct{}, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:remove"); err != nil {
		return nil, err
	}

	operatorID, _ := adminapi.CtxAdminID(ctx)
	if svcErr := adminSvc(ctx).DeleteAdmin(req.ID, operatorID); svcErr != nil {
		return nil, svcErr
	}
	adminapi.InvalidatePerms(req.ID)
	return &struct{}{}, nil
}

// BatchDeleteAdmins 处理 POST /api/v1/admin/batch-delete（权限：admin:remove）。
func BatchDeleteAdmins(ctx context.Context, req *v1.BatchDeleteAdminsRequest) (*struct{}, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "admin:remove"); err != nil {
		return nil, err
	}

	operatorID, _ := adminapi.CtxAdminID(ctx)
	if svcErr := adminSvc(ctx).BatchDeleteAdmins(req.IDs, operatorID); svcErr != nil {
		return nil, svcErr
	}
	for _, id := range req.IDs {
		adminapi.InvalidatePerms(id)
	}
	return &struct{}{}, nil
}

// AdminStats 处理 GET /api/v1/admin/stats（权限：dashboard:view）。
func AdminStats(ctx context.Context, _ *struct{}) (*v1.AdminStatsResponse, e.ServiceError) {
	if err := adminapi.CheckPerm(ctx, "dashboard:view"); err != nil {
		return nil, err
	}

	atx := newATX(ctx)
	admins, svcErr := adminsvc.New(dao.NewAdminDao(atx), dao.NewRoleDao(atx)).CountAdmins()
	if svcErr != nil {
		return nil, svcErr
	}
	roles, svcErr := rolesvc.New(dao.NewRoleDao(atx)).CountRoles()
	if svcErr != nil {
		return nil, svcErr
	}
	menus, svcErr := menusvc.New(dao.NewMenuDao(atx)).ListMenus()
	if svcErr != nil {
		return nil, svcErr
	}
	return &v1.AdminStatsResponse{
		Admins: admins,
		Roles:  roles,
		Menus:  int64(len(menus)),
	}, nil
}

// GetAuthInfo 处理 GET /api/v1/auth/info（登录态即可访问）。
func GetAuthInfo(ctx context.Context, _ *struct{}) (*v1.AuthInfoResponse, e.ServiceError) {
	adminID, ok := adminapi.CtxAdminID(ctx)
	if !ok {
		return nil, adminapi.ErrUnauthorized
	}

	atx := newATX(ctx)
	authService := authsvc.New(
		dao.NewAdminDao(atx),
		dao.NewRoleDao(atx),
		dao.NewMenuDao(atx),
	)
	perms, svcErr := authService.GetPerms(adminID)
	if svcErr != nil {
		return nil, svcErr
	}

	a, svcErr := adminsvc.New(dao.NewAdminDao(atx), dao.NewRoleDao(atx)).GetAdmin(adminID)
	if svcErr != nil {
		return nil, svcErr
	}
	return &v1.AuthInfoResponse{
		ID:       a.ID,
		Username: a.Username,
		Nickname: a.Nickname,
		Avatar:   a.Avatar,
		Roles:    perms.Roles,
		Perms:    perms.Perms,
	}, nil
}

// ============================================================
// 私有辅助
// ============================================================

// newATX 构建请求级 AppContext。
func newATX(ctx context.Context) appctx.AppContext {
	return appctx.New(
		appctx.WithContext(ctx),
		appctx.WithLogger(resource.Logger),
		appctx.WithGORM(resource.DB),
	)
}

// adminSvc 构建管理员 Service。
func adminSvc(ctx context.Context) adminsvc.Service {
	atx := newATX(ctx)
	return adminsvc.New(dao.NewAdminDao(atx), dao.NewRoleDao(atx))
}

// toAdminItem 将领域模型转换为响应 DTO。
func toAdminItem(a *model.Admin) *v1.AdminItem {
	item := &v1.AdminItem{
		ID:          a.ID,
		Username:    a.Username,
		Nickname:    a.Nickname,
		Avatar:      a.Avatar,
		Email:       a.Email,
		Phone:       a.Phone,
		Status:      int8(a.Status),
		RoleIDs:     make([]uint64, 0, len(a.Roles)),
		LastLoginAt: a.LastLoginAt,
		CreatedAt:   a.CreatedAt,
	}
	for _, r := range a.Roles {
		item.Roles = append(item.Roles, v1.RoleBrief{
			ID:       r.ID,
			RoleName: r.RoleName,
			RoleKey:  r.RoleKey,
		})
		item.RoleIDs = append(item.RoleIDs, r.ID)
	}
	return item
}
