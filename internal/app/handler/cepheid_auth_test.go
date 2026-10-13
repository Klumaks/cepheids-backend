package handler

import (
	"cepheids-backend/internal/app/auth"
	"context"
	"errors"
	"github.com/gin-gonic/gin"

	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type unavailableCepheidBlacklist struct{}

func (unavailableCepheidBlacklist) IsRevoked(context.Context, string) (bool, error) {
	return false, errors.New("redis down")
}
func (unavailableCepheidBlacklist) Revoke(context.Context, string, time.Duration) error {
	return errors.New("redis down")
}
func TestCepheidAuthorizationAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens, _ := auth.NewCepheidTokens(strings.Repeat("a", 32), time.Hour, unavailableCepheidBlacklist{})
	token, _ := tokens.Issue(9)
	h := &Handler{CepheidTokens: tokens}
	r := gin.New()
	r.GET("/public", h.CepheidAuthorization(false), func(c *gin.Context) {
		if GetCurrentUserID(c) != 0 {
			t.Error("guest has user ID")
		}
		c.Status(200)
	})
	r.PUT("/private", h.CepheidAuthorization(true), func(c *gin.Context) { t.Error("protected handler executed"); c.Status(200) })
	for _, tc := range []struct {
		method, path, header string
		code                 int
	}{
		{"GET", "/public", "", 200}, {"GET", "/public", "Bearer broken", 200}, {"GET", "/public", "Bearer " + token, 200},
		{"PUT", "/private", "", 401}, {"PUT", "/private", "Bearer broken", 401}, {"PUT", "/private", "Bearer " + token, 503},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", tc.header)
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		if out.Code != tc.code {
			t.Fatalf("%s %s: got %d want %d", tc.method, tc.path, out.Code, tc.code)
		}
	}
}
