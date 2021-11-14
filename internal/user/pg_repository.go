package user

import (
	"context"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/pkg/utils"
	"github.com/google/uuid"
)

// User repository interface
type UserRepository interface {
	Register(ctx context.Context, user *models.User) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	FindByName(ctx context.Context, name string, p *utils.PaginationQuery) (*models.UsersList, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	GetUsers(ctx context.Context, p *utils.PaginationQuery) (*models.UsersList, error)
	UpdateRole(ctx context.Context, user *models.User) (*models.User, error)
}
