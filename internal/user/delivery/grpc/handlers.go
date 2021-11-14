package grpc

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/JamesHsu333/go-grpc/internal/models"
	"github.com/JamesHsu333/go-grpc/pkg/grpc_errors"
	"github.com/JamesHsu333/go-grpc/pkg/utils"
	userProto "github.com/JamesHsu333/go-grpc/proto/user"
)

// Register new user
func (u *usersService) Register(ctx context.Context, r *userProto.RegisterRequest) (*userProto.RegisterResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.Register")
	defer span.Finish()

	gender := r.GetGender()
	user := &models.User{
		Email:     r.GetEmail(),
		FirstName: r.GetFirstName(),
		LastName:  r.GetLastName(),
		Password:  r.GetFirstName(),
		Gender:    &gender,
	}

	if err := utils.ValidateStruct(ctx, user); err != nil {
		u.logger.Errorf("ValidateStruct: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "ValidateStruct: %v", err)
	}

	createdUser, err := u.userUC.Register(ctx, user)
	if err != nil {
		u.logger.Errorf("userUC.Register: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "Register: %v", err)
	}

	return &userProto.RegisterResponse{User: u.userModelToProto(createdUser)}, nil
}

// Login user with email and password
func (u *usersService) Login(ctx context.Context, r *userProto.LoginRequest) (*userProto.LoginResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.Login")
	defer span.Finish()

	email := r.GetEmail()

	if !utils.ValidateEmail(email) {
		u.logger.Errorf("ValidateEmail: %v", email)
		return nil, status.Errorf(codes.InvalidArgument, "ValidateEmail: %v", email)
	}

	user, err := u.userUC.Login(ctx, email, r.GetPassword())
	if err != nil {
		u.logger.Errorf("userUC.Login: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "Login: %v", err)
	}

	session, err := u.sessUC.CreateSession(ctx, &models.Session{
		UserID: user.UserID,
	}, u.cfg.Session.Expire)
	if err != nil {
		u.logger.Errorf("sessUC.CreateSession: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "sessUC.CreateSession: %v", err)
	}

	return &userProto.LoginResponse{User: u.userModelToProto(user), SessionId: session}, err
}

// Find user by uuid
func (u *usersService) GetUserByID(ctx context.Context, r *userProto.GetUserByIDRequest) (*userProto.GetUserByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.GetUserByID")
	defer span.Finish()

	userID, err := uuid.Parse(r.GetUserId())
	if err != nil {
		u.logger.Errorf("uuid.Parse: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "uuid.Parse: %v", err)
	}

	user, err := u.userUC.GetByID(ctx, userID)
	if err != nil {
		u.logger.Errorf("userUC.FindById: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.FindById: %v", err)
	}

	return &userProto.GetUserByIDResponse{User: u.userModelToProto(user)}, nil
}

// Get session id from, ctx metadata, find user by uuid and returns it
func (u *usersService) GetMe(ctx context.Context, r *userProto.GetMeRequest) (*userProto.GetMeResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.GetMe")
	defer span.Finish()

	sessID, err := u.getSessionIDFromCtx(ctx)
	if err != nil {
		u.logger.Errorf("getSessionIDFromCtx: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "sessUC.getSessionIDFromCtx: %v", err)
	}

	session, err := u.sessUC.GetSessionByID(ctx, sessID)
	if err != nil {
		u.logger.Errorf("sessUC.GetSessionByID: %v", err)
		if errors.Is(err, redis.Nil) {
			return nil, status.Errorf(codes.NotFound, "sessUC.GetSessionByID: %v", grpc_errors.ErrNotFound)
		}
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "sessUC.GetSessionByID: %v", err)
	}

	userUUID, err := uuid.Parse(session.UserID.String())
	if err != nil {
		u.logger.Errorf("uuid.Parse: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "uuid.Parse: %v", err)
	}

	user, err := u.userUC.GetByID(ctx, userUUID)
	if err != nil {
		u.logger.Errorf("userUC.FindById: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.FindById: %v", err)
	}

	return &userProto.GetMeResponse{User: u.userModelToProto(user)}, nil
}

// Logout user, delete current session
func (u *usersService) Logout(ctx context.Context, r *userProto.LogoutRequest) (*userProto.LogoutResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.Logout")
	defer span.Finish()

	sessID, err := u.getSessionIDFromCtx(ctx)
	if err != nil {
		u.logger.Errorf("getSessionIDFromCtx: %v", err)
		return nil, err
	}

	if err := u.sessUC.DeleteByID(ctx, sessID); err != nil {
		u.logger.Errorf("sessUC.DeleteByID: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "sessUC.DeleteByID: %v", err)
	}

	return &userProto.LogoutResponse{}, nil
}

// Update user, delete current user redis cache
func (u *usersService) Update(ctx context.Context, r *userProto.UpdateRequest) (*userProto.UpdateResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.Update")
	defer span.Finish()

	userID, err := uuid.Parse(r.User.GetUserId())
	if err != nil {
		u.logger.Errorf("uuid.Parse: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "uuid.Parse: %v", err)
	}

	about := r.User.GetAbout()
	avatar := r.User.GetAvatar()
	phoneNumber := r.User.GetPhoneNumber()
	address := r.User.GetAddress()
	city := r.User.GetCity()
	country := r.User.GetCountry()
	gender := r.User.GetGender()
	postcode := int(r.User.GetPostcode())
	birthday := r.User.GetBirthday().AsTime()

	user := &models.User{
		UserID:      userID,
		FirstName:   r.User.GetFirstName(),
		LastName:    r.User.GetLastName(),
		Email:       r.User.GetEmail(),
		About:       &about,
		Avatar:      &avatar,
		PhoneNumber: &phoneNumber,
		Address:     &address,
		City:        &city,
		Country:     &country,
		Gender:      &gender,
		Postcode:    &postcode,
		Birthday:    &birthday,
	}

	updatedUser, err := u.userUC.Update(ctx, user)

	if err != nil {
		u.logger.Errorf("userUC.Update: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.Update: %v", err)
	}

	return &userProto.UpdateResponse{User: u.userModelToProto(updatedUser)}, nil
}

// Update user's role, delete current user redis cache
func (u *usersService) UpdateRole(ctx context.Context, r *userProto.UpdateRoleRequest) (*userProto.UpdateRoleResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.UpdateRole")
	defer span.Finish()

	userID, err := uuid.Parse(r.User.GetUserId())
	if err != nil {
		u.logger.Errorf("uuid.Parse: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "uuid.Parse: %v", err)
	}

	role := r.User.GetRole()

	user := &models.User{
		UserID: userID,
		Role:   &role,
	}

	updatedUser, err := u.userUC.UpdateRole(ctx, user)

	if err != nil {
		u.logger.Errorf("userUC.UpdateRole: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.UpdateRole: %v", err)
	}

	return &userProto.UpdateRoleResponse{User: u.userModelToProto(updatedUser)}, nil
}

// Delete user account
func (u *usersService) Delete(ctx context.Context, r *userProto.DeleteRequest) (*userProto.DeleteResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.Delete")
	defer span.Finish()

	userID, err := uuid.Parse(r.GetUserId())
	if err != nil {
		u.logger.Errorf("uuid.Parse: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "uuid.Parse: %v", err)
	}

	err = u.userUC.Delete(ctx, userID)

	if err != nil {
		u.logger.Errorf("userUC.Delete: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.Delete: %v", err)
	}

	return &userProto.DeleteResponse{}, nil
}

// Find users by name
func (u *usersService) FindByName(ctx context.Context, r *userProto.FindByNameRequest) (*userProto.FindByNameResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.FindByName")
	defer span.Finish()

	var size int
	if r.Pagination.GetSize() == 0 {
		size = 10
	} else {
		size = int(r.Pagination.GetSize())
	}

	pq := &utils.PaginationQuery{
		Size:    size,
		Page:    int(r.Pagination.GetPage()),
		OrderBy: r.Pagination.GetOrderby(),
	}

	users, err := u.userUC.FindByName(ctx, r.GetName(), pq)

	if err != nil {
		u.logger.Errorf("userUC.FindByName: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.FindByName: %v", err)
	}

	return &userProto.FindByNameResponse{Users: u.userListModelToProto(users)}, nil
}

// Get users
func (u *usersService) GetUsers(ctx context.Context, r *userProto.GetUsersRequest) (*userProto.GetUsersResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "usersService.GetUsers")
	defer span.Finish()

	var size int
	if r.Pagination.GetSize() == 0 {
		size = 10
	} else {
		size = int(r.Pagination.GetSize())
	}

	pq := &utils.PaginationQuery{
		Size:    size,
		Page:    int(r.Pagination.GetPage()),
		OrderBy: r.Pagination.GetOrderby(),
	}

	users, err := u.userUC.GetUsers(ctx, pq)

	if err != nil {
		u.logger.Errorf("userUC.GetUsers: %v", err)
		return nil, status.Errorf(grpc_errors.ParseGRPCErrStatusCode(err), "userUC.GetUsers: %v", err)
	}

	return &userProto.GetUsersResponse{Users: u.userListModelToProto(users)}, nil
}

func (u *usersService) getSessionIDFromCtx(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext: %v", grpc_errors.ErrNoCtxMetaData)
	}

	sessionID := md.Get("session_id")
	if sessionID[0] == "" {
		return "", status.Errorf(codes.PermissionDenied, "md.Get sessionId: %v", grpc_errors.ErrInvalidSessionId)
	}

	return sessionID[0], nil
}

func (u *usersService) userModelToProto(user *models.User) *userProto.User {
	userProto := &userProto.User{
		UserId:      user.UserID.String(),
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Password:    user.Password,
		Role:        user.GetRole(),
		About:       user.GetAbout(),
		Avatar:      user.GetAvatar(),
		PhoneNumber: user.GetPhoneNumber(),
		Address:     user.GetAddress(),
		City:        user.GetCity(),
		Country:     user.GetCountry(),
		Gender:      user.GetGender(),
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
		LoginDate:   timestamppb.New(user.LoginDate),
	}

	if user.Birthday != nil {
		userProto.Birthday = timestamppb.New(*user.Birthday)
	}

	if user.Postcode != nil {
		userProto.Postcode = int32(*user.Postcode)
	}

	return userProto
}

func (u *usersService) userListModelToProto(users *models.UsersList) *userProto.UsersList {
	var usersList []*userProto.User

	for _, user := range users.Users {
		userProto := u.userModelToProto(user)
		usersList = append(usersList, userProto)
	}

	usersProto := &userProto.UsersList{
		TotalCount: int32(users.TotalCount),
		TotalPages: int32(users.TotalPages),
		Page:       int32(users.Page),
		Size:       int32(users.Size),
		HasMore:    users.HasMore,
		Users:      usersList,
	}

	return usersProto
}

/*
// UploadAvatar godoc
// @Summary Post avatar
// @Description Post user avatar image
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param file formData file true "Body with image file"
// @Param bucket query string true "aws s3 bucket" Format(bucket)
// @Param id path int true "user_id"
// @Success 200 {string} string	"ok"
// @Failure 500 {object} httpErrors.RestError
// @Router /auth/{id}/avatar [post]
func (h *AuthHandlers) UploadAvatar() echo.HandlerFunc {
	return func(c echo.Context) error {
		span, ctx := opentracing.StartSpanFromContext(utils.GetRequestCtx(c), "AuthHandlers.UploadAvatar")
		defer span.Finish()

		uID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}

		image, err := utils.ReadImage(c, "file")
		if err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}

		file, err := image.Open()
		if err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}
		defer file.Close()

		binaryImage := bytes.NewBuffer(nil)
		if _, err = io.Copy(binaryImage, file); err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}

		contentType, err := utils.CheckImageFileContentType(binaryImage.Bytes())
		if err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}

		reader := bytes.NewReader(binaryImage.Bytes())

		updatedUser, err := h.authUC.UploadAvatar(ctx, uID, models.UploadInput{
			File:        reader,
			Name:        image.Filename,
			Size:        image.Size,
			ContentType: contentType,
		})
		if err != nil {
			utils.LogResponseError(c, h.logger, err)
			return c.JSON(httpErrors.ErrorResponse(err))
		}

		return c.JSON(http.StatusOK, updatedUser)
	}
}
*/
