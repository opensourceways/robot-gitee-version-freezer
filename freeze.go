package main

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"
)

const (
	labelFreeze          = "branch-frozen"
	labelFreezeMergeable = "frozen-mergeable"
	freezeNotice         = "The target branch of this PR has been frozen. " +
		"If you want to merge, please invite the following peoples: @%s use the `/branch-freeze cancel` " +
		"command to indicate that the PR can be merged during the freezing period."
)

type freezeStatus int8

const (
	frozenUnknow         freezeStatus = 0
	frozenNoFLAndFML     freezeStatus = 1
	frozenNoFLHasFML     freezeStatus = 2
	frozenHasFLNoFML     freezeStatus = 3
	frozenHasFLAndFML    freezeStatus = 4
	notFrozenNoFLAndFML  freezeStatus = 5
	notFrozenNoFLHasFML  freezeStatus = 6
	notFrozenHasFLNoFML  freezeStatus = 7
	notFrozenHasFLAndFML freezeStatus = 8
)

type freezeInfo struct {
	FreezeItems []freezeItem `json:"freeze_items"`
}

func (fi *freezeInfo) getFreezeItem(org, repo string) *freezeItem {
	fp := fmt.Sprintf("%s/%s", org, repo)

	for _, v := range fi.FreezeItems {
		if v.Repo == fp {
			fi := v

			return &fi
		}
	}

	return nil
}

type freezeItem struct {
	Repo          string   `json:"repo"`
	FrozenBranchs []string `json:"frozen_branchs"`
	Owners        []string `json:"owners"`
}

func (it *freezeItem) branchIsFreeze(branch string) bool {
	if it == nil {
		return false
	}

	for _, v := range it.FrozenBranchs {
		if v == branch {
			return true
		}
	}

	return false
}

func (it *freezeItem) hasPermission(login string) bool {
	if it == nil {
		return false
	}

	for _, v := range it.Owners {
		if v == login {
			return true
		}
	}

	return false
}

func (it *freezeItem) getOwners() []string {
	if it == nil {
		return nil
	}

	return it.Owners
}

func (bot *robot) loadFreezeInfo(org, repo, branch, path string) (freezeInfo, error) {
	var fi freezeInfo

	content, err := bot.cli.GetPathContent(org, repo, path, branch)
	if err != nil {
		return fi, err
	}

	b, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return fi, err
	}

	err = yaml.Unmarshal(b, &fi)

	return fi, err
}

func (bot *robot) handleCheckVersionFreeze(
	org, repo string,
	pr *gitee.PullRequestHook,
	cfg *botConfig,
	log *logrus.Entry,
) error {
	fc := cfg.FreezeFile

	fi, err := bot.loadFreezeInfo(fc.Org, fc.Repo, fc.Branch, fc.FilePath)
	if err != nil {
		return err
	}

	item := fi.getFreezeItem(org, repo)
	fs := getFreezeStatus(item.branchIsFreeze(pr.GetBase().GetRef()), pr.LabelsToSet())

	return bot.doActionByFreezeStatus(org, repo, pr.GetNumber(), item.getOwners(), fs, log)
}

func (bot *robot) doActionByFreezeStatus(
	org, repo string,
	number int32,
	owners []string,
	status freezeStatus,
	log *logrus.Entry,
) error {
	switch status {
	case frozenHasFLNoFML, frozenNoFLHasFML, notFrozenNoFLAndFML:
		return nil
	case frozenHasFLAndFML:
		return bot.cli.RemovePRLabel(org, repo, number, labelFreeze)
	case frozenNoFLAndFML:
		return bot.addFreezeLabelAndNote(org, repo, number, owners)
	case notFrozenHasFLAndFML:
		return bot.cli.RemovePRLabels(org, repo, number, []string{labelFreeze, labelFreezeMergeable})
	case notFrozenHasFLNoFML:
		return bot.cli.RemovePRLabel(org, repo, number, labelFreeze)
	case notFrozenNoFLHasFML:
		return bot.cli.RemovePRLabel(org, repo, number, labelFreezeMergeable)
	default:
		log.Infof("unkonw pr's %s/%s:%d freeze status", org, repo, number)

		return nil
	}
}

func (bot *robot) addFreezeLabelAndNote(org, repo string, number int32, owners []string) error {
	err := bot.cli.AddPRLabel(org, repo, number, labelFreeze)
	if err != nil {
		return err
	}

	ownersStr := strings.Join(owners, " , @")
	comment := fmt.Sprintf(freezeNotice, ownersStr)

	return bot.cli.CreatePRComment(org, repo, number, comment)
}

func getFreezeStatus(isFreeze bool, labels sets.String) freezeStatus {
	if isFreeze {
		return getFrozendLabelStatus(labels)
	}

	return getUnFrozendLabelStatus(labels)
}

func getFrozendLabelStatus(labels sets.String) freezeStatus {
	if labels.Has(labelFreeze) && labels.Has(labelFreezeMergeable) {
		return frozenHasFLAndFML
	}

	if labels.Has(labelFreeze) && !labels.Has(labelFreezeMergeable) {
		return frozenHasFLNoFML
	}

	if !labels.Has(labelFreeze) && !labels.Has(labelFreezeMergeable) {
		return frozenNoFLAndFML
	}

	if !labels.Has(labelFreeze) && labels.Has(labelFreezeMergeable) {
		return frozenNoFLHasFML
	}

	return frozenUnknow
}

func getUnFrozendLabelStatus(labels sets.String) freezeStatus {
	if labels.Has(labelFreeze) && labels.Has(labelFreezeMergeable) {
		return notFrozenHasFLAndFML
	}

	if labels.Has(labelFreeze) && !labels.Has(labelFreezeMergeable) {
		return notFrozenHasFLNoFML
	}

	if !labels.Has(labelFreeze) && !labels.Has(labelFreezeMergeable) {
		return notFrozenNoFLAndFML
	}

	if !labels.Has(labelFreeze) && labels.Has(labelFreezeMergeable) {
		return notFrozenNoFLHasFML
	}

	return frozenUnknow
}
