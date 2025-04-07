package config

import (
	"encoding/json"
	"os"
	"sync"
)

type ConfigManager struct {
	configMu sync.RWMutex
	config   MonsterPipeAppConfig
	filePath string
}

func NewConfigManager(filePath string) (*ConfigManager, error) {
	var configManager = ConfigManager{
		configMu: sync.RWMutex{},
	}
	configManager.filePath = filePath
	if err := configManager.load(); err != nil {
		return nil, err
	}
	return &configManager, nil
}

func NewMemoryConfigManager() *ConfigManager {
	var configManager = ConfigManager{
		configMu: sync.RWMutex{},
	}
	return &configManager
}

func (c *ConfigManager) Save() (err error) {
	if err := c.check(); err != nil {
		return err
	}
	content, err := json.Marshal(&c.config)
	if err != nil {
		return err
	}
	c.configMu.Unlock()
	defer c.configMu.Unlock()
	return os.WriteFile(c.filePath, content, 0644)
}

func (c *ConfigManager) check() (err error) {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return nil
}

func (c *ConfigManager) load() (err error) {
	content, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}
	c.configMu.Lock()
	defer c.configMu.Unlock()
	return json.Unmarshal(content, &c.config)
}
