package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	mockusers "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/mock"
)

func TestGetAdByID(t *testing.T) {
	ctx := context.Background()

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		want := domain.Ad{ID: 42, Title: "bike", Price: 5000, UserID: "user-1"}
		users := mockusers.New(t)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(want, nil).Once()
		users.On("GetUserByID", ctx, want.UserID).Return(usersclient.User{
			UserID: want.UserID, Nickname: "alice",
		}, nil).Once()

		got, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 42)

		require.NoError(t, err)
		want.Username = "alice"
		assert.Equal(t, want, got)
	})

	t.Run("error no ad found", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("GetAdByID", ctx, domain.AdID(99)).Return(domain.Ad{}, adsdao.ErrNotFound).Once()

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 99)

		require.Error(t, err)
		assert.ErrorIs(t, err, adsdao.ErrNotFound)
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("GetAdByID", ctx, domain.AdID(1)).Return(domain.Ad{}, dbErr).Once()

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 1)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("error from user service", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		ad := domain.Ad{ID: 42, UserID: "user-1"}
		userErr := errors.New("identity service unavailable")
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(ad, nil).Once()
		users.On("GetUserByID", ctx, ad.UserID).Return(usersclient.User{}, userErr).Once()

		_, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 42)

		require.ErrorIs(t, err, userErr)
	})
}
