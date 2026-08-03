package v1

// ============================================================
// 菜单/按钮权限（Menu）
// ============================================================

// MenuNode 菜单树节点。
type MenuNode struct {
	ID       uint64      `json:"id"`
	ParentID uint64      `json:"parent_id"`
	MenuName string      `json:"menu_name"`
	MenuType string      `json:"menu_type"`
	Path     string      `json:"path"`
	Perms    string      `json:"perms"`
	Icon     string      `json:"icon"`
	OrderNum int         `json:"order_num"`
	Visible  int8        `json:"visible"`
	Children []*MenuNode `json:"children,omitempty"`
}

// MenuTreeResponse 菜单树响应（data 为数组，供 amis tree 组件直接渲染）。
type MenuTreeResponse []*MenuNode

// GetMenuRequest 菜单详情请求（GET /api/v1/menu/get）。
type GetMenuRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"菜单 ID"`
}

// CreateMenuRequest 新增菜单/按钮请求（POST /api/v1/menu/create）。
type CreateMenuRequest struct {
	ParentID uint64 `json:"parent_id" default:"0" doc:"父菜单 ID（0 为顶级）"`
	MenuName string `json:"menu_name" required:"true" minLength:"1" maxLength:"64" doc:"菜单名称"`
	MenuType string `json:"menu_type" required:"true" enum:"M,C,F" doc:"菜单类型（M 目录 / C 菜单 / F 按钮）"`
	Path     string `json:"path" maxLength:"200" doc:"页面路由路径"`
	Perms    string `json:"perms" maxLength:"100" doc:"权限标识（如 admin:list）"`
	Icon     string `json:"icon" maxLength:"100" doc:"图标类名"`
	OrderNum int    `json:"order_num" default:"0" doc:"显示顺序"`
	Visible  int8   `json:"visible" default:"1" doc:"是否显示（1 显示 / 0 隐藏）"`
}

// UpdateMenuRequest 更新菜单/按钮请求（PUT /api/v1/menu/update）。
type UpdateMenuRequest struct {
	ID       uint64 `json:"id" required:"true" minimum:"1" doc:"菜单 ID"`
	ParentID uint64 `json:"parent_id" default:"0" doc:"父菜单 ID（0 为顶级）"`
	MenuName string `json:"menu_name" required:"true" minLength:"1" maxLength:"64" doc:"菜单名称"`
	MenuType string `json:"menu_type" required:"true" enum:"M,C,F" doc:"菜单类型（M 目录 / C 菜单 / F 按钮）"`
	Path     string `json:"path" maxLength:"200" doc:"页面路由路径"`
	Perms    string `json:"perms" maxLength:"100" doc:"权限标识（如 admin:list）"`
	Icon     string `json:"icon" maxLength:"100" doc:"图标类名"`
	OrderNum int    `json:"order_num" default:"0" doc:"显示顺序"`
	Visible  int8   `json:"visible" default:"1" doc:"是否显示（1 显示 / 0 隐藏）"`
}

// DeleteMenuRequest 删除菜单/按钮请求（DELETE /api/v1/menu/delete?id=1）。
type DeleteMenuRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"菜单 ID"`
}
