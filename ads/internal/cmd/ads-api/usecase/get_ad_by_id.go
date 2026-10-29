package usecase

import (
	"context"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
)

type (
	// GetAdByIDFunc returns a single ad. It fails with ErrorCodeNotFound when no ad matches the identifier.
	GetAdByIDFunc func(ctx context.Context, id domain.AdID) (domain.Ad, error)

	getAdByID struct {
		adDAO      adsdao.DAO
		userClient usersclient.UserClient
	}
)

// NewGetAdByIDFunc returns a GetAdByIDFunc backed by the given DAO and user client.
func NewGetAdByIDFunc(adDAO adsdao.DAO, userClient usersclient.UserClient) GetAdByIDFunc {
	return getAdByID{adDAO: adDAO, userClient: userClient}.getAdByID
}

func (uc getAdByID) getAdByID(ctx context.Context, id domain.AdID) (domain.Ad, error) {
	ad, err := uc.adDAO.GetAdByID(ctx, id)
	if err != nil {
		return domain.Ad{}, fmt.Errorf("unable to get ad %s: %w", id, err)
	}

	user, err := uc.userClient.GetUserByID(ctx, ad.UserID)
	if err != nil {
		return domain.Ad{}, fmt.Errorf("unable to get user %s for ad %s: %w", ad.UserID, id, err)
	}

	ad.Username = user.Nickname
	return ad, nil
}

// TODO: remove this code
// func (uc getAdByID) getAdByIDLegacy(ctx context.Context, id model.AdID) (model.Ad, error) {
// 	ads, err := uc.adDAO.SearchAds(ctx, adsdao.SearchAdsRequest{})
// 	if err != nil {
// 		return model.Ad{}, fmt.Errorf("unable to list ads: %w", err)
// 	}
//
// 	for _, ad := range ads {
// 		if ad.ID == id {
// 			return ad, nil
// 		}
// 	}
//
// 	return model.Ad{}, adsdao.ErrNotFound
// }
