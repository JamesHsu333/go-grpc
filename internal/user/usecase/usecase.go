package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/internal/user"
	"github.com/JamesHsu333/go-grpc/pkg/grpc_errors"
	"github.com/JamesHsu333/go-grpc/pkg/logger"
	"github.com/JamesHsu333/go-grpc/pkg/utils"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	basePrefix            = "api-user:"
	userByIdCacheDuration = 3600
)

// Auth UseCase
type userUC struct {
	userRepo  user.UserRepository
	redisRepo user.RedisRepository
	//fileRepo  user.FileRepository
	logger logger.Logger
}

// Auth UseCase constructor
func NewUserUC(userRepo user.UserRepository, redisRepo user.RedisRepository, log logger.Logger) user.UseCase {
	return &userUC{userRepo: userRepo, redisRepo: redisRepo, logger: log}
}

// Create new user
func (u *userUC) Register(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.Register")
	defer span.Finish()

	existsUser, err := u.userRepo.FindByEmail(ctx, user.Email)
	if existsUser != nil || err == nil {
		return nil, grpc_errors.ErrEmailExists
	}

	createdUser, err := u.userRepo.Register(ctx, user)
	if err != nil {
		return nil, err
	}
	createdUser.SanitizePassword()

	return createdUser, nil
}

// Update existing user
func (u *userUC) Update(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.Update")
	defer span.Finish()

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Password = strings.TrimSpace(user.Password)

	updatedUser, err := u.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	updatedUser.SanitizePassword()

	if err = u.redisRepo.DeleteUserCtx(ctx, u.generateUserKey(user.UserID.String())); err != nil {
		u.logger.Errorf("userUC.Update.DeleteUserCtx: %s", err)
	}

	updatedUser.SanitizePassword()

	return updatedUser, nil
}

// Delete new user
func (u *userUC) Delete(ctx context.Context, userID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.Delete")
	defer span.Finish()

	if err := u.userRepo.Delete(ctx, userID); err != nil {
		return err
	}

	if err := u.redisRepo.DeleteUserCtx(ctx, u.generateUserKey(userID.String())); err != nil {
		u.logger.Errorf("userUC.Delete.DeleteUserCtx: %s", err)
	}

	return nil
}

// Get user by id
func (u *userUC) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.GetByID")
	defer span.Finish()

	cachedUser, err := u.redisRepo.GetByIDCtx(ctx, u.generateUserKey(userID.String()))

	if err != nil {
		u.logger.Errorf("userUC.redisRepo.GetByIDCtx: %v", err)
	}
	if cachedUser != nil {
		return cachedUser, nil
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err = u.redisRepo.SetUserCtx(ctx, u.generateUserKey(userID.String()), userByIdCacheDuration, user); err != nil {
		u.logger.Errorf("userUC.redisRepo.SetUserCtx: %v", err)
	}

	user.SanitizePassword()

	return user, nil
}

// Find users by name
func (u *userUC) FindByName(ctx context.Context, name string, pq *utils.PaginationQuery) (*models.UsersList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.FindByName")
	defer span.Finish()

	return u.userRepo.FindByName(ctx, name, pq)
}

// Get users with pagination
func (u *userUC) GetUsers(ctx context.Context, pq *utils.PaginationQuery) (*models.UsersList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.GetUsers")
	defer span.Finish()

	return u.userRepo.GetUsers(ctx, pq)
}

// Login user, returns user model with jwt token
func (u *userUC) Login(ctx context.Context, email string, password string) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.Login")
	defer span.Finish()

	foundUser, err := u.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, errors.Wrap(err, "userRepo.FindByEmail")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(password)); err != nil {
		return nil, errors.Wrap(err, "user.ComparePasswords")
	}

	foundUser.SanitizePassword()

	return foundUser, nil
}

// Update user role
func (u *userUC) UpdateRole(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.UpdateRole")
	defer span.Finish()

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	user.Password = strings.TrimSpace(user.Password)

	updatedUser, err := u.userRepo.UpdateRole(ctx, user)
	if err != nil {
		return nil, err
	}

	updatedUser.SanitizePassword()

	if err = u.redisRepo.DeleteUserCtx(ctx, u.generateUserKey(user.UserID.String())); err != nil {
		u.logger.Errorf("userUC.UpdateRole.DeleteUserCtx: %s", err)
	}

	updatedUser.SanitizePassword()

	return updatedUser, nil
}

/*
// Upload user avatar
func (u *userUC) UploadAvatar(ctx context.Context, userID uuid.UUID, file models.UploadInput) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userUC.UploadAvatar")
	defer span.Finish()

	filepath, err := u.fileRepo.PutObject(ctx, file)
	if err != nil {
		return nil, httpErrors.NewInternalServerError(errors.Wrap(err, "userUC.UploadAvatar.PutObject"))
	}

	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Avatar != nil {
		err := u.fileRepo.RemoveObject(ctx, *user.Avatar)
		if err != nil {
			return nil, err
		}
	}

	updatedUser, err := u.userRepo.Update(ctx, &models.User{
		UserID: userID,
		Avatar: filepath,
	})

	if err != nil {
		return nil, err
	}

	updatedUser.SanitizePassword()

	if err = u.redisRepo.DeleteUserCtx(ctx, u.GenerateUserKey(userID.String())); err != nil {
		u.logger.Errorf("userUC.UploadAvatar.DeleteUserCtx: %s", err)
	}

	updatedUser.SanitizePassword()

	return updatedUser, nil
}*/
func (u *userUC) generateUserKey(userID string) string {
	return fmt.Sprintf("%s: %s", basePrefix, userID)
}
