package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestImportAds(t *testing.T) {
	ctx := context.Background()
	reqs := []usecase.CreateAdRequest{
		{Title: "bike", PriceCents: 5000, PhotoURL: "bike.jpg", OwnerID: "user-1"},
		{Title: "scooter", PriceCents: 8000, PhotoURL: "scooter.jpg", OwnerID: "user-1"},
	}

	t.Run("commits all ads together", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil, nil).Once()
		m.On("CreateAd", ctx, adsdao.CreateAdRequest{
			Title: "bike", PriceCents: 5000, PhotoURL: "bike.jpg", OwnerID: "user-1",
		}).Return(domain.AdID(10), nil).Once()
		m.On("CreateAd", ctx, adsdao.CreateAdRequest{
			Title: "scooter", PriceCents: 8000, PhotoURL: "scooter.jpg", OwnerID: "user-1",
		}).Return(domain.AdID(11), nil).Once()
		m.On("Commit").Return(nil).Once()

		ids, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.NoError(t, err)
		require.Equal(t, []domain.AdID{10, 11}, ids)
	})

	t.Run("rolls back the batch when an ad fails", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil, nil).Once()
		m.On("CreateAd", ctx, adsdao.CreateAdRequest{
			Title: "bike", PriceCents: 5000, PhotoURL: "bike.jpg", OwnerID: "user-1",
		}).Return(domain.AdID(10), nil).Once()
		createErr := errors.New("insert failed")
		m.On("CreateAd", ctx, adsdao.CreateAdRequest{
			Title: "scooter", PriceCents: 8000, PhotoURL: "scooter.jpg", OwnerID: "user-1",
		}).Return(domain.AdID(0), createErr).Once()
		m.On("Rollback").Return(nil).Once()

		ids, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.ErrorIs(t, err, createErr)
		require.Nil(t, ids)
	})

	t.Run("returns error when transaction cannot begin", func(t *testing.T) {
		m := mockdao.New(t)
		beginErr := errors.New("database unavailable")
		m.On("Begin", ctx).Return(nil, beginErr).Once()

		ids, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.ErrorIs(t, err, beginErr)
		require.Nil(t, ids)
	})
}
