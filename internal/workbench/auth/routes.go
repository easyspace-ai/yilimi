package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type loginBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerBody struct {
	Username string `json:"username"`
	Contact  string `json:"contact"`
	Password string `json:"password"`
}

// RegisterRoutes mounts /me, /login, /register, /logout, /send-code（调用方应使用 Group("/api/v1/auth")）。
func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
	if svc == nil {
		panic("auth: nil service")
	}

	rg.GET("/me", func(c *gin.Context) {
		u := UserFromBearer(c.GetHeader("Authorization"))
		if u.ID == guestUser.ID && u.Role == guestUser.Role {
			c.JSON(http.StatusOK, gin.H{"user": nil, "loggedIn": false})
			return
		}
		fresh, err := svc.Me(c.Request.Context(), u.ID)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				c.JSON(http.StatusOK, gin.H{"user": nil, "loggedIn": false})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":     fresh,
			"loggedIn": true,
		})
	})

	rg.POST("/login", func(c *gin.Context) {
		var body loginBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		login := strings.TrimSpace(body.Login)
		if login == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "login and password required"})
			return
		}
		u, tok, err := svc.Login(c.Request.Context(), login, body.Password)
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":     u,
			"loggedIn": true,
			"token":    tok,
		})
	})

	rg.POST("/register", func(c *gin.Context) {
		var body registerBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if strings.TrimSpace(body.Username) == "" || strings.TrimSpace(body.Contact) == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username, contact and password required"})
			return
		}
		u, tok, err := svc.Register(c.Request.Context(), body.Username, body.Contact, body.Password)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrInvalidPassword), errors.Is(err, ErrInvalidContact):
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrDuplicateContact):
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			default:
				if isUniqueConstraint(err) {
					c.JSON(http.StatusConflict, gin.H{"error": "username or contact already in use"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
				return
			}
		}
		c.JSON(http.StatusCreated, gin.H{
			"user":     u,
			"loggedIn": true,
			"token":    tok,
		})
	})

	rg.POST("/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "loggedIn": false})
	})

	rg.POST("/send-code", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "mode": "disabled", "message": "verification not enabled on server"})
	})
}
