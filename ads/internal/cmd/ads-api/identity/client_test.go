package identity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/identity"
	commonmodel "github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
)

func TestClientImplementsUsersClientAndMapsUser(t *testing.T) {
	const userID = "00000000-0000-0000-0000-000000000001"
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"user_id": userID,
			"nickname": "alice",
		})
	}))
	defer server.Close()

	var client usersclient.UserClient = identity.NewClient(server.URL, time.Second)
	got, err := client.GetUserByID(context.Background(), commonmodel.UserID(userID))

	require.NoError(t, err)
	assert.Equal(t, "/v1/users/"+userID, gotPath)
	require.Equal(t, usersclient.User{UserID: commonmodel.UserID(userID), Nickname: "alice"}, got)
}
