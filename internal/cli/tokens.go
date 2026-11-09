package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/spf13/cobra"
	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func (a *application) tokens() *cobra.Command {
	root := &cobra.Command{Use: "token", Short: "Manage personal API tokens locally on the server"}
	create := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a token and print its secret once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return CreateToken(cmd, a.store, args[0])
		},
	}
	create.Flags().String("name", "", "Token name")
	create.Flags().String("expires-at", "", "Optional expiration in RFC3339 format")
	root.AddCommand(create, &cobra.Command{
		Use:   "list <username>",
		Short: "List token metadata without token secrets or hashes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListTokens(cmd, a.store, args[0])
		},
	}, &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke a personal API token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RevokeToken(cmd, a.store, args[0])
		},
	})
	return root
}
func CreateToken(cmd *cobra.Command, storage store.Storage, username string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	expires, err := cmd.Flags().GetString("expires-at")
	if err != nil {
		return err
	}
	var expiresAt *time.Time
	if expires != "" {
		value, err := time.Parse(time.RFC3339, expires)
		if err != nil {
			return fmt.Errorf("invalid expiration date: %w", err)
		}
		expiresAt = &value
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > 255 {
		return fmt.Errorf("token name must not exceed 255 characters")
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return fmt.Errorf("token expiration must be in the future")
	}
	user, err := storage.GetUserByUsername(cmd.Context(), username)
	if err != nil {
		return err
	}
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	secret := "agenda_" + hex.EncodeToString(random)
	hash := sha256.Sum256([]byte(secret))
	token := &models.APIToken{UserID: user.ID, TokenHash: hex.EncodeToString(hash[:]), ExpiresAt: expiresAt}
	if name != "" {
		token.Name = &name
	}
	if err = storage.CreateToken(cmd.Context(), token); err != nil {
		return err
	}
	return json.MarshalWrite(cmd.OutOrStdout(), struct {
		ID        uuid.UUID  `json:"id"`
		Token     string     `json:"token"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}{token.ID, secret, token.ExpiresAt})
}

func ListTokens(cmd *cobra.Command, storage store.Storage, username string) error {
	user, err := storage.GetUserByUsername(cmd.Context(), username)
	if err != nil {
		return err
	}
	tokens, err := storage.ListTokens(cmd.Context(), user.ID)
	if err != nil {
		return err
	}
	return json.MarshalWrite(cmd.OutOrStdout(), tokens)
}

func RevokeToken(cmd *cobra.Command, storage store.Storage, tokenID string) error {
	id, err := uuid.Parse(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	if err = storage.RevokeToken(cmd.Context(), id); err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), "Token revoked")
	return err
}
