package cmd

import (
	"fmt"

	"github.com/iximiuz/labctl/api"
	"github.com/sagikazarmark/labx/internal/config"
)

type Cli struct {
	client *api.Client
}

func NewCli() *Cli {
	return &Cli{}
}

func (c *Cli) Init(cfg *config.Config, version string) {
	c.client = api.NewClient(api.ClientOptions{
		BaseURL:     cfg.BaseURL,
		APIBaseURL:  cfg.APIBaseURL,
		SessionID:   cfg.SessionID,
		AccessToken: cfg.AccessToken,
		UserAgent:   fmt.Sprintf("labx/%s", version),
	})
}
