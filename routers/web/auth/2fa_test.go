package auth

import (
	"net/http"
	"testing"

	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func Test2FARedirect(t *testing.T) {
	ctx, resp := contexttest.MockContext(t, "/user/two_factor")
	TwoFactor(ctx)
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "/user/login?redirect_to=%2Fuser%2Ftwo_factor", resp.Header().Get("location"))
	assert.Equal(t, "must be logged in to set 2fa", ctx.Flash.ErrorMsg)
}
