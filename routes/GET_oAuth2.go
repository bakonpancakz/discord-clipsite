package routes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"shareclip/env"
	"shareclip/tools"
	"strings"

	"github.com/gin-gonic/gin"
)

// Allow the User to Login with Discord
func GET_oAuth2_Callback(c *gin.Context) {
	authCode := c.Request.URL.Query().Get("code")

	// No code provided, redirect user to Discord
	if authCode == "" {
		redirectTo := fmt.Sprintf(
			"https://discord.com/oauth2/authorize?response_type=code&client_id=%s&scope=identify&redirect_uri=%s",
			env.DISCORD_CLIENT_ID,
			url.QueryEscape(env.DISCORD_REDIRECT),
		)
		c.Redirect(http.StatusTemporaryRedirect, redirectTo)
		return
	}

	// Attempt to Receive Access Token
	var DiscordSession struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	{
		// Create and Send API Request
		body := url.Values{}
		body.Add("redirect_uri", env.DISCORD_REDIRECT)
		body.Add("grant_type", "authorization_code")
		body.Add("code", authCode)

		Request, err := http.NewRequest(
			"POST", "https://discord.com/api/oauth2/token",
			strings.NewReader(body.Encode()),
		)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		Request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		Request.Header.Add("Authorization", "Basic "+base64.URLEncoding.EncodeToString(
			[]byte(env.DISCORD_CLIENT_ID+":"+env.DISCORD_SECRET),
		))

		Response, err := http.DefaultClient.Do(Request)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer Response.Body.Close()

		// Decode API Response
		b, err := io.ReadAll(Response.Body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if Response.StatusCode != 200 {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf(
				"cannot retrieve access token, server responded with %s. (%s)",
				Response.Status,
				string(b),
			))
			return
		}
		if err := json.Unmarshal(b, &DiscordSession); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	// Fetch Discord User
	var DiscordUser struct {
		ID          string  `json:"id"`
		Username    string  `json:"username"`
		Displayname *string `json:"display_name"`
		Avatar      *string `json:"avatar"`
		MFAEnabled  bool    `json:"mfa_enabled"`
	}
	{
		// Create and Send API Request
		Request, err := http.NewRequest("GET", "https://discord.com/api/users/@me", http.NoBody)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		Request.Header.Add("Authorization", "Bearer "+DiscordSession.AccessToken)

		Response, err := http.DefaultClient.Do(Request)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer Response.Body.Close()

		// Decode API Response
		b, err := io.ReadAll(Response.Body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if Response.StatusCode != 200 {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf(
				"cannot retrieve user profile, server responded with %s. (%s)",
				Response.Status,
				string(b),
			))
			return
		}
		if err := json.Unmarshal(b, &DiscordUser); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	// Spam Prevention!
	if !DiscordUser.MFAEnabled {
		c.AbortWithStatusJSON(http.StatusBadRequest, "Your Discord account must have MFA Enabled")
		return
	}

	// Upsert Discord User
	userToken := tools.GenerateToken()
	userName := DiscordUser.Username
	if DiscordUser.Displayname != nil {
		userName = *DiscordUser.Displayname
	}
	_, err := env.DB.Exec(
		`INSERT INTO users 
			(id, avatar, name, token) VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET 
			avatar = $2, name = $3, token = $4`,
		DiscordUser.ID,
		DiscordUser.Avatar,
		userName,
		userToken,
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Redirect to Homepage
	c.SetCookie("session", userToken, env.COOKIE_LIFETIME, "/", "", env.TLS_ENABLED, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}
