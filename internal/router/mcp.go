// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package router

import (
	"github.com/lin-snow/ech0/internal/handler"
	"github.com/lin-snow/ech0/internal/middleware"
	authModel "github.com/lin-snow/ech0/internal/model/auth"
)

func setupMCPRoutes(groups *AppRouterGroup, h *handler.Bundle) {
	g := groups.MCPRouterGroup
	g.Use(
		middleware.RateLimit(20, 40),
		middleware.OriginGuard(nil),
		middleware.RequireAudience(authModel.AudienceMCPRemote),
	)
	g.POST("", h.MCPHandler.ServeEndpoint())
	// 2026-07-28 Streamable HTTP is POST-only; ServeEndpoint answers legacy
	// GET (status/SSE probe) and DELETE (session teardown) with 405.
	g.GET("", h.MCPHandler.ServeEndpoint())
	g.DELETE("", h.MCPHandler.ServeEndpoint())
}
