package graphql

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	logPkg "github.com/showbaba/query-bridge/bridge-core/log"
	log "github.com/showbaba/query-bridge/bridge-core/logger"

	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/endpoint"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, logRepo logPkg.Repository) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"dashboard": &graphql.Field{
				Type: DashboardType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					userID, ok := p.Context.Value("id").(string)
					if !ok || userID == "" {
						return nil, errors.New("unauthorized access")
					}

					now := time.Now().UTC()

					var appCount, dbCount, epCount int64
					db.Model(&application.Application{}).Where("user_id = ?", userID).Count(&appCount)
					db.Model(&database.Database{}).Where("user_id = ?", userID).Count(&dbCount)
					db.Model(&endpoint.Endpoint{}).Where("user_id = ?", userID).Count(&epCount)

					subQ := db.Model(&endpoint.Endpoint{}).Select("id").Where("user_id = ?", userID)

					curStart := now.Add(-7 * 24 * time.Hour)
					prevStart := now.Add(-14 * 24 * time.Hour)

					var curReqs, prevReqs int64
					db.Model(&endpoint.EndpointStats{}).
						Where("created_at >= ? AND created_at < ? AND endpoint_id IN (?)", curStart, now, subQ).
						Count(&curReqs)
					db.Model(&endpoint.EndpointStats{}).
						Where("created_at >= ? AND created_at < ? AND endpoint_id IN (?)", prevStart, curStart, subQ).
						Count(&prevReqs)

					deltaPercent, trend := calcDelta(curReqs, prevReqs)

					healthStart := now.Add(-24 * time.Hour)

					var total, total5xx int64
					db.Model(&endpoint.EndpointStats{}).
						Where("created_at >= ? AND endpoint_id IN (?)", healthStart, subQ).
						Count(&total)
					db.Model(&endpoint.EndpointStats{}).
						Where("response_status_code >= 500 AND created_at >= ? AND endpoint_id IN (?)", healthStart, subQ).
						Count(&total5xx)

					var avgLat sql.NullFloat64
					db.Model(&endpoint.EndpointStats{}).
						Select("AVG(execution_time_ms)").
						Where("created_at >= ? AND endpoint_id IN (?)", healthStart, subQ).
						Scan(&avgLat)

					apiUptimePercent := 100.0
					errorRatePercent := 0.0
					if total > 0 {
						apiUptimePercent = float64(total-total5xx) * 100.0 / float64(total)
						errorRatePercent = float64(total5xx) * 100.0 / float64(total)
					}

					avgLatencyMs := 0.0
					if avgLat.Valid {
						avgLatencyMs = avgLat.Float64
					}

					// Percentiles (whole window)
					var p95, p99 sql.NullFloat64
					_ = db.Raw(`
			SELECT
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p95,
				PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p99
			FROM endpoint_stats
			WHERE created_at >= ? AND endpoint_id IN (?)`,
						healthStart, subQ,
					).Row().Scan(&p95, &p99)

					// Reliability breakdown
					var x4xx, x5xx int64
					db.Model(&endpoint.EndpointStats{}).
						Where("response_status_code BETWEEN 400 AND 499 AND created_at >= ? AND endpoint_id IN (?)", healthStart, subQ).
						Count(&x4xx)
					db.Model(&endpoint.EndpointStats{}).
						Where("response_status_code >= 500 AND created_at >= ? AND endpoint_id IN (?)", healthStart, subQ).
						Count(&x5xx)

					// Top endpoints (by RPM, last 60m)
					type topRow struct {
						Path         string
						Rpm          float64
						AvgLatencyMs float64
					}
					var topEndpoints []topRow
					db.Raw(`
			SELECT e.path,
			       COUNT(es.id)::float / 60.0 AS rpm,
			       COALESCE(AVG(es.execution_time_ms),0) AS avg_latency_ms
			FROM endpoints e
			JOIN endpoint_stats es ON es.endpoint_id = e.id
			WHERE es.created_at >= ? AND e.user_id = ?
			GROUP BY e.path
			ORDER BY rpm DESC
			LIMIT 5`, now.Add(-60*time.Minute), userID).Scan(&topEndpoints)

					// Slowest endpoints (by avg latency, last 24h)
					type slowRow struct {
						Path         string
						AvgLatencyMs float64
						P95LatencyMs sql.NullFloat64
					}
					var slowest []slowRow
					_ = db.Raw(`
			WITH base AS (
			  SELECT e.path, es.execution_time_ms
			  FROM endpoints e
			  JOIN endpoint_stats es ON es.endpoint_id = e.id
			  WHERE es.created_at >= ? AND e.user_id = ?
			)
			SELECT path,
			       AVG(execution_time_ms) AS avg_latency_ms,
			       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p95_latency_ms
			FROM base
			GROUP BY path
			ORDER BY avg_latency_ms DESC
			LIMIT 5`, healthStart, userID).Scan(&slowest)

					// Latency timeseries (last 60 min, 1-min buckets)
					type ltRow struct {
						Bucket time.Time
						P50    sql.NullFloat64
						P95    sql.NullFloat64
						P99    sql.NullFloat64
					}
					var lt []ltRow
					tsStart := now.Add(-60 * time.Minute)
					// Postgres version. If your DB doesn’t support percentile functions, you can replace with AVG as a fallback.
					err := db.Raw(`
			SELECT
			  DATE_TRUNC('minute', created_at) AS bucket,
			  PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p50,
			  PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p95,
			  PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY execution_time_ms)::float AS p99
			FROM endpoint_stats
			WHERE created_at >= ? AND endpoint_id IN (?)
			GROUP BY 1
			ORDER BY 1 ASC`, tsStart, subQ).Scan(&lt).Error
					if err != nil {
						// Fallback if percentile_cont unsupported: use AVG for p50 and leave others nil
						type avgRow struct {
							Bucket time.Time
							Avg    sql.NullFloat64
						}
						var avgs []avgRow
						_ = db.Raw(`
				SELECT DATE_TRUNC('minute', created_at) AS bucket,
				       AVG(execution_time_ms)::float AS avg
				FROM endpoint_stats
				WHERE created_at >= ? AND endpoint_id IN (?)
				GROUP BY 1
				ORDER BY 1 ASC`, tsStart, subQ).Scan(&avgs)
						lt = make([]ltRow, 0, len(avgs))
						for _, r := range avgs {
							lt = append(lt, ltRow{Bucket: r.Bucket, P50: r.Avg})
						}
					}
					series := make([]map[string]any, 0, len(lt))
					for _, r := range lt {
						var p50v, p95v, p99v *float64
						if r.P50.Valid {
							v := r.P50.Float64
							p50v = &v
						}
						if r.P95.Valid {
							v := r.P95.Float64
							p95v = &v
						}
						if r.P99.Valid {
							v := r.P99.Float64
							p99v = &v
						}
						series = append(series, map[string]any{
							"ts":  r.Bucket.UTC().Format(time.RFC3339),
							"p50": p50v,
							"p95": p95v,
							"p99": p99v,
						})
					}

					displayName := "You"
					var u struct {
						FirstName sql.NullString
						LastName  sql.NullString
						AvatarURL sql.NullString
					}
					db.Raw(`SELECT first_name, last_name, avatar_url FROM users WHERE id = ?`, userID).Scan(&u)
					if u.FirstName.Valid || u.LastName.Valid {
						displayName = strings.TrimSpace(u.FirstName.String + " " + u.LastName.String)
					}

					resp := map[string]any{
						"lastUpdatedISO": now.Format(time.RFC3339),
						"user": map[string]any{
							"displayName": displayName,
							"avatarUrl":   nullOrStr(u.AvatarURL),
						},
						"summary": map[string]any{
							"applications": map[string]any{"count": appCount},
							"databases":    map[string]any{"count": dbCount},
							"endpoints":    map[string]any{"count": epCount},
							"api_requests": map[string]any{
								"count":        curReqs,
								"deltaPercent": roundToTwoDecimalPlaces(deltaPercent),
								"deltaWindow":  "7d",
								"trend":        trend,
							},
						},
						"performance": map[string]any{
							"reliability": map[string]any{
								"successRatePercent": roundToTwoDecimalPlaces((100.0-errorRatePercent)*10) / 10,
								"errorRatePercent":   roundToTwoDecimalPlaces(errorRatePercent*10) / 10,
								"errorBreakdown": map[string]any{
									"x4xx": x4xx,
									"x5xx": x5xx,
								},
							},
							"latency": map[string]any{
								"average":       roundIf(avgLatencyMs),
								"p95":           nullFloatToAny(p95),
								"p99":           nullFloatToAny(p99),
								"unit":          "ms",
								"window":        "last_60m",
								"bucketSizeSec": 60,
								"timeseries":    series,
							},
							"topEndpoints": func() []map[string]any {
								out := make([]map[string]any, 0, len(topEndpoints))
								for _, r := range topEndpoints {
									out = append(out, map[string]any{
										"path":         r.Path,
										"rpm":          math.Ceil(r.Rpm),
										"avgLatencyMs": roundToTwoDecimalPlaces(r.AvgLatencyMs),
									})
								}
								return out
							}(),
							"slowestEndpoints": func() []map[string]any {
								out := make([]map[string]any, 0, len(slowest))
								for _, r := range slowest {
									var p95v *float64
									if r.P95LatencyMs.Valid {
										v := r.P95LatencyMs.Float64
										p95v = &v
									}
									out = append(out, map[string]any{
										"path":         r.Path,
										"avgLatencyMs": roundToTwoDecimalPlaces(r.AvgLatencyMs),
										"p95LatencyMs": p95v,
									})
								}
								return out
							}(),
						},
						"systemHealth": map[string]any{
							"apiUptimePercent": roundToTwoDecimalPlaces(apiUptimePercent),
							"averageLatencyMs": roundToTwoDecimalPlaces(avgLatencyMs),
							"errorRatePercent": roundToTwoDecimalPlaces(errorRatePercent),
						},
					}
					return resp, nil
				},
			},
			"users": makeListField(
				makeNodeListType("UserList", UserType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list  ListResult
						tx    *gorm.DB
						users []user.User
					)
					tx = parseDbClause(params, db.Model(&user.User{}), UserType)
					tx = tx.Where("id = ?", userID)
					res := tx.Debug().Scan(&users)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range users {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"applications": makeListField(
				makeNodeListType("ApplicationList", ApplicationType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list         ListResult
						tx           *gorm.DB
						applications []application.Application
					)
					tx = db.Model(&application.Application{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, ApplicationType).Order("created_at DESC")
					res := tx.Debug().Scan(&applications)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range applications {
							if u.ApiKey != "" {
								// decrypt
								rawApiKey, err := utils.Decrypt(u.ApiKey, []byte(utils.GetConfig().EncryptionKey))
								if err != nil {
									log.Error("error decrypting api key: %v", err)
								}
								u.ApiKey = string(rawApiKey)
							}
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"databases": makeListField(
				makeNodeListType("DatabaseList", DatabaseType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list      ListResult
						tx        *gorm.DB
						databases []database.Database
					)
					tx = db.Model(&database.Database{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, DatabaseType).Order("created_at DESC")
					res := tx.Debug().Scan(&databases)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range databases {
							// decrypt db passwords
							dec, _ := utils.Decrypt(u.Password, []byte(utils.GetConfig().EncryptionKey))
							u.Password = string(dec)
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"columns": makeListField(
				makeNodeListType("ColumnList", ColumnType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						columns []database.Column
					)
					tx = db.Model(&database.Column{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, ColumnType).Order("created_at DESC")
					res := tx.Debug().Scan(&columns)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range columns {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"index": makeListField(
				makeNodeListType("IndexList", IndexType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						indexes []database.Index
					)
					tx = db.Model(&database.Index{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, IndexType)
					res := tx.Debug().Scan(&indexes)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range indexes {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"endpoints": makeListField(
				makeNodeListType("EndpointList", EndpointType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}

					var (
						list      ListResult
						tx        *gorm.DB
						endpoints []endpoint.Endpoint
					)
					tx = db.Model(&endpoint.Endpoint{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, EndpointType).Order("created_at DESC")
					res := tx.Debug().Scan(&endpoints)

					if res.Error != nil {
						return nil, res.Error
					}

					if res.RowsAffected > 0 {
						appIDSet := make(map[string]struct{}, len(endpoints))
						for _, e := range endpoints {
							appIDSet[e.ApplicationID] = struct{}{}
						}
						appIDs := make([]string, 0, len(appIDSet))
						for id := range appIDSet {
							appIDs = append(appIDs, id)
						}

						type appRow struct {
							ID   string
							Slug string
						}
						var apps []appRow
						if err := db.Table("applications").
							Select("id, slug").
							Where("id IN ?", appIDs).
							Where("user_id = ?", userID).
							Scan(&apps).Error; err != nil {
							return nil, err
						}
						slugByApp := make(map[string]string, len(apps))
						for _, a := range apps {
							slugByApp[a.ID] = a.Slug
						}

						base := utils.GetConfig().ServerBaseURL
						list.Nodes = make([]interface{}, 0, len(endpoints))
						for _, u := range endpoints {
							slug := slugByApp[u.ApplicationID]
							version := "v1"
							if u.Version != "" {
								version = u.Version
							}
							u.URL = utils.FormatPublicURL(base, version, slug, u.Path)
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"schemas": makeListField(
				makeNodeListType("SchemaList", SchemaType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						schemas []database.Schema
					)
					tx = db.Model(&database.Schema{})
					tx = tx.Where("user_id = ?", userID)

					tx = parseDbClause(params, tx, SchemaType)
					res := tx.Debug().Scan(&schemas)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range schemas {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"tables": makeListField(
				makeNodeListType("TableList", TableType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list   ListResult
						tx     *gorm.DB
						tables []database.Table
					)
					tx = db.Model(&database.Table{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, TableType)
					res := tx.Debug().Scan(&tables)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range tables {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"endpointScript": makeListField(
				makeNodeListType("EndpointScriptList", EndpointScriptType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						scripts []endpoint.EndpointScript
					)
					tx = db.Model(&endpoint.EndpointScript{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, EndpointScriptType)
					res := tx.Debug().Scan(&scripts)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range scripts {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"audits": makeListField( // todo: make application_id required instead
				makeNodeListType("AuditList", AuditType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list   ListResult
						tx     *gorm.DB
						audits []audit.Audit
					)
					tx = db.Model(&audit.Audit{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, AuditType).Order("created_at DESC")
					res := tx.Debug().Scan(&audits)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range audits {
							row := db.Raw(`SELECT 
	    		  	COALESCE(first_name, '')      AS first_name
							FROM users WHERE id = ?`, userID).Row()
							if err := row.Scan(&u.Username); err != nil {
								fmt.Println("Error scanning row:", err)
								return nil, errors.New("something went wrong")
							}

							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"stream_logs": &graphql.Field{
				Type: makeNodeListType("StreamLogList", StreamLogType),
				Args: streamLogArgs,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					userID, ok := p.Context.Value("id").(string)
					if !ok || userID == "" {
						return nil, errors.New("unauthorized access")
					}

					limit := 100
					if v, ok := p.Args["limit"].(int); ok && v > 0 && v <= 1000 {
						limit = v
					}

					level, _ := p.Args["level"].(string)
					appID, _ := p.Args["application"].(string)
					q, _ := p.Args["q"].(string)

					var since *time.Time
					if s, ok := p.Args["sinceISO"].(string); ok && s != "" {
						if t, err := time.Parse(time.RFC3339, s); err == nil {
							since = &t
						}
					}

					filter := logPkg.Filter{
						UserID:      userID,
						Application: appID,
						Level:       level,
						Since:       since,
						Query:       q,
					}
					opts := logPkg.ListOptions{
						Limit: limit,
						Sort:  logPkg.SortDesc,
					}

					docs, total, err := logRepo.List(p.Context, filter, opts)
					if err != nil {
						return nil, err
					}

					var list ListResult
					list.Nodes = make([]interface{}, 0, len(docs))
					for i := range docs {
						var (
							username        string
							applicationName string
						)

						row := db.Raw(`
				    SELECT
				        COALESCE(u.first_name, '') AS username,
				        COALESCE(a.name, '') AS application_name
				    FROM users u
				    JOIN applications a ON a.user_id = u.id
				    WHERE u.id = ? AND a.id = ?
				`, docs[i].User, docs[i].Application).Row()

						if err := row.Scan(&username, &applicationName); err != nil {
							log.Error(`error scanning row for user firstname; %v`, err)
						}
						docs[i].User = username
						docs[i].Application = applicationName
						list.Nodes = append(list.Nodes, interface{}(docs[i]))
					}
					if total >= 0 {
						list.TotalCount = int(total)
					} else {
						list.TotalCount = len(docs)
					}
					return list, nil
				},
			},
		}})
}

var streamLogArgs = graphql.FieldConfigArgument{
	"limit":       &graphql.ArgumentConfig{Type: graphql.Int},
	"level":       &graphql.ArgumentConfig{Type: graphql.String},
	"application": &graphql.ArgumentConfig{Type: graphql.String},
	"sinceISO":    &graphql.ArgumentConfig{Type: graphql.String},
	"q":           &graphql.ArgumentConfig{Type: graphql.String},
}

func roundToTwoDecimalPlaces(val float64) float64 {
	return math.Round(val*100) / 100
}
