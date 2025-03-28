package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig  `yaml:"server"`
	Default  DefaultConfig `yaml:"default"`
	Switches []Switch      `yaml:"switches"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DefaultConfig struct {
	Port           uint              `yaml:"port"`
	Username       string            `yaml:"username"`
	Password       string            `yaml:"password"`
	Labels         map[string]string `yaml:"labels"`
	SampleInterval int               `yaml:"sample_interval"`
}

type Switch struct {
	SampleInterval int               `yaml:"sample_interval"`
	Address        string            `yaml:"address"`
	Port           *uint             `yaml:"port,omitempty"`
	Username       *string           `yaml:"username,omitempty"`
	Password       *string           `yaml:"password,omitempty"`
	Labels         map[string]string `yaml:"labels,omitempty"`
}

type ResolvedSwitch struct {
	SampleInterval int
	Address        string
	Port           uint
	Username       string
	Password       string
	Labels         map[string]string
}

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "Path to YAML config file")
}

func LoadConfig() (*Config, []ResolvedSwitch, error) {
	flag.Parse()

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	if cfg.Default.Labels == nil {
		cfg.Default.Labels = make(map[string]string)
	}

	if cfg.Default.SampleInterval == 0 {
		cfg.Default.SampleInterval = 10
	}

	resolvedSwitches := make([]ResolvedSwitch, len(cfg.Switches))
	for i, sw := range cfg.Switches {
		resolved := ResolvedSwitch{
			SampleInterval: cfg.Default.SampleInterval,
			Address:        sw.Address,
			Port:           cfg.Default.Port,
			Username:       cfg.Default.Username,
			Password:       cfg.Default.Password,
			Labels:         make(map[string]string),
		}
		if sw.Port != nil {
			resolved.Port = *sw.Port
		}
		if sw.Username != nil {
			resolved.Username = *sw.Username
		}
		if sw.Password != nil {
			resolved.Password = *sw.Password
		}
		if sw.SampleInterval != 0 {
			resolved.SampleInterval = sw.SampleInterval
		}

		// 复制 default 的标签
		for k, v := range cfg.Default.Labels {
			resolved.Labels[k] = v
		}
		// 检查并覆盖标签
		for k, v := range sw.Labels {
			if _, exists := cfg.Default.Labels[k]; exists {
				resolved.Labels[k] = v
			} else {
				return nil, nil, fmt.Errorf("undefined label '%s' in switch %s; only labels defined in default are allowed", k, sw.Address)
			}
		}

		resolvedSwitches[i] = resolved
	}

	log.Printf("Loaded config from %s: server port=%d, %d switches", configFile, cfg.Server.Port, len(cfg.Switches))
	return &cfg, resolvedSwitches, nil
}
