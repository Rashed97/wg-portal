package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/domain"
)

type UserManagerRepo interface {
	GetUser(ctx context.Context, id domain.UserIdentifier) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	DeleteUser(ctx context.Context, id domain.UserIdentifier) error
}

// WireGuardManagerRepo is the per-(user × interface) pool surface
// (BNet-m76e). Provided by the wireguard.Manager.
type WireGuardManagerRepo interface {
	GetUserInterfacePools(ctx context.Context, user domain.UserIdentifier) ([]domain.UserInterfacePool, error)
	SetUserInterfacePool(ctx context.Context, user domain.UserIdentifier, iface domain.InterfaceIdentifier, pool *domain.UserInterfacePool, skipRenumber bool) (*domain.UserInterfacePool, error)
	DeleteUserInterfacePool(ctx context.Context, user domain.UserIdentifier, iface domain.InterfaceIdentifier) error
}

type UserService struct {
	cfg *config.Config

	users UserManagerRepo
	wg    WireGuardManagerRepo
}

func NewUserService(cfg *config.Config, users UserManagerRepo, wg WireGuardManagerRepo) *UserService {
	return &UserService{
		cfg:   cfg,
		users: users,
		wg:    wg,
	}
}

func (s UserService) GetAll(ctx context.Context) ([]domain.User, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return nil, err
	}

	allUsers, err := s.users.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	return allUsers, nil
}

func (s UserService) GetById(ctx context.Context, id domain.UserIdentifier) (*domain.User, error) {
	if err := domain.ValidateUserAccessRights(ctx, id); err != nil {
		return nil, err
	}

	if s.cfg.Advanced.ApiAdminOnly && !domain.GetUserInfo(ctx).IsAdmin {
		return nil, errors.Join(errors.New("only admins can access this endpoint"), domain.ErrNoPermission)
	}

	user, err := s.users.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s UserService) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return nil, err
	}

	createdUser, err := s.users.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}

func (s UserService) Update(ctx context.Context, id domain.UserIdentifier, user *domain.User) (
	*domain.User,
	error,
) {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return nil, err
	}

	if id != user.Identifier {
		return nil, fmt.Errorf("user id mismatch: %s != %s: %w", id, user.Identifier, domain.ErrInvalidData)
	}

	updatedUser, err := s.users.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

func (s UserService) Delete(ctx context.Context, id domain.UserIdentifier) error {
	if err := domain.ValidateAdminAccessRights(ctx); err != nil {
		return err
	}

	err := s.users.DeleteUser(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// Per-(user × interface) pool delegations (BNet-m76e).

func (s UserService) GetUserInterfacePools(ctx context.Context, user domain.UserIdentifier) ([]domain.UserInterfacePool, error) {
	return s.wg.GetUserInterfacePools(ctx, user)
}

func (s UserService) SetUserInterfacePool(ctx context.Context, user domain.UserIdentifier, iface domain.InterfaceIdentifier, pool *domain.UserInterfacePool, skipRenumber bool) (*domain.UserInterfacePool, error) {
	return s.wg.SetUserInterfacePool(ctx, user, iface, pool, skipRenumber)
}

func (s UserService) DeleteUserInterfacePool(ctx context.Context, user domain.UserIdentifier, iface domain.InterfaceIdentifier) error {
	return s.wg.DeleteUserInterfacePool(ctx, user, iface)
}
