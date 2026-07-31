package config

import (
	"time"

	"dns-failover/internal/model"
)

type SupportLink struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

type Support struct {
	Enabled bool          `yaml:"enabled"`
	Links   []SupportLink `yaml:"links"`
}

type Selectel struct {
	AccountID string `yaml:"account_id"`
	ProjectName string `yaml:"project_name"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

type TelegramConfig struct {
	Token  string `yaml:"token"`
	ChatID int64  `yaml:"chat_id"`
}

type Config struct {
	Interval time.Duration `yaml:"interval"`

	FailThreshold int `yaml:"fail_threshold"`

	SuccessThreshold int `yaml:"success_threshold"`

    Prometheus struct {
        URL string `yaml:"url"`
    } `yaml:"prometheus"`
	Selectel Selectel `yaml:"selectel"`
	Hosts []model.Host `yaml:"hosts"`
	Telegram TelegramConfig `yaml:"telegram"`
	Support Support `yaml:"support"`
}

