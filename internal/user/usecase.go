//go:generate mockgen -source usecase.go -destination mock/usecase_mock.go -package mock
package user

import (
	"context"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/pkg/utils"
	"github.com/google/uuid"
)

// Auth repository interface
type UseCase interface {
	Register(ctx context.Context, user *models.User) (*models.User, error)
	Login(ctx context.Context, email string, password string) (*models.User, error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	UpdateRole(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, userID uuid.UUID) error
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	FindByName(ctx context.Context, name string, pq *utils.PaginationQuery) (*models.UsersList, error)
	GetUsers(ctx context.Context, pq *utils.PaginationQuery) (*models.UsersList, error)
	//UploadAvatar(ctx context.Context, userID uuid.UUID, file models.UploadInput) (*models.User, error)
}
