package config

import "github.com/Monster-Pipe/monster-pipe-core/pkg/utils"

type MonsterPipeAppConfig struct {
	ManagerListenAddr string                         `json:"manager_listen_addr"`
	GinMode           utils.ConfigItemReader[string] `json:"gin_mode"`
}
