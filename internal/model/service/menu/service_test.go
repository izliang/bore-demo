// Package menu 菜单 Service 层单元测试。
//
// 测试策略：通过 fake dao 完全隔离数据库，覆盖菜单树组装、CRUD、
// 类型校验与删除保护（有子节点不可删、不能挂到自身子树）。
package menu

import (
	"testing"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"gorm.io/gorm"
)

// fakeMenuDao 是 dao.MenuDao 的测试替身（仅配置用到的行为）。
type fakeMenuDao struct {
	getMenu         func(id uint64) (*dao.Menu, error)
	createMenu      func(m *dao.Menu) error
	updateMenu      func(m *dao.Menu) error
	deleteMenu      func(id uint64) error
	listMenus       func() ([]*dao.Menu, error)
	listChildren    func(parentID uint64) ([]*dao.Menu, error)
	countMenus      func() (int64, error)
	getPermsByAdmin func(adminID uint64) ([]string, error)
	listAllPerms    func() ([]string, error)
}

func (f *fakeMenuDao) GetMenu(id uint64) (*dao.Menu, error) {
	if f.getMenu != nil {
		return f.getMenu(id)
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeMenuDao) CreateMenu(m *dao.Menu) error {
	if f.createMenu != nil {
		return f.createMenu(m)
	}
	return nil
}
func (f *fakeMenuDao) UpdateMenu(m *dao.Menu) error {
	if f.updateMenu != nil {
		return f.updateMenu(m)
	}
	return nil
}
func (f *fakeMenuDao) DeleteMenu(id uint64) error {
	if f.deleteMenu != nil {
		return f.deleteMenu(id)
	}
	return nil
}
func (f *fakeMenuDao) ListMenus() ([]*dao.Menu, error) {
	if f.listMenus != nil {
		return f.listMenus()
	}
	return nil, nil
}
func (f *fakeMenuDao) ListChildren(parentID uint64) ([]*dao.Menu, error) {
	if f.listChildren != nil {
		return f.listChildren(parentID)
	}
	return nil, nil
}
func (f *fakeMenuDao) CountMenus() (int64, error) {
	if f.countMenus != nil {
		return f.countMenus()
	}
	return 0, nil
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

// menuNode 构造一个菜单实体。
func menuNode(id, parentID uint64, menuType, perms string, order int) *dao.Menu {
	return &dao.Menu{
		ID:       id,
		ParentID: parentID,
		MenuName: "菜单",
		MenuType: menuType,
		Path:     "/x",
		Perms:    perms,
		OrderNum: order,
		Visible:  1,
	}
}

func TestTree(t *testing.T) {
	svc := New(&fakeMenuDao{
		listMenus: func() ([]*dao.Menu, error) {
			return []*dao.Menu{
				menuNode(1, 0, model.MenuTypeDir, "", 2),
				menuNode(2, 0, model.MenuTypeMenu, "user:list", 1),
				menuNode(3, 2, model.MenuTypeButton, "user:add", 0),
				menuNode(4, 2, model.MenuTypeButton, "user:edit", 1),
			}, nil
		},
	})

	tree, err := svc.Tree()
	if err != nil {
		t.Fatalf("Tree 失败: %+v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("期望 2 个根节点，实际 %d", len(tree))
	}
	// 根节点按 order_num 升序：id=2(order 1) 在前，id=1(order 2) 在后。
	if tree[0].ID != 2 || tree[1].ID != 1 {
		t.Fatalf("根节点顺序错误: %+v", tree)
	}
	// 子节点保持 list 顺序（按钮 add/edit）。
	if len(tree[0].Children) != 2 || tree[0].Children[0].ID != 3 || tree[0].Children[1].ID != 4 {
		t.Fatalf("子节点组装错误: %+v", tree[0].Children)
	}
}

func TestCreateMenu(t *testing.T) {
	tests := []struct {
		name     string
		param    *model.CreateMenuParam
		menuDao  dao.MenuDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_菜单类型不合法",
			param:    &model.CreateMenuParam{MenuName: "x", MenuType: "X"},
			wantCode: "M0003",
			wantErr:  true,
		},
		{
			name:     "失败_父菜单不存在",
			param:    &model.CreateMenuParam{ParentID: 99, MenuName: "x", MenuType: model.MenuTypeMenu},
			menuDao:  &fakeMenuDao{},
			wantCode: "M0004",
			wantErr:  true,
		},
		{
			name:  "失败_按钮不能作为父菜单",
			param: &model.CreateMenuParam{ParentID: 5, MenuName: "x", MenuType: model.MenuTypeMenu},
			menuDao: &fakeMenuDao{
				getMenu: func(id uint64) (*dao.Menu, error) {
					return menuNode(id, 0, model.MenuTypeButton, "user:add", 0), nil
				},
			},
			wantCode: "M0006",
			wantErr:  true,
		},
		{
			name: "成功_顶级目录",
			param: &model.CreateMenuParam{
				MenuName: "系统管理", MenuType: model.MenuTypeDir, Path: "/system", OrderNum: 1,
			},
			menuDao: &fakeMenuDao{
				createMenu: func(m *dao.Menu) error { m.ID = 7; return nil },
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.menuDao).CreateMenu(tt.param)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %+v", err)
			}
			if got.ID != 7 || got.MenuType != model.MenuTypeDir {
				t.Errorf("创建结果异常: %+v", got)
			}
		})
	}
}

func TestUpdateMenu_CycleProtection(t *testing.T) {
	// 菜单树：1 → 2 → 3（3 是 2 的子节点）。
	svc := New(&fakeMenuDao{
		getMenu: func(id uint64) (*dao.Menu, error) {
			if id == 1 {
				return menuNode(1, 0, model.MenuTypeDir, "", 0), nil
			}
			return menuNode(2, 1, model.MenuTypeMenu, "user:list", 0), nil
		},
		listMenus: func() ([]*dao.Menu, error) {
			return []*dao.Menu{
				menuNode(1, 0, model.MenuTypeDir, "", 0),
				menuNode(2, 1, model.MenuTypeMenu, "user:list", 0),
				menuNode(3, 2, model.MenuTypeButton, "user:add", 0),
			}, nil
		},
	})

	tests := []struct {
		name     string
		param    *model.UpdateMenuParam
		wantCode string
	}{
		{
			name:     "失败_挂到自己",
			param:    &model.UpdateMenuParam{ID: 2, ParentID: 2, MenuName: "x", MenuType: model.MenuTypeMenu},
			wantCode: "M0005",
		},
		{
			name:     "失败_挂到子节点",
			param:    &model.UpdateMenuParam{ID: 2, ParentID: 3, MenuName: "x", MenuType: model.MenuTypeMenu},
			wantCode: "M0005",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateMenu(tt.param)
			if err == nil || err.GetCode() != tt.wantCode {
				t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
			}
		})
	}
}

func TestDeleteMenu(t *testing.T) {
	tests := []struct {
		name     string
		id       uint64
		menuDao  dao.MenuDao
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_菜单不存在",
			id:       99,
			menuDao:  &fakeMenuDao{},
			wantCode: "M0001",
			wantErr:  true,
		},
		{
			name: "失败_存在子菜单",
			id:   1,
			menuDao: &fakeMenuDao{
				getMenu: func(id uint64) (*dao.Menu, error) { return menuNode(id, 0, model.MenuTypeDir, "", 0), nil },
				listChildren: func(uint64) ([]*dao.Menu, error) {
					return []*dao.Menu{menuNode(2, 1, model.MenuTypeMenu, "user:list", 0)}, nil
				},
			},
			wantCode: "M0002",
			wantErr:  true,
		},
		{
			name: "成功_删除叶子节点",
			id:   3,
			menuDao: &fakeMenuDao{
				getMenu:    func(id uint64) (*dao.Menu, error) { return menuNode(id, 1, model.MenuTypeButton, "user:add", 0), nil },
				deleteMenu: func(id uint64) error { return nil },
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.menuDao).DeleteMenu(tt.id)
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

func TestListMenus(t *testing.T) {
	svc := New(&fakeMenuDao{
		listMenus: func() ([]*dao.Menu, error) {
			return []*dao.Menu{
				menuNode(1, 0, model.MenuTypeDir, "", 1),
				menuNode(2, 1, model.MenuTypeMenu, "user:list", 0),
			}, nil
		},
		getMenu: func(id uint64) (*dao.Menu, error) {
			return menuNode(id, 0, model.MenuTypeDir, "", 1), nil
		},
	})

	list, err := svc.ListMenus()
	if err != nil {
		t.Fatalf("ListMenus 失败: %+v", err)
	}
	if len(list) != 2 || list[1].Perms != "user:list" {
		t.Errorf("ListMenus 结果异常: %+v", list)
	}

	got, err := svc.GetMenu(1)
	if err != nil || got.ID != 1 {
		t.Errorf("GetMenu 异常: %+v err=%v", got, err)
	}
}
