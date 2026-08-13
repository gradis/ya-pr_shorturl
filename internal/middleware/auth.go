package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradis/ya-pr_shorturl/internal/auth"
)

func Authentication(secret string) gin.HandlerFunc {
	signingKey := []byte(secret)

	return func(c *gin.Context) {
		userID, err := userIDFromRequest(c.Request, signingKey)
		switch {
		case errors.Is(err, http.ErrNoCookie):
			userID, err = issueAuthenticationCookie(c.Writer, signingKey)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

		case err != nil:
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		requestContext := auth.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	}
}

func userIDFromRequest(request *http.Request, signingKey []byte) (string, error) {
	cookie, err := request.Cookie(auth.CookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", http.ErrNoCookie
		}
		return "", err
	}

	return auth.Verify(cookie.Value, signingKey)
}

func issueAuthenticationCookie(writer http.ResponseWriter, signingKey []byte) (string, error) {
	userID, err := auth.NewUserID()
	if err != nil {
		return "", err
	}

	token, err := auth.Sign(userID, signingKey)
	if err != nil {
		return "", err
	}

	http.SetCookie(writer, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return userID, nil
}
