package user

import (
	"context"

	"github.com/JamesHsu333/go-grpc/internal/models"
)

type FileRepository interface {
	PutObject(ctx context.Context, input models.UploadInput) (*string, error)
	RemoveObject(ctx context.Context, fileName string) error
}
