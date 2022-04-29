package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

var (
	regCheckFreeze  = regexp.MustCompile(`(?mi)^/check-freeze\s*$`)
	regFreeze       = regexp.MustCompile(`(?mi)^/branch-freeze\s*$`)
	regFreezeCancel = regexp.MustCompile(`(?mi)^/branch-freeze cancel\s*$`)
)

func (bot *robot) handleParseCmd(e *gitee.NoteEvent, cfg *botConfig, log *logrus.Entry) error {
	comment := e.GetComment().GetBody()

	if regCheckFreeze.MatchString(comment) {
		org, repo := e.GetOrgRepo()

		return bot.handleCheckVersionFreeze(org, repo, e.GetPullRequest(), cfg, log)
	}

	if regFreeze.MatchString(comment) {
		return bot.handleFreezeCommand(e, cfg.FreezeFile, log)
	}

	if regFreezeCancel.MatchString(comment) {
		return bot.handleFreezeCancelCommand(e, cfg.FreezeFile, log)
	}

	return nil
}

func (bot *robot) handleFreezeCommand(e *gitee.NoteEvent, fc freezeConfig, log *logrus.Entry) error {
	if !bot.isValidCommand(e, fc, "/branch-freeze", log) {
		return nil
	}

	ls := e.GetPRLabelSet()
	org, repo := e.GetOrgRepo()

	if ls.Has(labelFreezeMergeable) {
		if err := bot.cli.RemovePRLabel(org, repo, e.GetPRNumber(), labelFreezeMergeable); err != nil {
			return err
		}
	}

	if !ls.Has(labelFreeze) {
		return bot.cli.AddPRLabel(org, repo, e.GetPRNumber(), labelFreeze)
	}

	return nil
}

func (bot *robot) handleFreezeCancelCommand(e *gitee.NoteEvent, fc freezeConfig, log *logrus.Entry) error {
	if !bot.isValidCommand(e, fc, "/branch-freeze cacel", log) {
		return nil
	}

	ls := e.GetPRLabelSet()
	org, repo := e.GetOrgRepo()

	if !ls.Has(labelFreezeMergeable) {
		return bot.cli.AddPRLabel(org, repo, e.GetPRNumber(), labelFreezeMergeable)
	}

	if ls.Has(labelFreeze) {
		if err := bot.cli.RemovePRLabel(org, repo, e.GetPRNumber(), labelFreeze); err != nil {
			return err
		}
	}

	return nil
}

func (bot *robot) isValidCommand(e *gitee.NoteEvent, fc freezeConfig, cmd string, log *logrus.Entry) bool {
	fi, err := bot.loadFreezeInfo(fc.Org, fc.Repo, fc.Branch, fc.FilePath)
	if err != nil {
		log.Error(err)

		return false
	}

	org, repo := e.GetOrgRepo()
	item := fi.getFreezeItem(org, repo)
	number := e.GetPRNumber()
	commenter := strings.ToLower(e.GetCommenter())

	if !item.branchIsFreeze(e.GetPullRequest().GetBase().GetRef()) {
		if err := bot.invalidCmdComment(org, repo, commenter, number); err != nil {
			log.Error(err)
		}

		return false
	}

	if !item.hasPermission(commenter) {
		if err := bot.noPremissCmdComment(org, repo, commenter, cmd, number); err != nil {
			log.Error(err)
		}

		return false
	}

	return true
}

func (bot *robot) invalidCmdComment(org, repo, commenter string, number int32) error {
	comment := fmt.Sprintf("@%s Invalid command: The target branch of this PR is not frozen.", commenter)

	return bot.cli.CreatePRComment(org, repo, number, comment)
}

func (bot *robot) noPremissCmdComment(org, repo, commenter, cmd string, number int32) error {
	comment := fmt.Sprintf("@%s you do not have permission to use the `%s` command.", commenter, cmd)

	return bot.cli.CreatePRComment(org, repo, number, comment)
}
