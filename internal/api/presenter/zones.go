package presenter

// ZoneItem is a single zone entry in list responses.
type ZoneItem struct {
	JakimCode string `json:"jakimCode"`
	Negeri    string `json:"negeri"`
	Daerah    string `json:"daerah"`
}

// ZoneByCoordinateResponse is the JSON envelope for GPS-lookup responses.
type ZoneByCoordinateResponse struct {
	Zone     string `json:"zone"`
	State    string `json:"state"`
	District string `json:"district"`
}
