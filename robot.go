package main

import (
	"fmt"

	"github.com/opensourceways/community-robot-lib/config"
	framework "github.com/opensourceways/community-robot-lib/robot-gitee-framework"
	"github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

const botName = "version-freezer"

type iClient interface {
	AddPRLabel(org, repo string, number int32, label string) error
	CreatePRComment(org, repo string, number int32, comment string) error
	GetPathContent(org, repo, path, ref string) (gitee.Content, error)
	RemovePRLabel(org, repo string, number int32, label string) error
	RemovePRLabels(org, repo string, number int32, label []string) error
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(f framework.HandlerRegitster) {
	f.RegisterPullRequestHandler(bot.handlePREvent)
	f.RegisterNoteEventHandler(bot.handleNoteEvent)
}

func (bot *robot) handlePREvent(e *gitee.PullRequestEvent, c config.Config, log *logrus.Entry) error {
	org, repo := e.GetOrgRepo()

	cfg, err := bot.getConfig(c, org, repo)
	if err != nil {
		return err
	}

	if action := e.GetAction(); action != gitee.ActionOpen && action != "update" {
		return nil
	}

	return bot.handleCheckVersionFreeze(org, repo, e.GetPullRequest(), cfg, log)
}

func (bot *robot) handleNoteEvent(e *gitee.NoteEvent, c config.Config, log *logrus.Entry) error {
	if !e.IsPullRequest() || !e.IsPROpen() {
		return nil
	}

	org, repo := e.GetOrgRepo()

	cfg, err := bot.getConfig(c, org, repo)
	if err != nil {
		return err
	}

	return bot.handleParseCmd(e, cfg, log)
}
