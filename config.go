package main

import (
	"errors"

	"github.com/opensourceways/community-robot-lib/config"
)

type configuration struct {
	ConfigItems []botConfig `json:"config_items,omitempty"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	v := make([]config.IRepoFilter, len(items))

	for i := range items {
		v[i] = &items[i]
	}

	if i := config.Find(org, repo, v); i >= 0 {
		return &items[i]
	}

	return nil
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	items := c.ConfigItems
	for i := range items {
		if err := items[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	Items := c.ConfigItems
	for i := range Items {
		Items[i].setDefault()
	}
}

type botConfig struct {
	config.RepoFilter
	FreezeFile freezeConfig `json:"freeze_file"`
}

func (c *botConfig) setDefault() {
	c.FreezeFile.setDefault()
}

func (c *botConfig) validate() error {
	if err := c.FreezeFile.validate(); err != nil {
		return err
	}

	return c.RepoFilter.Validate()
}

type freezeConfig struct {
	// Org the organization to which the version freeze profile belongs
	Org string `json:"org" required:"true"`
	// Repo the repository to which the version freeze profile belongs
	Repo string `json:"repo" required:"true"`
	// Branch the branch to which the version freeze profile belongs, default master
	Branch string `json:"branch,omitempty"`
	// FilePath freeze configuration file's path in the repository
	FilePath string `json:"file_path" required:"true"`
}

func (fc *freezeConfig) validate() error {
	if fc.Org == "" {
		return errors.New("the org configuration item can't be set empty")
	}

	if fc.Repo == "" {
		return errors.New("the repo configuration item can't be set empty")
	}

	if fc.FilePath == "" {
		return errors.New("the file_path configuration item can't be set empty")
	}

	return nil
}

func (fc *freezeConfig) setDefault() {
	if fc.Branch == "" {
		fc.Branch = "master"
	}
}
