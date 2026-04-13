package config

import (
	"context"

	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/util/gconv"
)

// Load 使用 go-frame 配置组件从指定文件加载配置。
func Load(path string) (*Config, error) {
	adapter, err := gcfg.NewAdapterFile(path)
	if err != nil {
		return nil, err
	}
	cfg := gcfg.NewWithAdapter(adapter)
	data, err := cfg.Data(context.Background())
	if err != nil {
		return nil, err
	}

	var result Config
	if err := gconv.Scan(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
