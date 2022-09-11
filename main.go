package main

import (
	"context"
	"fmt"
	"gacha-simulator/handler"
	"gacha-simulator/job"
	"gacha-simulator/model"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/errors"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/google/uuid"
	secrets "github.com/ijustfool/docker-secrets"
	"github.com/joho/godotenv"
	oauth2gorm "src.techknowlogick.com/oauth2-gorm"
)

var dockerSecrets *secrets.DockerSecrets

func main() {
	dsn := getDSN()

	model.SetupDB(dsn)

	manager := manage.NewDefaultManager()

	clientStore := oauth2gorm.NewClientStore(oauth2gorm.NewConfig(dsn, oauth2gorm.MySQL, "oauth2_clients"))
	ctx := context.Background()
	clientStore.Create(ctx, &models.Client{
		ID:     os.Getenv("OAUTH_PUBLIC_CLIENT_ID"),
		Domain: os.Getenv("OAUTH_PUBLIC_CLIENT_DOMAIN"),
	})
	clientStore.Create(ctx, &models.Client{
		ID:     getSecretWithhEnvFallback("oauth_private_client_id", "OAUTH_PRIVATE_CLIENT_ID"),
		Secret: getSecretWithhEnvFallback("oauth_private_client_secret", "OAUTH_PRIVATE_CLIENT_SECRET"),
	})
	manager.MapClientStorage(clientStore)

	tokenStore := oauth2gorm.NewTokenStore(oauth2gorm.NewConfig(dsn, oauth2gorm.MySQL, "oauth2_token"), 600)
	manager.MapTokenStorage(tokenStore)

	srv := server.NewServer(&server.Config{
		TokenType:                   "Bearer",
		AllowGetAccessRequest:       false,
		AllowedResponseTypes:        []oauth2.ResponseType{oauth2.Code},
		AllowedGrantTypes:           []oauth2.GrantType{oauth2.AuthorizationCode, oauth2.Refreshing, oauth2.ClientCredentials},
		AllowedCodeChallengeMethods: []oauth2.CodeChallengeMethod{oauth2.CodeChallengeS256},
		ForcePKCE:                   true,
	}, manager)
	srv.SetClientInfoHandler(func(r *http.Request) (string, string, error) {
		clientID := r.Form.Get("client_id")
		if clientID == "" {
			username, password, ok := r.BasicAuth()
			if !ok {
				return "", "", errors.ErrInvalidClient
			}
			return username, password, nil
		}
		clientSecret := r.Form.Get("client_secret")
		return clientID, clientSecret, nil
	})
	srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
		uuid, err := uuid.NewRandom()
		return uuid.String(), err
	})

	job.InitJobs()

	ginEngine := gin.Default()
	apiGroup := ginEngine.Group("/api")
	{
		apiGroup.GET("/authorize", func(ctx *gin.Context) {
			err := srv.HandleAuthorizeRequest(ctx.Writer, ctx.Request)
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			ctx.Abort()
		})
		apiGroup.POST("/token", func(ctx *gin.Context) {
			err := srv.HandleTokenRequest(ctx.Writer, ctx.Request)
			if err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
			ctx.Abort()
		})
		validateBearerToken := func(ctx *gin.Context) {
			ti, err := srv.ValidationBearerToken(ctx.Request)
			if err != nil {
				ctx.AbortWithError(500, err)
				return
			}
			ctx.Set("access_token", ti)
			ctx.Next()
		}
		gameTitlesGroup := apiGroup.Group("/game-titles")
		{
			gameTitlesGroup.Use(validateBearerToken)
			gameTitlesGroup.GET("", handler.GetGameTitles)
			gameTitlesGroup.GET("/:gameTitleSlug", handler.GetGameTitle)
			gameTitlesGroup.GET("/:gameTitleSlug/tiers", handler.GetTiers)
			gameTitlesGroup.GET("/:gameTitleSlug/items", handler.GetItems)
			gameTitlesGroup.GET("/:gameTitleSlug/pricings", handler.GetPricings)
			gameTitlesGroup.GET("/:gameTitleSlug/policies", handler.GetPolicies)
			gameTitlesGroup.GET("/:gameTitleSlug/plans", handler.GetPlans)
			gameTitlesGroup.GET("/:gameTitleSlug/gachas", handler.GetGachas)
		}
		gachasGroup := apiGroup.Group("/gachas")
		{
			gachasGroup.Use(validateBearerToken)
			gachasGroup.POST("", handler.PostGachas)
			gachasGroup.GET("/:resultID", handler.GetGacha)
			gachasGroup.PATCH("/:resultID", handler.PatchGacha)
			gachasGroup.DELETE("/:resultID", handler.DeleteGacha)
		}
		adminGroup := apiGroup.Group("/admin")
		{
			adminGroup.Use(func(ctx *gin.Context) {
				ti, err := srv.ValidationBearerToken(ctx.Request)
				if err != nil {
					ctx.AbortWithError(500, err)
					return
				}
				clientID := ti.(oauth2.TokenInfo).GetClientID()
				privateClientID := getSecretWithhEnvFallback("oauth_private_client_id", "OAUTH_PRIVATE_CLIENT_ID")
				if clientID != privateClientID {
					ctx.AbortWithStatus(403)
					return
				}
				ctx.Next()
			})
			adminGroup.POST("game-titles-bulk", handler.PostGameTitlesBulk)
			adminGroup.DELETE("game-titles/:gameTitleSlug", handler.DeleteGameTitle)
		}
	}
	ginEngine.Run()
}

func getDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		getSecretWithhEnvFallback("db_user", "DB_USER"),
		getSecretWithhEnvFallback("db_password", "DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		getSecretWithhEnvFallback("db_name", "DB_NAME"),
	)
}

func getSecretWithhEnvFallback(secretKey, envKey string) string {
	if secretValue, _ := dockerSecrets.Get(secretKey); secretValue != "" {
		return secretValue
	} else {
		return os.Getenv(envKey)
	}
}

func init() {
	env := os.Getenv("GACHA_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")
	if "test" != env {
		godotenv.Load(".env.local")
	}
	godotenv.Load(".env." + env)
	godotenv.Load()

	dockerSecrets, _ = secrets.NewDockerSecrets("")
}
