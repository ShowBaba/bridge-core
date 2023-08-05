package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func RunGQL(w http.ResponseWriter, r *http.Request, schema graphql.Schema, ctx context.Context) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Add("Access-Control-Allow-Headers", "Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userId, ok := r.Context().Value("id").(uint)
	if !ok {
		utils.Dispatch401Error(w, "Unauthorized access; id missing")
		return
	}

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

	ctx = context.WithValue(ctx, utils.KeyID, userId)

	if err := json.Unmarshal(body, &payload); err == nil {
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
