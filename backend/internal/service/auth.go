package service

import (
	"time"

	"github.com/luongtran/luna/backend/internal/auth"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// initAuth builds the account service and the middleware that resolves the signed-in user.
func (c *Container) initAuth() error {
	svc, err := auth.NewService(c.store.Users(), c.store.Sessions(), auth.NewLockout(time.Now), time.Now)
	if err != nil {
		return err
	}
	c.auth = svc
	c.authHandler = auth.NewHandler(svc, c.cfg.CookieSecure, c.log)
	c.requireAuth = httpx.RequireAuth(c.authHandler.ResolvePrincipal)
	return nil
}
