package main

import (
	"encoding/json"
	"fmt"

	"github.com/Monster-Pipe/monster-pipe-core/pkg/utils"
)

type Config struct {
	Item  utils.ConfigItem[string] `json:"item"`
	Item2 utils.ConfigItem[int]    `json:"item2"`
	Item3 utils.ConfigItem[bool]   `json:"item3"`
}

func main() {
	config := &Config{}
	config.Item = utils.NewConfigItem("test")
	config.Item2 = utils.NewConfigItem(19)

	data, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	config2 := &Config{}
	json.Unmarshal(data, config2)
	fmt.Printf("%#v\n", config2.Item.Get())
	fmt.Printf("%#v\n", config2.Item2.Get())
	fmt.Printf("%#v\n", config2.Item3.Get())
}
