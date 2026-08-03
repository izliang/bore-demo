// Package admin 管理员 Service 层单元测试。
//
// 测试策略：通过 fake dao 完全隔离数据库，覆盖登录、CRUD 保护规则。
package admin

import (
	"errors"
	"testing"
	"time"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore-demo/internal/security"
	"gorm.io/gorm"
)

// fakeAdminDao 是 dao.AdminDao 的测试替身（仅配置用到的行为）。
type fakeAdminDao struct {
	getAdminByUsername    func(username string) (*dao.Admin, error)
	getAdmin              func(id uint64) (*dao.Admin, error)
	createAdmin           func(a *dao.Admin) error
	updateAdmin           func(a *dao.Admin) error
	deleteAdmin           func(id uint64) error
	touchLastLogin        func(id uint64, t time.Time) error
	replaceAdminRoles     func(adminID uint64, roleIDs []uint64) error
	getRoleIDsByAdminID   func(adminID uint64) ([]uint64, error)
	countOtherSuperAdmins func(excludeID uint64) (int64, error)
}

func (f *fakeAdminDao) GetAdmin(id uint64) (*dao.Admin, error) {
	if f.getAdmin != nil {
		return f.getAdmin(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeAdminDao) GetAdminByUsername(username string) (*dao.Admin, error) {
	if f.getAdminByUsername != nil {
		return f.getAdminByUsername(username)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeAdminDao) CreateAdmin(a *dao.Admin) error {
	if f.createAdmin != nil {
		return f.createAdmin(a)
	}
	return nil
}
func (f *fakeAdminDao) UpdateAdmin(a *dao.Admin) error {
	if f.updateAdmin != nil {
		return f.updateAdmin(a)
	}
	return nil
}
func (f *fakeAdminDao) DeleteAdmin(id uint64) error {
	if f.deleteAdmin != nil {
		return f.deleteAdmin(id)
	}
	return nil
}
func (f *fakeAdminDao) ListAdmins(string, int, int) ([]*dao.Admin, int64, error) {
	return nil, 0, nil
}
func (f *fakeAdminDao) CountAdmins() (int64, error) { return 0, nil }
func (f *fakeAdminDao) TouchLastLogin(id uint64, t time.Time) error {
	if f.touchLastLogin != nil {
		return f.touchLastLogin(id, t)
	}
	return nil
}
func (f *fakeAdminDao) ReplaceAdminRoles(adminID uint64, roleIDs []uint64) error {
	if f.replaceAdminRoles != nil {
		return f.replaceAdminRoles(adminID, roleIDs)
	}
	return nil
}
func (f *fakeAdminDao) GetRoleIDsByAdminID(adminID uint64) ([]uint64, error) {
	if f.getRoleIDsByAdminID != nil {
		return f.getRoleIDsByAdminID(adminID)
	}
	return nil, nil
}
func (f *fakeAdminDao) CountOtherSuperAdmins(excludeID uint64) (int64, error) {
	if f.countOtherSuperAdmins != nil {
		return f.countOtherSuperAdmins(excludeID)
	}
	return 1, nil
}

// fakeRoleDao 是 dao.RoleDao 的测试替身。
type fakeRoleDao struct {
	getRole            func(id uint64) (*dao.Role, error)
	getRoleKeysByAdmin func(adminID uint64) ([]string, error)
}

func (f *fakeRoleDao) GetRole(id uint64) (*dao.Role, error) {
	if f.getRole != nil {
		return f.getRole(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRoleDao) GetRoleByKey(string) (*dao.Role, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRoleDao) CreateRole(*dao.Role) error { return nil }
func (f *fakeRoleDao) UpdateRole(*dao.Role) error { return nil }
func (f *fakeRoleDao) DeleteRole(uint64) error    { return nil }
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
func (f *fakeRoleDao) GetRoleKeysByAdminID(adminID uint64) ([]string, error) {
	if f.getRoleKeysByAdmin != nil {
		return f.getRoleKeysByAdmin(adminID)
	}
	return nil, nil
}

// enabledAdmin 构造一个启用状态的管理员实体（密码 admin123）。
func enabledAdmin(t *testing.T) *dao.Admin {
	t.Helper()
	hash, err := security.HashPassword("admin123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return &dao.Admin{ID: 1, Username: "admin", PasswordHash: hash, Status: 1}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name     string
		adminDao dao.AdminDao
		password string
		wantCode string
		wantErr  bool
	}{
		{
			name: "成功_凭证正确",
			adminDao: &fakeAdminDao{
				getAdminByUsername: func(string) (*dao.Admin, error) { return enabledAdmin(t), nil },
			},
			password: "admin123",
			wantErr:  false,
		},
		{
			name: "失败_密码错误",
			adminDao: &fakeAdminDao{
				getAdminByUsername: func(string) (*dao.Admin, error) { return enabledAdmin(t), nil },
			},
			password: "wrong",
			wantCode: "A0003",
			wantErr:  true,
		},
		{
			name: "失败_账号禁用",
			adminDao: &fakeAdminDao{
				getAdminByUsername: func(string) (*dao.Admin, error) {
					a := enabledAdmin(t)
					a.Status = 0
					return a, nil
				},
			},
			password: "admin123",
			wantCode: "A0004",
			wantErr:  true,
		},
		{
			name: "失败_用户不存在",
			adminDao: &fakeAdminDao{
				getAdminByUsername: func(string) (*dao.Admin, error) {
					return nil, gorm.ErrRecordNotFound
				},
			},
			password: "admin123",
			wantCode: "A0003",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.adminDao, &fakeRoleDao{})
			_, err := svc.Login("admin", tt.password)

			if tt.wantErr {
				if err == nil {
					t.Fatal("期望错误，实际为 nil")
				}
				if err.GetCode() != tt.wantCode {
					t.Errorf("期望错误码 %s，实际 %s", tt.wantCode, err.GetCode())
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
		})
	}
}

func TestCreateAdmin(t *testing.T) {
	tests := []struct {
		name     string
		param    *model.CreateAdminParam
		adminDao dao.AdminDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_密码过短",
			param:    &model.CreateAdminParam{Username: "ops", Password: "123"},
			wantCode: "A0010",
			wantErr:  true,
		},
		{
			name:  "失败_用户名已存在",
			param: &model.CreateAdminParam{Username: "admin", Password: "admin123"},
			adminDao: &fakeAdminDao{
				getAdminByUsername: func(string) (*dao.Admin, error) {
					return &dao.Admin{ID: 9, Username: "admin"}, nil
				},
			},
			wantCode: "A0002",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.adminDao, &fakeRoleDao{})
			_, err := svc.CreateAdmin(tt.param)
			if !tt.wantErr {
				t.Fatal("本组用例均为失败场景")
			}
			if err == nil || err.GetCode() != tt.wantCode {
				t.Errorf("期望错误码 %s，实际 %v", tt.wantCode, err)
			}
		})
	}
}

func TestDeleteAdmin(t *testing.T) {
	tests := []struct {
		name     string
		id       uint64
		operator uint64
		adminDao dao.AdminDao
		roleDao  dao.RoleDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_删除自己",
			id:       1,
			operator: 1,
			wantCode: "A0005",
			wantErr:  true,
		},
		{
			name:     "失败_删除最后一个超管",
			id:       2,
			operator: 1,
			adminDao: &fakeAdminDao{
				getAdmin: func(id uint64) (*dao.Admin, error) {
					return &dao.Admin{ID: id, Username: "boss", Status: 1}, nil
				},
				countOtherSuperAdmins: func(uint64) (int64, error) { return 0, nil },
			},
			roleDao: &fakeRoleDao{
				getRoleKeysByAdmin: func(uint64) ([]string, error) {
					return []string{model.RoleKeySuperAdmin}, nil
				},
			},
			wantCode: "A0006",
			wantErr:  true,
		},
		{
			name:     "成功_删除普通管理员",
			id:       2,
			operator: 1,
			adminDao: &fakeAdminDao{
				getAdmin: func(id uint64) (*dao.Admin, error) {
					return &dao.Admin{ID: id, Username: "ops", Status: 1}, nil
				},
				deleteAdmin: func(id uint64) error { return nil },
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.adminDao == nil {
				tt.adminDao = &fakeAdminDao{}
			}
			if tt.roleDao == nil {
				tt.roleDao = &fakeRoleDao{}
			}
			svc := New(tt.adminDao, tt.roleDao)
			err := svc.DeleteAdmin(tt.id, tt.operator)

			if tt.wantErr {
				if err == nil {
					t.Fatal("期望错误，实际为 nil")
				}
				if err.GetCode() != tt.wantCode {
					t.Errorf("期望错误码 %s，实际 %s", tt.wantCode, err.GetCode())
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
		})
	}
}

func TestUpdateAdmin_DisableLastSuperAdmin(t *testing.T) {
	svc := New(&fakeAdminDao{
		getAdmin: func(id uint64) (*dao.Admin, error) {
			return &dao.Admin{ID: id, Username: "boss", Status: 1}, nil
		},
		countOtherSuperAdmins: func(uint64) (int64, error) { return 0, nil },
	}, &fakeRoleDao{
		getRoleKeysByAdmin: func(uint64) ([]string, error) {
			return []string{model.RoleKeySuperAdmin}, nil
		},
	})

	_, err := svc.UpdateAdmin(&model.UpdateAdminParam{
		ID:       2,
		Username: "boss",
		Status:   model.AdminStatusDisabled,
	})
	if err == nil {
		t.Fatal("期望禁用最后一个超管被拒绝，实际成功")
	}
	if err.GetCode() != "A0006" {
		t.Errorf("期望错误码 A0006，实际 %s", err.GetCode())
	}
}

var _ = errors.New // 保留 errors 引用（后续用例扩展）
