package web

import (
	"github.com/doraemonkeys/monster-pipe-core/pkg/utils"
	"github.com/gin-gonic/gin"
)

type MonsterPipeServer struct {
	config *MonsterPipeServerConfig
	engine *gin.Engine
}

type MonsterPipeServerConfig struct {
	TrustedProxies utils.ConfigItem[[]string]
	ListenAddr     utils.ConfigItem[string]
}

func NewMonsterPipeServer(config *MonsterPipeServerConfig) *MonsterPipeServer {
	engine := gin.New()
	engine.SetTrustedProxies(config.TrustedProxies.Get())

	return &MonsterPipeServer{
		config: config,
		engine: gin.New(),
	}
}

func (s *MonsterPipeServer) Run() error {
	return s.engine.Run(s.config.ListenAddr.Get())
}
