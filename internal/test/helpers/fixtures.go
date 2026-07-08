// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package helpers

import (
	echoModel "github.com/lin-snow/ech0/internal/model/echo"
	userModel "github.com/lin-snow/ech0/internal/model/user"
)

// NewUser 构造带合理默认值的用户；用 option 覆盖字段。例：helpers.NewUser(helpers.AsAdmin)。
func NewUser(opts ...func(*userModel.User)) userModel.User {
	u := userModel.User{
		ID:       "user-test-0001",
		Username: "tester",
	}
	for _, o := range opts {
		o(&u)
	}
	return u
}

// AsAdmin 把用户标记为管理员。
func AsAdmin(u *userModel.User) { u.IsAdmin = true }

// AsOwner 把用户标记为站长（同时是管理员）。
func AsOwner(u *userModel.User) {
	u.IsAdmin = true
	u.IsOwner = true
}

// NewEcho 构造带合理默认值的 echo；用 option 覆盖字段。例：helpers.NewEcho(helpers.AsPrivate)。
func NewEcho(opts ...func(*echoModel.Echo)) echoModel.Echo {
	e := echoModel.Echo{
		ID:      "echo-test-0001",
		Content: "hello world",
		UserID:  "user-test-0001",
	}
	for _, o := range opts {
		o(&e)
	}
	return e
}

// AsPrivate 把 echo 标记为私密。
func AsPrivate(e *echoModel.Echo) { e.Private = true }
