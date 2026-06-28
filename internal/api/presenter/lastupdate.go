package presenter

// LastUpdateResponse is the JSON envelope for /api/last-update.
type LastUpdateResponse struct {
	LastRun    string `json:"last_run"`
	LastStatus string `json:"last_status"`
}
