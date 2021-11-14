package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/internal/user"
	"github.com/JamesHsu333/go-grpc/pkg/grpc_errors"
	"github.com/JamesHsu333/go-grpc/pkg/logger"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
)

// Auth redis repository
type userRedisRepo struct {
	redisClient *redis.Client
	basePrefix  string
	logger      logger.Logger
}

// Auth redis repository constructor
func NewUserRedisRepo(redisClient *redis.Client, logger logger.Logger) user.RedisRepository {
	return &userRedisRepo{redisClient: redisClient, basePrefix: "user:", logger: logger}
}

// Get user by id
func (u *userRedisRepo) GetByIDCtx(ctx context.Context, key string) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepo.GetByIDCtx")
	defer span.Finish()

	userBytes, err := u.redisClient.Get(ctx, u.createKey(key)).Bytes()
	if err != nil {
		if err != redis.Nil {
			return nil, grpc_errors.ErrNotFound
		}
		return nil, err
	}
	user := &models.User{}
	if err = json.Unmarshal(userBytes, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Cache user with duration in seconds
func (u *userRedisRepo) SetUserCtx(ctx context.Context, key string, seconds int, user *models.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepo.SetUserCtx")
	defer span.Finish()

	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return u.redisClient.Set(ctx, u.createKey(key), userBytes, time.Second*time.Duration(seconds)).Err()
}

// Delete user by key
func (u *userRedisRepo) DeleteUserCtx(ctx context.Context, key string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepo.DeleteUserCtx")
	defer span.Finish()

	return u.redisClient.Del(ctx, u.createKey(key)).Err()
}

func (r *userRedisRepo) createKey(value string) string {
	return fmt.Sprintf("%s: %s", r.basePrefix, value)
}
