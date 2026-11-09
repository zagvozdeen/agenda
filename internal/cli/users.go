package cli

import (
	"encoding/json/v2"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

var usernamePattern = regexp.MustCompile("^[a-z0-9_]{3,64}$")

func (a *application) users() *cobra.Command {
	root := &cobra.Command{Use: "user", Short: "Manage users locally on the server"}
	create := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateUser(cmd, a.store, args[0])
		},
	}
	create.Flags().String("name", "", "Display name")
	_ = create.MarkFlagRequired("name")
	root.AddCommand(create, &cobra.Command{
		Use:   "list",
		Short: "List users, including disabled users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListUsers(cmd, a.store)
		},
	}, &cobra.Command{
		Use:   "disable <username>",
		Short: "Disable a user and invalidate all their tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return DisableUser(cmd, a.store, args[0])
		},
	})
	return root
}

func CreateUser(cmd *cobra.Command, storage store.Storage, username string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("username must contain 3 to 64 lowercase letters, digits or underscores")
	}
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return fmt.Errorf("name must contain 1 to 255 characters")
	}
	u, err := storage.CreateUser(cmd.Context(), username, name)
	if err != nil {
		return err
	}
	return json.MarshalWrite(cmd.OutOrStdout(), u)
}

func ListUsers(cmd *cobra.Command, storage store.Storage) error {
	users, err := storage.ListUsers(cmd.Context())
	if err != nil {
		return err
	}
	return json.MarshalWrite(cmd.OutOrStdout(), users)
}

func DisableUser(cmd *cobra.Command, storage store.Storage, username string) error {
	if err := storage.DisableUser(cmd.Context(), username); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "User disabled")
	return err
}
