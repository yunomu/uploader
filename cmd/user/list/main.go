package list

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/yunomu/uploader/lib/userdb"
)

type Command struct {
	region    *string
	userTable *string
	nameIndex *string
}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) Name() string     { return "list" }
func (c *Command) Synopsis() string { return "list users" }
func (c *Command) Usage() string {
	return `list:
	list users
`
}

func (c *Command) SetFlags(f *flag.FlagSet) {
	c.region = f.String("region", "ap-northeast-1", "aws region")
	c.userTable = f.String("table", "", "user table")
	c.nameIndex = f.String("name-index", "", "user name index")
}

func (c *Command) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	cfg := args[0].(map[string]string)
	if *c.userTable == "" {
		*c.userTable = cfg["UserTable"]
	}
	if *c.nameIndex == "" {
		*c.nameIndex = cfg["UserNameIndex"]
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, func(opt *config.LoadOptions) error {
		opt.Region = *c.region
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "config.LoadDefaultConfig:", err)
		return subcommands.ExitFailure
	}

	db := userdb.NewDynamoDB(
		dynamodb.NewFromConfig(awsCfg),
		*c.userTable,
		*c.nameIndex,
	)

	users, err := db.List(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db.List:", err)
		return subcommands.ExitFailure
	}

	fmt.Println("ID\tName")
	for _, user := range users {
		fmt.Printf("%s\t%s\n", user.Id, user.Name)
	}

	return subcommands.ExitSuccess
}
