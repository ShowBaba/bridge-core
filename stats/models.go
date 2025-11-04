package stats

type StatsResponse struct {
	TotalDatabases       int `json:"total_databases"`
	TotalApplications    int `json:"total_applications"`
	TotalEndpoints       int `json:"total_endpoints"`
	TotalAPIRequests     int `json:"total_api_requests"`
	EndpointDistribution []struct {
		Method     string  `json:"method"`
		Percentage float64 `json:"percentage"`
	}
}
