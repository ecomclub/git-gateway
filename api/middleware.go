package api

import (
	"context"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/git-gateway/models"
)

const (
	jwsSignatureHeaderName = "x-store-id"
)

type NetlifyMicroserviceClaims struct {
	SiteURL    string `json:"site_url"`
	InstanceID string `json:"id"`
	NetlifyID  string `json:"netlify_id"`
	jwt.StandardClaims
}

func (a *API) loadJWSSignatureHeader(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()
	signature := r.Header.Get(jwsSignatureHeaderName)
	if signature == "" {
		return nil, badRequestError("Operator microservice headers missing")
	}
	return withSignature(ctx, signature), nil
}

func (a *API) loadInstanceConfig(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	ctx := r.Context()

	signature := getSignature(ctx)
	if signature == "" {
		return nil, badRequestError("Operator signature missing")
	}

	storeID := signature
	if storeID == "" {
		return nil, badRequestError("Store ID is missing")
	}

	logEntrySetField(r, "store_id", storeID)
	instance, err := a.db.GetInstanceByStoreID(storeID)

	if err != nil {
		if models.IsNotFoundError(err) {
			return nil, notFoundError("Unable to locate site configuration")
		}
		return nil, internalServerError("Database error loading instance").WithInternalError(err)
	}

	config, err := instance.Config()
	if err != nil {
		return nil, internalServerError("Error loading environment config").WithInternalError(err)
	}

	ctx, err = WithInstanceConfig(ctx, config, instance.ID)
	if err != nil {
		return nil, internalServerError("Error loading instance config").WithInternalError(err)
	}

	return ctx, nil
}

func (a *API) verifyOperatorRequest(w http.ResponseWriter, req *http.Request) (context.Context, error) {
	c, _, err := a.extractOperatorRequest(w, req)
	return c, err
}

func (a *API) extractOperatorRequest(w http.ResponseWriter, req *http.Request) (context.Context, string, error) {
	token, err := a.extractBearerToken(w, req)
	if err != nil {
		return nil, token, err
	}
	if token == "" || token != a.config.OperatorToken {
		return nil, token, unauthorizedError("Request does not include an Operator token")
	}
	return req.Context(), token, nil
}
