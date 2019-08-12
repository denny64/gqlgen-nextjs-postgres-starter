package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/kichiyaki/graphql-starter/backend/middleware"

	"github.com/google/uuid"
	_authErrors "github.com/kichiyaki/graphql-starter/backend/auth/errors"
	_emailMock "github.com/kichiyaki/graphql-starter/backend/email/mocks"
	"github.com/kichiyaki/graphql-starter/backend/models"
	"github.com/kichiyaki/graphql-starter/backend/seed"
	_tokenMocks "github.com/kichiyaki/graphql-starter/backend/token/mocks"
	_userErrors "github.com/kichiyaki/graphql-starter/backend/user/errors"
	"github.com/kichiyaki/graphql-starter/backend/user/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSignup(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	mockUser := &seed.Users()[0]
	role := models.AdministrativeRole
	mockInput := models.UserInput{
		Login:    &mockUser.Login,
		Password: &mockUser.Password,
		Role:     &role,
		Email:    &mockUser.Email,
	}

	t.Run("user cannot be logged in", func(t *testing.T) {
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Signup(middleware.StoreUserInContext(context.Background(), mockUser), mockInput)
		require.Equal(t, _authErrors.ErrCannotCreateAccountWhileLoggedIn, err)
	})

	t.Run("login is occupied ", func(t *testing.T) {
		mockUserRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(_userErrors.ErrLoginIsOccupied).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Signup(context.TODO(), mockInput)
		require.Equal(t, _userErrors.ErrLoginIsOccupied, err)
	})

	t.Run("email is occupied ", func(t *testing.T) {
		mockUserRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(_userErrors.ErrEmailIsOccupied).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Signup(context.TODO(), mockInput)
		require.Equal(t, _userErrors.ErrEmailIsOccupied, err)
	})

	t.Run("token cannot be created", func(t *testing.T) {
		mockUserRepo.On("Store", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.Anything).Return(fmt.Errorf("Error")).Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Signup(context.TODO(), mockInput)
		require.Equal(t, _authErrors.ErrActivationTokenCannotBeCreated, err)
	})

	t.Run("success", func(t *testing.T) {
		mockUserRepo.On("Store", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.Anything).Return(nil).Once()
		mockEmail.On("Send", mock.Anything, mock.Anything).Return(nil).Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		user, err := usecase.Signup(context.TODO(), mockInput)
		require.Equal(t, nil, err)
		require.Equal(t, mockUser.Login, user.Login)
		require.Equal(t, mockUser.Email, user.Email)
	})
}

func TestLogin(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	mockUser := &seed.Users()[0]

	t.Run("user cannot be logged in", func(t *testing.T) {
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Login(middleware.StoreUserInContext(context.Background(), mockUser), mockUser.Login, mockUser.Password)
		require.Equal(t, _authErrors.ErrCannotLoginWhileLoggedIn, err)
	})

	t.Run("success", func(t *testing.T) {
		mockUserRepo.On("GetByCredentials", mock.Anything, mockUser.Login, mockUser.Password).Return(mockUser, nil).Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		user, err := usecase.Login(context.Background(), mockUser.Login, mockUser.Password)
		require.Equal(t, nil, err)
		require.Equal(t, user.Login, mockUser.Login)
	})
}

func TestLogout(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	mockUser := &seed.Users()[0]

	t.Run("user cannot be logged out", func(t *testing.T) {
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.Logout(context.Background())
		require.Equal(t, _authErrors.ErrNotLoggedIn, err)
	})

	t.Run("success", func(t *testing.T) {
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.Logout(middleware.StoreUserInContext(context.Background(), mockUser))
		require.Equal(t, nil, err)
	})
}

func TestActivate(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	users := seed.Users()
	tokens := seed.Tokens()

	t.Run("user account is activated", func(t *testing.T) {
		id := users[0].ID
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[0], nil).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		token := uuid.New().String()
		_, err := usecase.Activate(context.TODO(), id, token)
		require.Equal(t, _authErrors.ErrAccountHasBeenActivated, err)
	})

	t.Run("no token found", func(t *testing.T) {
		id := users[1].ID
		tok := tokens[0]
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Activate(context.TODO(), id, tok.Value)
		require.Equal(t, _authErrors.ErrInvalidActivationToken, err)
	})

	t.Run("cannot update user", func(t *testing.T) {
		id := users[1].ID
		tok := tokens[0]
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockUserRepo.
			On("Update",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(fmt.Errorf("error")).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		_, err := usecase.Activate(context.TODO(), id, tok.Value)
		require.Equal(t, _authErrors.ErrAccountCannotBeActivated, err)
	})

	t.Run("success", func(t *testing.T) {
		id := users[1].ID
		tok := tokens[0]
		users[1].Activated = false
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockTokenRepo.
			On("Delete",
				mock.Anything,
				[]int{id}).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockUserRepo.
			On("Update",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(nil).
			Once()
		mockTokenRepo.On("Delete", mock.Anything, []int{tok.ID}).Return([]*models.Token{&tok}, nil).Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		user, err := usecase.Activate(context.TODO(), id, tok.Value)
		require.Equal(t, nil, err)
		require.Equal(t, true, user.Activated)
	})
}

func TestGenerateNewActivationToken(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	users := seed.Users()

	t.Run("user account is activated", func(t *testing.T) {
		id := users[0].ID
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[0], nil).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewActivationToken(context.TODO(), id)
		require.Equal(t, _authErrors.ErrAccountHasBeenActivated, err)
	})

	t.Run("the token limit has been reached", func(t *testing.T) {
		id := users[1].ID
		tokens := seed.Tokens()
		fetchedTokens := []*models.Token{}
		for i := 0; i < limitOfActivationTokens+1; i++ {
			token := tokens[0]
			fetchedTokens = append(fetchedTokens, &token)
		}
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return(fetchedTokens, nil).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewActivationToken(context.TODO(), id)
		require.Equal(t, _authErrors.ErrReachedLimitOfActivationTokens, err)
	})

	t.Run("cannot create token", func(t *testing.T) {
		id := users[1].ID
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil)
		mockTokenRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.Token")).
			Return(fmt.Errorf("Error")).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewActivationToken(context.TODO(), id)
		require.Equal(t, _authErrors.ErrActivationTokenCannotBeCreated, err)
	})

	t.Run("success", func(t *testing.T) {
		id := users[1].ID
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil)
		mockTokenRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.Token")).
			Return(nil).
			Once()
		mockEmail.
			On("Send", mock.Anything, mock.Anything).
			Return(nil).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewActivationToken(context.TODO(), id)
		require.Equal(t, nil, err)
	})
}

func TestGenerateNewResetPasswordToken(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	users := seed.Users()
	email := users[0].Email

	t.Run("the token limit has been reached", func(t *testing.T) {
		tokens := seed.Tokens()
		fetchedTokens := []*models.Token{}
		for i := 0; i < limitOfActivationTokens+1; i++ {
			token := tokens[0]
			fetchedTokens = append(fetchedTokens, &token)
		}
		mockUserRepo.
			On("GetByEmail",
				mock.Anything,
				email).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return(fetchedTokens, nil).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewResetPasswordToken(context.TODO(), email)
		require.Equal(t, _authErrors.ErrReachedLimitOfResetPasswordTokens, err)
	})

	t.Run("cannot create token", func(t *testing.T) {
		mockUserRepo.
			On("GetByEmail",
				mock.Anything,
				email).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil)
		mockTokenRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.Token")).
			Return(fmt.Errorf("Error")).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewResetPasswordToken(context.TODO(), email)
		require.Equal(t, _authErrors.ErrResetPasswordTokenCannotBeCreated, err)
	})

	t.Run("success", func(t *testing.T) {
		mockUserRepo.
			On("GetByEmail",
				mock.Anything,
				email).
			Return(&users[1], nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil)
		mockTokenRepo.
			On("Store",
				mock.Anything,
				mock.AnythingOfType("*models.Token")).
			Return(nil).
			Once()
		mockEmail.
			On("Send", mock.Anything, mock.Anything).
			Return(nil).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.GenerateNewResetPasswordToken(context.TODO(), email)
		require.Equal(t, nil, err)
	})
}

func TestResetPassword(t *testing.T) {
	mockUserRepo := new(mocks.Repository)
	mockTokenRepo := new(_tokenMocks.Repository)
	mockEmail := new(_emailMock.Email)
	users := seed.Users()
	tokens := seed.Tokens()
	id := users[0].ID

	t.Run("no token found", func(t *testing.T) {
		tok := tokens[1]
		user := users[0]
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&user, nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{}, nil).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.ResetPassword(context.TODO(), id, tok.Value)
		require.Equal(t, _authErrors.ErrInvalidResetPasswordToken, err)
	})

	t.Run("cannot update user", func(t *testing.T) {
		tok := tokens[1]
		user := users[0]
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&user, nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockUserRepo.
			On("Update",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(fmt.Errorf("error")).
			Once()
		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.ResetPassword(context.TODO(), id, tok.Value)
		require.Equal(t, fmt.Errorf(_userErrors.UserCannotBeUpdatedErrFormatWithLogin, user.Login), err)
	})

	t.Run("success", func(t *testing.T) {
		tok := tokens[1]
		user := users[0]
		mockUserRepo.
			On("GetByID",
				mock.Anything,
				id).
			Return(&user, nil).
			Once()
		mockTokenRepo.
			On("Fetch",
				mock.Anything,
				mock.AnythingOfType("*pgfilter.Filter")).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockUserRepo.
			On("Update",
				mock.Anything,
				mock.AnythingOfType("*models.User")).
			Return(nil).
			Once()
		mockTokenRepo.
			On("Delete", mock.Anything, []int{tok.ID}).
			Return([]*models.Token{&tok}, nil).
			Once()
		mockEmail.
			On("Send", mock.Anything, mock.Anything).
			Return(nil).
			Once()

		usecase := NewAuthUsecase(mockUserRepo, mockTokenRepo, mockEmail)
		err := usecase.ResetPassword(context.TODO(), id, tok.Value)
		require.Equal(t, nil, err)
	})
}
