package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// DefaultMonths 是默认的统计月份数。
const DefaultMonths = 6

// Config 表示应用配置。
type Config struct {
	Months  int     `mapstructure:"months" yaml:"months"`   // 默认统计的月份数
	Email   string  `mapstructure:"email" yaml:"email"`     // 默认的邮箱过滤条件
	Aliases []Alias `mapstructure:"aliases" yaml:"aliases"` // 作者身份别名映射
}

// Alias 定义一个作者身份及其关联邮箱。
type Alias struct {
	Name   string   `mapstructure:"name" yaml:"name"`
	Emails []string `mapstructure:"emails" yaml:"emails"`
}

var (
	once     sync.Once
	instance *Config
	loadErr  error
)

// Dir 返回配置目录的路径 (~/.config/git-visible)。
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "git-visible"), nil
}

// File 返回配置文件的完整路径 (~/.config/git-visible/config.yaml)。
func File() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// EnsureDir 确保配置目录存在，如不存在则创建。
// 目录权限设置为 0700，仅允许所有者访问。
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o700)
}

// Load 返回单例配置实例，首次调用时从配置文件加载。
// 如果配置文件不存在，返回默认配置（months=6, email=""）。
func Load() (*Config, error) {
	once.Do(func() {
		configFile, err := File()
		if err != nil {
			loadErr = err
			return
		}

		v := viper.New()
		v.SetConfigFile(configFile)
		v.SetConfigType("yaml")
		v.SetDefault("months", DefaultMonths)
		v.SetDefault("email", "")

		if err := v.ReadInConfig(); err != nil {
			// 配置文件不存在时静默忽略，使用默认值
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
				loadErr = err
				return
			}
		}

		aliases := make([]Alias, 0)
		if err := v.UnmarshalKey("aliases", &aliases); err != nil {
			loadErr = err
			return
		}

		instance = &Config{
			Email:   v.GetString("email"),
			Months:  v.GetInt("months"),
			Aliases: aliases,
		}
	})

	return instance, loadErr
}

// Save 将配置保存到配置文件。
// 如果配置目录不存在，会自动创建。
func Save(config Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	configFile, err := File()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("email", config.Email)
	v.Set("months", config.Months)
	v.Set("aliases", config.Aliases)

	// 将配置写入文件（viper 默认 0644，需手动修正权限）
	if err := v.WriteConfigAs(configFile); err != nil {
		return err
	}
	if err := os.Chmod(configFile, 0o600); err != nil {
		return err
	}

	// 如果单例已加载，同步内存副本，避免后续读取到旧值。
	if instance != nil {
		*instance = config
	}
	return nil
}

// NormalizeEmail 根据 alias 配置将邮箱规范化为主邮箱（每组第一个邮箱）。
// 匹配逻辑大小写不敏感，并对输入及配置邮箱执行 TrimSpace。
// 若无匹配则返回原始输入值。
func (c *Config) NormalizeEmail(email string) string {
	if c == nil || len(c.Aliases) == 0 {
		return email
	}

	target := strings.TrimSpace(email)
	if target == "" {
		return email
	}

	for _, alias := range c.Aliases {
		if len(alias.Emails) == 0 {
			continue
		}

		primary := strings.TrimSpace(alias.Emails[0])
		for _, aliasEmail := range alias.Emails {
			if strings.EqualFold(target, strings.TrimSpace(aliasEmail)) {
				return primary
			}
		}
	}

	return email
}

// ValidateConfig 检查配置合法性，返回问题列表。
func ValidateConfig(cfg *Config) []string {
	if cfg == nil {
		return []string{"config is nil"}
	}

	var issues []string

	if cfg.Months <= 0 {
		issues = append(issues, fmt.Sprintf("months must be > 0, got %d", cfg.Months))
	}

	if cfg.Email != "" {
		email := strings.TrimSpace(cfg.Email)
		if !strings.Contains(email, "@") {
			issues = append(issues, fmt.Sprintf("invalid email format: %q", cfg.Email))
		}
	}

	return issues
}
