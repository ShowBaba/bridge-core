package graphql

type ListResult struct {
	Nodes      []interface{} `json:"nodes"`
	TotalCount int           `json:"totalCount"`
}

type Payload struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}
