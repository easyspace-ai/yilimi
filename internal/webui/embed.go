package webui

import "embed"

// webdist 目錄在編譯時由 scripts/sync-frontend-dist.sh 從 frontend/dist 同步；不可使用 go:embed 的「..」跨目錄。
//
//go:embed all:webdist
var webDist embed.FS
