package repository

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/JamesHsu333/go-grpc/config"
	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/internal/user"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type userFileRepository struct {
	cfg *config.Config
}

func NewuserFileRepository(cfg *config.Config) user.FileRepository {
	return &userFileRepository{cfg: cfg}
}

func (f *userFileRepository) PutObject(ctx context.Context, input models.UploadInput) (*string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "userFileRepository.PutObject")
	defer span.Finish()

	filepath := path.Join(f.cfg.File.FilePath, f.generateFileName(input.Name))

	dst, err := os.Create(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "userFileRepository.FileUpload.PutObject.os.Create")
	}
	defer dst.Close()

	if _, err = io.Copy(dst, input.File); err != nil {
		return nil, errors.Wrap(err, "userFileRepository.FileUpload.PutObject.io.Copy")
	}
	return &filepath, nil
}

func (f *userFileRepository) RemoveObject(ctx context.Context, filepath string) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "userFileRepository.PutObject")
	defer span.Finish()

	err := os.Remove(filepath)
	if err != nil {
		return errors.Wrap(err, "userFileRepository.FileUpload.RemoveObject.os.Remove")
	}

	return nil
}

func (f *userFileRepository) generateFileName(fileName string) string {
	uid := uuid.New().String()
	return fmt.Sprintf("%s-%s", uid, fileName)
}
