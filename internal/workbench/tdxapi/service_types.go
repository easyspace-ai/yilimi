package tdxapi

import "github.com/easyspace-ai/tdx"

// Service holds runtime TDX state (client, data manager, async tasks).
type Service struct {
	client  *tdx.Client
	manager *tdx.Manage
	tasks   *TaskManager
}
