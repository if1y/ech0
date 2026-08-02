// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package mcp

// ProtocolVersion is the MCP revision this server implements. Revision
// 2026-07-28 is stateless: there is no initialize handshake, every request
// carries its protocol version in params._meta (mirrored into the
// MCP-Protocol-Version header), and every result carries resultType plus,
// where required, caching hints.
const (
	ProtocolVersion = "2026-07-28"
	ServerName      = "ech0-mcp"
)

// SupportedVersions is advertised by server/discover and in
// UnsupportedProtocolVersion error data.
var SupportedVersions = []string{ProtocolVersion}

// _meta keys defined by the 2026-07-28 revision.
const (
	metaKeyProtocolVersion = "io.modelcontextprotocol/protocolVersion"
	metaKeyServerInfo      = "io.modelcontextprotocol/serverInfo"
)

const resultTypeComplete = "complete"

// Cache scopes for CacheableResult (ttlMs + cacheScope).
const (
	cacheScopePublic  = "public"
	cacheScopePrivate = "private"
)

// Freshness hints in milliseconds. Tool and resource definitions are fixed
// at process start, so discover/list results cache well; read results are
// live data and marked immediately stale.
const (
	discoverTTLMs = 60 * 60 * 1000
	listTTLMs     = 5 * 60 * 1000
)

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

// ResultEnvelope carries the fields every 2026-07-28 result must include:
// the mandatory resultType, and the _meta serverInfo the spec recommends.
// Server.handlePost stamps it on every successful result via complete().
type ResultEnvelope struct {
	ResultType string         `json:"resultType"`
	Meta       map[string]any `json:"_meta,omitempty"`
}

func (e *ResultEnvelope) complete(info ServerInfo) {
	e.ResultType = resultTypeComplete
	if e.Meta == nil {
		e.Meta = make(map[string]any, 1)
	}
	e.Meta[metaKeyServerInfo] = info
}

// completer is implemented by every result type through ResultEnvelope.
type completer interface{ complete(info ServerInfo) }

// CacheInfo is the CacheableResult contract: required on server/discover,
// tools/list, resources/list and resources/read results.
type CacheInfo struct {
	TTLMs      int64  `json:"ttlMs"`
	CacheScope string `json:"cacheScope"`
}

// DiscoverResult answers server/discover, the one method every 2026-07-28
// server must implement. It replaces the removed initialize handshake.
type DiscoverResult struct {
	ResultEnvelope
	SupportedVersions []string           `json:"supportedVersions"`
	Capabilities      ServerCapabilities `json:"capabilities"`
	Instructions      string             `json:"instructions,omitempty"`
	CacheInfo
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
