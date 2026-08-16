package user

import (
	"context"
	"flag"

	"github.com/google/subcommands"

	"github.com/yunomu/uploader/cmd/user/list"
)

type userCommand struct {
	commander *subcommands.Commander
}

func (c *userCommand) Name() string {
	return "user"
}

func (c *userCommand) Synopsis() string {
	return "user commands"
}

func (c *userCommand) Usage() string {
	return `user <subcommand> [args]
`
}

func (c *userCommand) SetFlags(f *flag.FlagSet) {
	commander := subcommands.NewCommander(f, "user")
	commander.Register(list.NewCommand(), "")
	c.commander = commander
}

func (c *userCommand) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	return c.commander.Execute(ctx, args...)
}

func NewCommand() subcommands.Command {
	return &userCommand{}
}
