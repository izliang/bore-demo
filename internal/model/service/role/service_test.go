// Package role 角色 Service 层单元测试。
//
// 测试策略：通过 fake dao 完全隔离数据库，覆盖角色 CRUD 与保护规则
// （内置角色不可改删、被引用角色不可删、role_key 唯一）。
package role

import (
	"reflect"
	"testing"
	"time"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"gorm.io/gorm"
)

// fakeRoleDao 是 dao.RoleDao 的测试替身（仅配置用到的行为）。
type fakeRoleDao struct {
	getRole            func(id uint64) (*dao.Role, error)
	getRoleByKey       func(key string) (*dao.Role, error)
	createRole         func(r *dao.Role) error
	updateRole         func(r *dao.Role) error
	deleteRole         func(id uint64) error
	listRoles          func(keyword string, offset, limit int) ([]*dao.Role, int64, error)
	listEnabledRoles   func() ([]*dao.Role, error)
	replaceRoleMenus   func(roleID uint64, menuIDs []uint64) error
	getMenuIDsByRole   func(roleID uint64) ([]uint64, error)
	countAdminRefs     func(roleID uint64) (int64, error)
	getRoleKeysByAdmin func(adminID uint64) ([]string, error)
	countRoles         func() (int64, error)
}

func (f *fakeRoleDao) GetRole(id uint64) (*dao.Role, error) {
	if f.getRole != nil {
		return f.getRole(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRoleDao) GetRoleByKey(key string) (*dao.Role, error) {
	if f.getRoleByKey != nil {
		return f.getRoleByKey(key)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRoleDao) CreateRole(r *dao.Role) error {
	if f.createRole != nil {
		return f.createRole(r)
	}
	return nil
}
func (f *fakeRoleDao) UpdateRole(r *dao.Role) error {
	if f.updateRole != nil {
		return f.updateRole(r)
	}
	return nil
}
func (f *fakeRoleDao) DeleteRole(id uint64) error {
	if f.deleteRole != nil {
		return f.deleteRole(id)
	}
	return nil
}
func (f *fakeRoleDao) ListRoles(keyword string, offset, limit int) ([]*dao.Role, int64, error) {
	if f.listRoles != nil {
		return f.listRoles(keyword, offset, limit)
	}
	return nil, 0, nil
}
func (f *fakeRoleDao) ListEnabledRoles() ([]*dao.Role, error) {
	if f.listEnabledRoles != nil {
		return f.listEnabledRoles()
	}
	return nil, nil
}
func (f *fakeRoleDao) ReplaceRoleMenus(roleID uint64, menuIDs []uint64) error {
	if f.replaceRoleMenus != nil {
		return f.replaceRoleMenus(roleID, menuIDs)
	}
	return nil
}
func (f *fakeRoleDao) GetMenuIDsByRoleID(roleID uint64) ([]uint64, error) {
	if f.getMenuIDsByRole != nil {
		return f.getMenuIDsByRole(roleID)
	}
	return nil, nil
}
func (f *fakeRoleDao) CountAdminRefs(roleID uint64) (int64, error) {
	if f.countAdminRefs != nil {
		return f.countAdminRefs(roleID)
	}
	return 0, nil
}
func (f *fakeRoleDao) GetRoleKeysByAdminID(adminID uint64) ([]string, error) {
	if f.getRoleKeysByAdmin != nil {
		return f.getRoleKeysByAdmin(adminID)
	}
	return nil, nil
}
func (f *fakeRoleDao) CountRoles() (int64, error) {
	if f.countRoles != nil {
		return f.countRoles()
	}
	return 0, nil
}

// demoRole 构造一个普通角色实体。
func demoRole(id uint64, key string) *dao.Role {
	return &dao.Role{
		ID:        id,
		RoleName:  "运维角色",
		RoleKey:   key,
		RoleSort:  3,
		DataScope: "1",
		Status:    1,
		Remark:    "测试",
	}
}

func TestGetRole(t *testing.T) {
	tests := []struct {
		name     string
		roleDao  dao.RoleDao
		wantCode string
		wantErr  bool
		want     *model.Role
	}{
		{
			name:     "失败_角色不存在",
			roleDao:  &fakeRoleDao{},
			wantCode: "R0001",
			wantErr:  true,
		},
		{
			name: "成功_返回角色及菜单关联",
			roleDao: &fakeRoleDao{
				getRole: func(id uint64) (*dao.Role, error) { return demoRole(id, "ops"), nil },
				getMenuIDsByRole: func(uint64) ([]uint64, error) {
					return []uint64{1, 2}, nil
				},
			},
			want: &model.Role{
				ID:        1,
				RoleName:  "运维角色",
				RoleKey:   "ops",
				RoleSort:  3,
				DataScope: "1",
				Status:    1,
				Remark:    "测试",
				MenuIDs:   []uint64{1, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.roleDao).GetRole(1)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\n期望: %+v\n实际: %+v", tt.want, got)
			}
		})
	}
}

func TestCreateRole(t *testing.T) {
	tests := []struct {
		name     string
		param    *model.CreateRoleParam
		roleDao  dao.RoleDao
		wantCode string
		wantErr  bool
	}{
		{
			name:  "失败_角色标识已存在",
			param: &model.CreateRoleParam{RoleName: "运维", RoleKey: "ops", Status: 1},
			roleDao: &fakeRoleDao{
				getRoleByKey: func(string) (*dao.Role, error) { return demoRole(9, "ops"), nil },
			},
			wantCode: "R0002",
			wantErr:  true,
		},
		{
			name: "成功_创建并分配菜单",
			param: &model.CreateRoleParam{
				RoleName: "运维", RoleKey: "ops", RoleSort: 2, Status: 1, Remark: "x",
				MenuIDs: []uint64{1, 2},
			},
			roleDao: &fakeRoleDao{
				createRole: func(r *dao.Role) error { r.ID = 10; return nil },
				replaceRoleMenus: func(roleID uint64, menuIDs []uint64) error {
					if roleID != 10 || !reflect.DeepEqual(menuIDs, []uint64{1, 2}) {
						t.Errorf("ReplaceRoleMenus 参数错误: roleID=%d menuIDs=%v", roleID, menuIDs)
					}
					return nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.roleDao).CreateRole(tt.param)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
		})
	}
}

func TestUpdateRole(t *testing.T) {
	tests := []struct {
		name     string
		param    *model.UpdateRoleParam
		roleDao  dao.RoleDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_角色不存在",
			param:    &model.UpdateRoleParam{ID: 99},
			roleDao:  &fakeRoleDao{},
			wantCode: "R0001",
			wantErr:  true,
		},
		{
			name:  "失败_内置角色不可修改",
			param: &model.UpdateRoleParam{ID: 1, RoleName: "改名"},
			roleDao: &fakeRoleDao{
				getRole: func(id uint64) (*dao.Role, error) {
					return demoRole(id, model.RoleKeySuperAdmin), nil
				},
			},
			wantCode: "R0003",
			wantErr:  true,
		},
		{
			name:  "成功_更新普通角色并重分配菜单",
			param: &model.UpdateRoleParam{ID: 2, RoleName: "新名字", Status: 1, MenuIDs: []uint64{3}},
			roleDao: &fakeRoleDao{
				getRole: func(id uint64) (*dao.Role, error) { return demoRole(id, "ops"), nil },
				updateRole: func(r *dao.Role) error {
					if r.RoleName != "新名字" || r.RoleKey != "ops" {
						t.Errorf("UpdateRole 参数错误: %+v", r)
					}
					return nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.roleDao).UpdateRole(tt.param)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
		})
	}
}

func TestDeleteRole(t *testing.T) {
	tests := []struct {
		name     string
		id       uint64
		roleDao  dao.RoleDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_角色不存在",
			id:       99,
			roleDao:  &fakeRoleDao{},
			wantCode: "R0001",
			wantErr:  true,
		},
		{
			name: "失败_内置角色不可删除",
			id:   1,
			roleDao: &fakeRoleDao{
				getRole: func(id uint64) (*dao.Role, error) {
					return demoRole(id, model.RoleKeyAdmin), nil
				},
			},
			wantCode: "R0003",
			wantErr:  true,
		},
		{
			name: "失败_角色被管理员绑定",
			id:   2,
			roleDao: &fakeRoleDao{
				getRole:        func(id uint64) (*dao.Role, error) { return demoRole(id, "ops"), nil },
				countAdminRefs: func(uint64) (int64, error) { return 2, nil },
			},
			wantCode: "R0004",
			wantErr:  true,
		},
		{
			name: "成功_删除无引用的普通角色",
			id:   2,
			roleDao: &fakeRoleDao{
				getRole:        func(id uint64) (*dao.Role, error) { return demoRole(id, "ops"), nil },
				countAdminRefs: func(uint64) (int64, error) { return 0, nil },
				deleteRole:     func(id uint64) error { return nil },
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.roleDao).DeleteRole(tt.id)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
		})
	}
}

func TestListAndCount(t *testing.T) {
	svc := New(&fakeRoleDao{
		listRoles: func(keyword string, offset, limit int) ([]*dao.Role, int64, error) {
			if keyword == "ops" && (offset != 10 || limit != 10) {
				t.Errorf("ListRoles 分页参数错误: offset=%d limit=%d", offset, limit)
			}
			return []*dao.Role{demoRole(1, "ops")}, 1, nil
		},
		listEnabledRoles: func() ([]*dao.Role, error) { return []*dao.Role{demoRole(2, "ops")}, nil },
		countRoles:       func() (int64, error) { return 3, nil },
	})

	list, total, err := svc.ListRoles(&model.RoleQueryParam{Keyword: "ops", Page: 2, PerPage: 10})
	if err != nil {
		t.Fatalf("ListRoles 失败: %+v", err)
	}
	if total != 1 || len(list) != 1 || list[0].RoleKey != "ops" {
		t.Errorf("ListRoles 结果异常: total=%d list=%+v", total, list)
	}

	opts, err := svc.ListOptions()
	if err != nil || len(opts) != 1 {
		t.Fatalf("ListOptions 异常: %+v err=%v", opts, err)
	}

	// 分页参数归一化：Page/PerPage 缺省时使用默认值。
	_, _, err = svc.ListRoles(&model.RoleQueryParam{})
	if err != nil {
		t.Fatalf("ListRoles 默认分页失败: %+v", err)
	}

	n, err := svc.CountRoles()
	if err != nil || n != 3 {
		t.Fatalf("CountRoles 异常: n=%d err=%v", n, err)
	}
}

var _ = time.Now // 保留 time 引用（后续用例扩展）
