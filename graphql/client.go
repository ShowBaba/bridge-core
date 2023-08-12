package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func RunGQL(w http.ResponseWriter, r *http.Request, schema graphql.Schema, ctx context.Context) {
	// Read the query
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.Dispatch400Error(w, "invalid request body: %s")
		return
	}

	var (
		payload GraphQLPayload
		resp    *graphql.Result
	)

	if err := json.Unmarshal(body, &payload); err == nil {
		if !strings.Contains(payload.Query, "__schema") {
			// not an introspection query, so add auth validator
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.Dispatch400Error(w, "auth token not in header")
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.Dispatch400Error(w, "bearer token not in header")
				return
			}
			claim, err := utils.ValidateAuthToken(parts[1], utils.GetConfig().JWTSecretKey)
			if err != nil {
				utils.Dispatch400Error(w, fmt.Sprintf("error validating auth token token: %v", err))
				return
			}
			ctx = context.WithValue(r.Context(), utils.KeyID, claim.ID)
		}
		resp = graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  payload.Query,
			VariableValues: payload.Variables,
			Context:        ctx,
		})
	} else {
		resp = graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: string(body),
			Context:       ctx,
		})
	}

	if len(resp.Errors) > 0 {
		utils.Dispatch400Error(w, fmt.Sprintf("%+v", resp.Errors))
		return
	}

	responseJSON(w, resp)
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
