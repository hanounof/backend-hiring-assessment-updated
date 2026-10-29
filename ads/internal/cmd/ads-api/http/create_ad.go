package http

import (
	"encoding/json"
	"net/http"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

type createAdBody struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	PhotoURL string  `json:"photo_url"`
}

type createAdResponse struct {
	ID int64 `json:"id"`
}

// handlerCreateAd serves POST /ads.
func handlerCreateAd(createAd usecase.CreateAdFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body createAdBody
		if err := decodeBody(r, &body); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		priceCents, err := priceToCents(body.Price)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		adID, err := createAd(r.Context(), usecase.CreateAdRequest{
			Title:      body.Title,
			PriceCents: priceCents,
			PhotoURL:   body.PhotoURL,
			OwnerID:    middleware.UserIDFromHeader(r),
		})
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(createAdResponse{ID: int64(adID)})
	}
}
