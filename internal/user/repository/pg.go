package repository

import (
	"context"
	"database/sql"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/internal/user"
	"github.com/JamesHsu333/go-grpc/pkg/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

// User Repository
type userRepo struct {
	db *sqlx.DB
}

// Auth Repository constructor
func NewUserRepository(db *sqlx.DB) user.UserRepository {
	return &userRepo{db: db}
}

// Create new user
func (u *userRepo) Register(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.Register")
	defer span.Finish()

	createdUser := &models.User{}
	if err := u.db.QueryRowxContext(ctx, createUserQuery, &user.FirstName, &user.LastName, &user.Email,
		&user.Password, &user.Role, &user.About, &user.Avatar, &user.PhoneNumber, &user.Address, &user.City,
		&user.Gender, &user.Postcode, utils.ParseTimeFormat(user.Birthday),
	).StructScan(createdUser); err != nil {
		return nil, errors.Wrap(err, "userRepo.Register.StructScan")
	}

	return createdUser, nil
}

// Update existing user
func (u *userRepo) Update(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.Update")
	defer span.Finish()

	updatedUser := &models.User{}
	if err := u.db.GetContext(ctx, updatedUser, updateUserQuery, &user.FirstName, &user.LastName, &user.Email,
		&user.About, &user.Avatar, &user.PhoneNumber, &user.Address, &user.City, &user.Gender,
		&user.Postcode, utils.ParseTimeFormat(user.Birthday), &user.UserID,
	); err != nil {
		return nil, errors.Wrap(err, "userRepo.Update.GetContext")
	}

	return updatedUser, nil
}

// Delete existing user
func (u *userRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.Delete")
	defer span.Finish()

	result, err := u.db.ExecContext(ctx, deleteUserQuery, userID)
	if err != nil {
		return errors.WithMessage(err, "userRepo Delete ExecContext")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "userRepo.Delete.RowsAffected")
	}
	if rowsAffected == 0 {
		return errors.Wrap(sql.ErrNoRows, "userRepo.Delete.rowsAffected")
	}

	return nil
}

// Get user by id
func (u *userRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.GetByID")
	defer span.Finish()

	user := &models.User{}
	if err := u.db.QueryRowxContext(ctx, getUserQuery, userID).StructScan(user); err != nil {
		return nil, errors.Wrap(err, "userRepo.GetByID.QueryRowxContext")
	}

	return user, nil
}

// Find users by name
func (u *userRepo) FindByName(ctx context.Context, name string, pq *utils.PaginationQuery) (*models.UsersList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.FindByName")
	defer span.Finish()

	var totalCount int
	if err := u.db.GetContext(ctx, &totalCount, getTotalCount, name); err != nil {
		return nil, errors.Wrap(err, "userRepo.FindByName.GetContext.totalCount")
	}

	if totalCount == 0 {
		return &models.UsersList{
			TotalCount: totalCount,
			TotalPages: utils.GetTotalPages(totalCount, pq.GetSize()),
			Page:       pq.GetPage(),
			Size:       pq.GetSize(),
			HasMore:    utils.GetHasMore(pq.GetPage(), totalCount, pq.GetSize()),
			Users:      make([]*models.User, 0),
		}, nil
	}

	rows, err := u.db.QueryxContext(ctx, findUsers, name, pq.GetOffset(), pq.GetSize())
	if err != nil {
		return nil, errors.Wrap(err, "userRepo.FindByName.QueryxContext")
	}
	defer rows.Close()

	var users = make([]*models.User, 0, pq.GetSize())
	for rows.Next() {
		var user models.User
		if err = rows.StructScan(&user); err != nil {
			return nil, errors.Wrap(err, "userRepo.FindByName.StructScan")
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "userRepo.FindByName.rows.Err")
	}

	return &models.UsersList{
		TotalCount: totalCount,
		TotalPages: utils.GetTotalPages(totalCount, pq.GetSize()),
		Page:       pq.GetPage(),
		Size:       pq.GetSize(),
		HasMore:    utils.GetHasMore(pq.GetPage(), totalCount, pq.GetSize()),
		Users:      users,
	}, nil
}

// Find user by email
func (u *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.FindByEmail")
	defer span.Finish()

	foundUser := &models.User{}
	if err := u.db.QueryRowxContext(ctx, findUserByEmail, email).StructScan(foundUser); err != nil {
		return nil, errors.Wrap(err, "userRepo.FindByEmail.QueryRowxContext")
	}
	return foundUser, nil
}

// Get users with pagination
func (u *userRepo) GetUsers(ctx context.Context, pq *utils.PaginationQuery) (*models.UsersList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.GetUsers")
	defer span.Finish()

	var totalCount int
	if err := u.db.GetContext(ctx, &totalCount, getTotal); err != nil {
		return nil, errors.Wrap(err, "userRepo.GetUsers.GetContext.totalCount")
	}

	if totalCount == 0 {
		return &models.UsersList{
			TotalCount: totalCount,
			TotalPages: utils.GetTotalPages(totalCount, pq.GetSize()),
			Page:       pq.GetPage(),
			Size:       pq.GetSize(),
			HasMore:    utils.GetHasMore(pq.GetPage(), totalCount, pq.GetSize()),
			Users:      make([]*models.User, 0),
		}, nil
	}

	var users = make([]*models.User, 0, pq.GetSize())
	if err := u.db.SelectContext(
		ctx,
		&users,
		getUsers,
		pq.GetOrderBy(),
		pq.GetOffset(),
		pq.GetLimit(),
	); err != nil {
		return nil, errors.Wrap(err, "userRepo.GetUsers.SelectContext")
	}

	return &models.UsersList{
		TotalCount: totalCount,
		TotalPages: utils.GetTotalPages(totalCount, pq.GetSize()),
		Page:       pq.GetPage(),
		Size:       pq.GetSize(),
		HasMore:    utils.GetHasMore(pq.GetPage(), totalCount, pq.GetSize()),
		Users:      users,
	}, nil
}

// Update existing user role
func (u *userRepo) UpdateRole(ctx context.Context, user *models.User) (*models.User, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRepo.UpdateRole")
	defer span.Finish()

	updatedUser := &models.User{}
	if err := u.db.GetContext(ctx, updatedUser, updateUserRoleQuery, &user.Role, &user.UserID); err != nil {
		return nil, errors.Wrap(err, "userRepo.UpdateRole.GetContext")
	}
	return updatedUser, nil
}
