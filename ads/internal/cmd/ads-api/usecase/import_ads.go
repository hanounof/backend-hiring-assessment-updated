package usecase

import (
	"context"
	"errors"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

// ImportAdsFunc creates a batch of ads atomically.
type ImportAdsFunc func(ctx context.Context, reqs []CreateAdRequest) ([]domain.AdID, error)

type importAds struct {
	adDAO adsdao.DAO
}

// NewImportAdsFunc returns an ImportAdsFunc backed by the given DAO.
func NewImportAdsFunc(adDAO adsdao.DAO) ImportAdsFunc {
	return importAds{adDAO: adDAO}.importAds
}

func (uc importAds) importAds(ctx context.Context, reqs []CreateAdRequest) ([]domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to begin ad import: %w", err)
	}

	ids := make([]domain.AdID, 0, len(reqs))
	for _, req := range reqs {
		id, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
			Title:      req.Title,
			PriceCents: req.PriceCents,
			PhotoURL:   req.PhotoURL,
			OwnerID:    req.OwnerID,
		})
		if err != nil {
			createErr := fmt.Errorf("unable to create ad during import: %w", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return nil, errors.Join(createErr, fmt.Errorf("unable to roll back ad import: %w", rollbackErr))
			}
			return nil, createErr
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(); err != nil {
		commitErr := fmt.Errorf("unable to commit ad import: %w", err)
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return nil, errors.Join(commitErr, fmt.Errorf("unable to roll back ad import: %w", rollbackErr))
		}
		return nil, commitErr
	}

	return ids, nil
}
