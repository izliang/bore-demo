// Package auth 鉴权 Service 层单元测试。
//
// 测试策略：通过 fake dao 完全隔离数据库，覆盖权限聚合逻辑：
// 账号不存在/禁用拒绝、普通管理员按角色聚合、超管返回全量权限。
package auth

import (
	"reflect"
	"testing"
	"time"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"gorm.io/gorm"
)

// fakeAdminDao 是 dao.AdminDao 的测试替身（仅 GetAdmin 被本模块使用）。
type fakeAdminDao struct {
	getAdmin func(id uint64) (*dao.Admin, error)
}

func (f *fakeAdminDao) GetAdmin(id uint64) (*dao.Admin, error) {
	if f.getAdmin != nil {
		return f.getAdmin(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeAdminDao) GetAdminByUsername(string) (*dao.Admin, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeAdminDao) CreateAdmin(*dao.Admin) error { return nil }
func (f *fakeAdminDao) UpdateAdmin(*dao.Admin) error { return nil }
func (f *fakeAdminDao) DeleteAdmin(uint64) error     { return nil }
func (f *fakeAdminDao) ListAdmins(string, int, int) ([]*dao.Admin, int64, error) {
	return nil, 0, nil
}
func (f *fakeAdminDao) CountAdmins() (int64, error)              { return 0, nil }
func (f *fakeAdminDao) TouchLastLogin(uint64, time.Time) error   { return nil }
func (f *fakeAdminDao) ReplaceAdminRoles(uint64, []uint64) error { return nil }
func (f *fakeAdminDao) GetRoleIDsByAdminID(uint64) ([]uint64, error) {
	return nil, nil
}
func (f *fakeAdminDao) CountOtherSuperAdmins(uint64) (int64, error) {
	return 0, nil
}

// fakeRoleDao 是 dao.RoleDao 的测试替身（仅 GetRoleKeysByAdminID 被本模块使用）。
type fakeRoleDao struct {
	getRoleKeysByAdmin func(adminID uint64) ([]string, error)
}

func (f *fakeRoleDao) GetRoleKeysByAdminID(adminID uint64) ([]string, error) {
	if f.getRoleKeysByAdmin != nil {
		return f.getRoleKeysByAdmin(adminID)
	}
	return nil, nil
}
func (f *fakeRoleDao) GetRole(uint64) (*dao.Role, error)      { return nil, gorm.ErrRecordNotFound }
func (f *fakeRoleDao) GetRoleByKey(string) (*dao.Role, error) { return nil, gorm.ErrRecordNotFound }
func (f *fakeRoleDao) CreateRole(*dao.Role) error             { return nil }
func (f *fakeRoleDao) UpdateRole(*dao.Role) error             { return nil }
func (f *fakeRoleDao) DeleteRole(uint64) error                { return nil }
func (f *fakeRoleDao) ListRoles(string, int, int) ([]*dao.Role, int64, error) {
	return nil, 0, nil
}
func (f *fakeRoleDao) ListEnabledRoles() ([]*dao.Role, error)  { return nil, nil }
func (f *fakeRoleDao) ReplaceRoleMenus(uint64, []uint64) error { return nil }
func (f *fakeRoleDao) GetMenuIDsByRoleID(uint64) ([]uint64, error) {
	return nil, nil
}
func (f *fakeRoleDao) CountAdminRefs(uint64) (int64, error) { return 0, nil }
func (f *fakeRoleDao) CountRoles() (int64, error)           { return 0, nil }

// fakeMenuDao 是 dao.MenuDao 的测试替身（仅权限聚合方法被本模块使用）。
type fakeMenuDao struct {
	getPermsByAdmin func(adminID uint64) ([]string, error)
	listAllPerms    func() ([]string, error)
}

func (f *fakeMenuDao) GetPermsByAdminID(adminID uint64) ([]string, error) {
	if f.getPermsByAdmin != nil {
		return f.getPermsByAdmin(adminID)
	}
	return nil, nil
}
func (f *fakeMenuDao) ListAllPerms() ([]string, error) {
	if f.listAllPerms != nil {
		return f.listAllPerms()
	}
	return nil, nil
}
func (f *fakeMenuDao) GetMenu(uint64) (*dao.Menu, error)        { return nil, gorm.ErrRecordNotFound }
func (f *fakeMenuDao) CreateMenu(*dao.Menu) error               { return nil }
func (f *fakeMenuDao) UpdateMenu(*dao.Menu) error               { return nil }
func (f *fakeMenuDao) DeleteMenu(uint64) error                  { return nil }
func (f *fakeMenuDao) ListMenus() ([]*dao.Menu, error)          { return nil, nil }
func (f *fakeMenuDao) ListChildren(uint64) ([]*dao.Menu, error) { return nil, nil }
func (f *fakeMenuDao) CountMenus() (int64, error)               { return 0, nil }

// enabledAdmin 构造一个启用状态的管理员。
func enabledAdmin() *dao.Admin {
	return &dao.Admin{ID: 1, Username: "admin", Status: 1}
}

func TestGetPerms(t *testing.T) {
	tests := []struct {
		name     string
		adminDao dao.AdminDao
		roleDao  dao.RoleDao
		menuDao  dao.MenuDao
		wantCode string
		wantErr  bool
		want     *model.AdminPerms
	}{
		{
			name:     "失败_管理员不存在",
			adminDao: &fakeAdminDao{},
			wantCode: "A0007",
			wantErr:  true,
		},
		{
			name: "失败_管理员已禁用",
			adminDao: &fakeAdminDao{
				getAdmin: func(uint64) (*dao.Admin, error) {
					a := enabledAdmin()
					a.Status = 0
					return a, nil
				},
			},
			wantCode: "A0007",
			wantErr:  true,
		},
		{
			name: "成功_普通管理员按角色聚合权限",
			adminDao: &fakeAdminDao{
				getAdmin: func(uint64) (*dao.Admin, error) { return enabledAdmin(), nil },
			},
			roleDao: &fakeRoleDao{
				getRoleKeysByAdmin: func(uint64) ([]string, error) {
					return []string{model.RoleKeyAdmin}, nil
				},
			},
			menuDao: &fakeMenuDao{
				getPermsByAdmin: func(uint64) ([]string, error) {
					return []string{"admin:list", "admin:add"}, nil
				},
			},
			want: &model.AdminPerms{
				Roles:   []string{model.RoleKeyAdmin},
				Perms:   []string{"admin:list", "admin:add"},
				IsSuper: false,
			},
		},
		{
			name: "成功_超管返回全量权限",
			adminDao: &fakeAdminDao{
				getAdmin: func(uint64) (*dao.Admin, error) { return enabledAdmin(), nil },
			},
			roleDao: &fakeRoleDao{
				getRoleKeysByAdmin: func(uint64) ([]string, error) {
					return []string{model.RoleKeySuperAdmin}, nil
				},
			},
			menuDao: &fakeMenuDao{
				listAllPerms: func() ([]string, error) {
					return []string{"admin:list", "role:list", "menu:list"}, nil
				},
				// 超管路径不应触发普通聚合查询。
				getPermsByAdmin: func(uint64) ([]string, error) {
					t.Fatal("超管不应调用 GetPermsByAdminID")
					return nil, nil
				},
			},
			want: &model.AdminPerms{
				Roles:   []string{model.RoleKeySuperAdmin},
				Perms:   []string{"admin:list", "role:list", "menu:list"},
				IsSuper: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.adminDao, tt.roleDao, tt.menuDao).GetPerms(1)
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
