package handler

import (
	"cepheids-backend/internal/app/auth"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const cepheidUserKey = "cepheid_user_id"
const cepheidClaimsKey = "cepheid_token_claims"

func GetCurrentUserID(ctx *gin.Context) uint {
	id, _ := ctx.Get(cepheidUserKey)
	userID, _ := id.(uint)
	return userID
}
func (h *Handler) CepheidAuthorization(required bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		parts := strings.Fields(ctx.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			if required {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			ctx.Next()
			return
		}
		claims, id, err := h.CepheidTokens.Validate(ctx.Request.Context(), parts[1])
		if err != nil {
			if required {
				code := http.StatusUnauthorized
				if !errors.Is(err, auth.ErrInvalidToken) {
					code = http.StatusServiceUnavailable
				}
				ctx.AbortWithStatus(code)
				return
			}
			ctx.Next()
			return
		}
		ctx.Set(cepheidUserKey, id)
		ctx.Set(cepheidClaimsKey, claims)
		ctx.Next()
	}
}
