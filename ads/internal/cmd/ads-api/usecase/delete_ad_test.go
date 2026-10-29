package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestDeleteAd(t *testing.T) {
	ctx := context.Background()
	id := domain.AdID(42)
	ownerID := domain.UserID("00000000-0000-0000-0000-000000000001")

	t.Run("deletes the ad for its owner", func(t *testing.T) {
		m := mock.New(t)
		m.On("DeleteAd", ctx, id, ownerID).Return(nil).Once()

		err := usecase.NewDeleteAdFunc(m)(ctx, id, ownerID)

		require.NoError(t, err)
	})

	t.Run("returns not found when the ad is missing or belongs to another user", func(t *testing.T) {
		m := mock.New(t)
		m.On("DeleteAd", ctx, id, ownerID).Return(adsdao.ErrNotFound).Once()

		err := usecase.NewDeleteAdFunc(m)(ctx, id, ownerID)

		require.ErrorIs(t, err, adsdao.ErrNotFound)
	})

	t.Run("returns DAO errors", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("DeleteAd", ctx, id, ownerID).Return(dbErr).Once()

		err := usecase.NewDeleteAdFunc(m)(ctx, id, ownerID)

		require.ErrorIs(t, err, dbErr)
	})
}
