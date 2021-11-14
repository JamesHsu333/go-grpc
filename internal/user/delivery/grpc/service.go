package grpc

import (
	"github.com/JamesHsu333/go-grpc/config"
	"github.com/JamesHsu333/go-grpc/internal/session"
	"github.com/JamesHsu333/go-grpc/internal/user"
	"github.com/JamesHsu333/go-grpc/pkg/logger"
	userProto "github.com/JamesHsu333/go-grpc/proto/user"
)

type usersService struct {
	logger logger.Logger
	cfg    *config.Config
	userUC user.UseCase
	sessUC session.UCSession
	userProto.UnimplementedUserServiceServer
}

// Auth service constructor
func NewUserServerGRPC(logger logger.Logger, cfg *config.Config, userUC user.UseCase, sessUC session.UCSession) *usersService {
	return &usersService{logger: logger, cfg: cfg, userUC: userUC, sessUC: sessUC}
}
