package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"gitverse.ru/zagvozdeen/agenda/internal/models"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

type actorKey struct{}

func withActor(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, actorKey{}, user)
}

func actor(ctx context.Context) *models.User {
	u, _ := ctx.Value(actorKey{}).(*models.User)
	return u
}

func (a *API) authenticateToken(ctx context.Context, token string) (*models.User, error) {
	if !strings.HasPrefix(token, "agenda_") || len(token) != 71 {
		return nil, ErrInvalidToken
	}
	hash := sha256.Sum256([]byte(token))
	user, err := a.store.AuthenticateToken(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, store.ErrInvalidToken) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return user, nil
}
