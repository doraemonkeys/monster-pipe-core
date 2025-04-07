package utils

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"
)

// A ConfigItemReader must not be copied after first use.
type ConfigItemReader[T any] struct {
	itemLock sync.RWMutex
	item     T
	// if notNil is false, make sure to set the item to default value of T
	notNil bool
}

func NewConfigItemReader[T any](item T) ConfigItemReader[T] {
	if reflect.TypeFor[T]().Kind() == reflect.Ptr {
		panic("item must not be a pointer")
	}
	return ConfigItemReader[T]{item: item, itemLock: sync.RWMutex{}, notNil: true}
}

func (c *ConfigItemReader[T]) Get() T {
	c.itemLock.RLock()
	defer c.itemLock.RUnlock()
	return c.item
}

func (c *ConfigItemReader[T]) GetNil() (T, bool) {
	c.itemLock.RLock()
	defer c.itemLock.RUnlock()
	if !c.notNil {
		return c.item, true
	}
	return c.item, false
}

func (c *ConfigItemReader[T]) IsNil() bool {
	c.itemLock.RLock()
	defer c.itemLock.RUnlock()
	return !c.notNil
}

func (c *ConfigItemReader[T]) MarshalJSON() ([]byte, error) {
	c.itemLock.RLock()
	defer c.itemLock.RUnlock()
	if !c.notNil {
		return json.Marshal(nil)
	}
	return json.Marshal(c.item)
}

func (c *ConfigItemReader[T]) UnmarshalJSON(data []byte) error {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()
	c.notNil = true
	if bytes.Equal(data, []byte("null")) {
		c.notNil = false
		c.item = *new(T)
		return nil
	}
	return json.Unmarshal(data, &c.item)
}

// A ConfigItem must not be copied after first use.
type ConfigItem[T any] struct {
	ConfigItemReader[T]
}

func NewConfigItem[T any](item T) ConfigItem[T] {
	return ConfigItem[T]{NewConfigItemReader(item)}
}

func (c *ConfigItem[T]) Set(item T) {
	c.itemLock.Lock()
	c.item = item
	c.notNil = true
	c.itemLock.Unlock()
}

func (c *ConfigItem[T]) SetNil() {
	c.itemLock.Lock()
	c.notNil = false
	c.item = *new(T)
	c.itemLock.Unlock()
}

func (c *ConfigItem[T]) GetReader() *ConfigItemReader[T] {
	return &c.ConfigItemReader
}
