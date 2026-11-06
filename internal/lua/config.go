package lua

import (
	"encoding/json"
	"os"
	"suffocat/internal/common"

	lua "github.com/yuin/gopher-lua"
)

type ConfigType int

const (
	TypeCheckbox ConfigType = iota
	TypeInput
	TypeSelect
	TypeMultiSelect
	TypeFightScript
	TypeNumberInput
)

type Config struct {
	Key     string     `json:"key,omitempty"`
	Index   int        `json:"index"`
	Type    ConfigType `json:"type"`
	Desc    string     `json:"desc"`
	Value   any        `json:"value"`
	Options []any      `json:"options,omitempty"`
}

type ConfigRegister struct {
	index     int
	keys      []string
	configMap map[string]Config
}

func NewConfigRegister() *ConfigRegister {
	return &ConfigRegister{
		index:     0,
		keys:      []string{},
		configMap: map[string]Config{},
	}
}

func (cr *ConfigRegister) register(key string, config Config) {
	if _, ok := cr.configMap[key]; !ok {
		cr.index++
		cr.keys = append(cr.keys, key)
	}

	config.Index = cr.index
	cr.configMap[key] = config
}

func (cr *ConfigRegister) AddCheckbox(key string, desc string, value bool) {
	cr.register(key, Config{
		Type:  TypeCheckbox,
		Desc:  desc,
		Value: value,
	})
}

func (cr *ConfigRegister) AddInput(key string, desc string, value string) {
	cr.register(key, Config{
		Type:  TypeInput,
		Desc:  desc,
		Value: value,
	})
}

func (cr *ConfigRegister) AddNumberInput(key string, desc string, value float64, options *lua.LTable) {
	var err error
	var config = Config{Type: TypeNumberInput, Desc: desc, Value: value}

	if config.Options, err = common.ToGoListAny(options); err != nil {
		return
	}

	cr.register(key, config)
}

func (cr *ConfigRegister) AddSelect(key string, desc string, value string, options *lua.LTable) {
	var err error
	var config = Config{Type: TypeSelect, Desc: desc, Value: value}

	if config.Options, err = common.ToGoListAny(options); err != nil {
		return
	}

	cr.register(key, config)
}

func (cr *ConfigRegister) AddMultiSelect(key string, desc string, value *lua.LTable, options *lua.LTable) {
	var err error
	var config = Config{Type: TypeMultiSelect, Desc: desc}

	if config.Value, err = common.ToGoList[string](value); err != nil {
		return
	}

	if config.Options, err = common.ToGoListAny(options); err != nil {
		return
	}

	cr.register(key, config)
}

func (cr *ConfigRegister) AddFightScript(key string, desc string) {
	cr.register(key, Config{
		Type:  TypeFightScript,
		Desc:  desc,
		Value: "",
	})
}

// OrderedConfigs 按 Add 调用顺序返回配置列表
func (cr *ConfigRegister) OrderedConfigs() []Config {
	result := make([]Config, 0, len(cr.keys))
	for _, k := range cr.keys {
		c := cr.configMap[k]
		c.Key = k
		result = append(result, c)
	}
	return result
}

// Configs 返回原始 map（兼容旧调用）
func (cr *ConfigRegister) Configs() map[string]Config {
	return cr.configMap
}

// Save 以有序数组格式保存配置到文件
func (cr *ConfigRegister) Save(path string) error {
	data, err := json.Marshal(cr.OrderedConfigs())
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load 加载配置（兼容新旧格式），将已保存的值合并回注册表中
func (cr *ConfigRegister) Load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// 尝试新格式：数组
	var arr []Config
	if json.Unmarshal(data, &arr) == nil && len(arr) > 0 && arr[0].Key != "" {
		for _, c := range arr {
			if existing, ok := cr.configMap[c.Key]; ok && existing.Type == c.Type {
				existing.Value = c.Value
				cr.configMap[c.Key] = existing
			}
		}
		return
	}

	// 回退到旧格式：map
	var configMap map[string]Config
	if json.Unmarshal(data, &configMap) != nil {
		return
	}

	for k, c := range configMap {
		if config, ok := cr.configMap[k]; ok && config.Type == c.Type {
			config.Value = c.Value
			cr.configMap[k] = config
		}
	}
}
