package identity

import (
	"context"
	"time"

	commonmodel "github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

// Client adapts the identity HTTP client to the users package interface.
type Client struct {
	client *usersapi.Client
}

var _ usersclient.UserClient = (*Client)(nil)

// NewClient creates an identity client for the given service URL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{client: usersapi.NewClient(baseURL, timeout)}
}

// GetUserByID returns the user with the given ID.
func (c *Client) GetUserByID(ctx context.Context, userID commonmodel.UserID) (usersclient.User, error) {
	user, err := c.client.GetUserByID(ctx, userID.String())
	if err != nil {
		return usersclient.User{}, err
	}

	return usersclient.User{UserID: user.UserID, Nickname: user.Nickname}, nil
}

// ListUsers returns all users known to the identity service.
func (c *Client) ListUsers(ctx context.Context) ([]usersclient.User, error) {
	users, err := c.client.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]usersclient.User, 0, len(users))
	for _, user := range users {
		result = append(result, usersclient.User{UserID: user.UserID, Nickname: user.Nickname})
	}

	return result, nil
}
