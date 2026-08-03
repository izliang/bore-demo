package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/izliang/bore-demo/internal/model/dao"
	adminsvc "github.com/izliang/bore-demo/internal/model/service/admin"
	"github.com/izliang/bore-demo/internal/resource"
	"github.com/izliang/bore/appctx"
)

// Authenticator 是 amis.SetupAuth 的登录回调。
//
// 校验通过后返回 "admin-{id}" 形式的会话 token，由 amis LoginHandler
// 写入 Cookie（HttpOnly）。错误消息会直接展示在登录页。
func Authenticator(username, password string) (token string, redirect string, err error) {
	atx := appctx.New(
		appctx.WithContext(context.Background()),
		appctx.WithLogger(resource.Logger),
		appctx.WithGORM(resource.DB),
	)
	svc := adminsvc.New(dao.NewAdminDao(atx), dao.NewRoleDao(atx))

	a, svcErr := svc.Login(username, password)
	if svcErr != nil {
		return "", "", errors.New(svcErr.GetMessage())
	}
	return fmt.Sprintf("%s%d", tokenPrefix, a.ID), "", nil
}
