package usecase

import (
	"context"

	"github.com/JamesHsu333/go-grpc/config"
	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/internal/session"
	"github.com/opentracing/opentracing-go"
)

// Session use case
type SessionUC struct {
	sessionRepo session.SessRepository
	cfg         *config.Config
}

// New session use case constructor
func NewSessionUseCase(sessionRepo session.SessRepository, cfg *config.Config) session.UCSession {
	return &SessionUC{sessionRepo: sessionRepo, cfg: cfg}
}

// Create new session
func (u *SessionUC) CreateSession(ctx context.Context, session *models.Session, expire int) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessionUC.CreateSession")
	defer span.Finish()

	return u.sessionRepo.CreateSession(ctx, session, expire)
}

// Delete session by id
func (u *SessionUC) DeleteByID(ctx context.Context, sessionID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessionUC.DeleteByID")
	defer span.Finish()

	return u.sessionRepo.DeleteByID(ctx, sessionID)
}

// get session by id
func (u *SessionUC) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessionUC.GetSessionByID")
	defer span.Finish()

	return u.sessionRepo.GetSessionByID(ctx, sessionID)
}
