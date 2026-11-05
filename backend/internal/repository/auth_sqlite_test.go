package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthRepositoryConsumesLoginStateOnceAndExpiresSessions(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "auth.db")
	database, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.OIDCLoginState{}, &model.AuthSession{}); err != nil {
		t.Fatal(err)
	}
	repo := NewAuthRepository(database)
	ctx := context.Background()
	now := time.Now().UTC()
	state := model.OIDCLoginState{
		StateHash: "state-hash", Nonce: "nonce", CodeVerifier: "verifier",
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := repo.SaveLoginState(ctx, state); err != nil {
		t.Fatal(err)
	}
	consumed, err := repo.ConsumeLoginState(ctx, state.StateHash, now)
	if err != nil || consumed.Nonce != state.Nonce {
		t.Fatalf("consumed=%+v err=%v", consumed, err)
	}
	if _, err := repo.ConsumeLoginState(ctx, state.StateHash, now); !errors.Is(err, ErrAuthRecordNotFound) {
		t.Fatalf("second consume error=%v, want missing state", err)
	}
	session := model.AuthSession{
		TokenHash: "session-hash", HeadscaleUserID: "hs-user-1",
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.CreateAuthSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDatabase.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo = NewAuthRepository(database)
	if restored, err := repo.AuthSession(ctx, session.TokenHash, now); err != nil || restored.HeadscaleUserID != session.HeadscaleUserID {
		t.Fatalf("session was not persisted after reopen: session=%+v err=%v", restored, err)
	}
	if err := repo.DeleteAuthSession(ctx, session.TokenHash); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AuthSession(ctx, session.TokenHash, now); !errors.Is(err, ErrAuthRecordNotFound) {
		t.Fatalf("revoked session error=%v, want missing session", err)
	}
	if err := repo.CreateAuthSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AuthSession(ctx, session.TokenHash, now.Add(2*time.Hour)); !errors.Is(err, ErrAuthRecordNotFound) {
		t.Fatalf("expired session error=%v, want missing session", err)
	}
	if err := repo.PurgeExpiredAuthRecords(ctx, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteAuthSession(ctx, session.TokenHash); err != nil {
		t.Fatal(err)
	}
}
