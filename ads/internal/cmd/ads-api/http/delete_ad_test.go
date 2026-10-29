package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

func newDeleteAdServer(fn usecase.DeleteAdFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("DELETE /ads/{adID}", middleware.RequireAuth(handlerDeleteAd(fn)))
	return httptest.NewServer(mux)
}

func TestHandlerDeleteAd(t *testing.T) {
	t.Run("returns 204 and passes ad ID and authenticated owner", func(t *testing.T) {
		ownerID := domain.UserID("user-42")
		srv := newDeleteAdServer(func(_ context.Context, id domain.AdID, owner domain.UserID) error {
			assert.Equal(t, domain.AdID(42), id)
			assert.Equal(t, ownerID, owner)
			return nil
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/ads/42", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", ownerID.String())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("returns 401 when Authorization header is missing", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ domain.UserID) error {
			t.Fatal("usecase should not be called")
			return nil
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/ads/42", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 400 for a non-numeric ad ID", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ domain.UserID) error {
			t.Fatal("usecase should not be called")
			return nil
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/ads/not-a-number", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "user-42")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 when ad is missing or belongs to another user", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ domain.UserID) error {
			return adsdao.ErrNotFound
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/ads/42", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "user-42")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 500 on unexpected usecase error", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ domain.AdID, _ domain.UserID) error {
			return errors.New("db down")
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/ads/42", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "user-42")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
