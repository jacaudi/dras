package renderer

// envelope is the JSON shape returned by the renderer.
type envelope struct {
	Image    string   `json:"image"`
	Metadata metadata `json:"metadata"`
}

type metadata struct {
	Station         string  `json:"station"`
	Product         string  `json:"product"`
	ScanTime        string  `json:"scan_time"`
	ElevationDeg    float64 `json:"elevation_deg"`
	VCP             int     `json:"vcp"`
	RendererVersion string  `json:"renderer_version"`
}

// errorBody is the JSON shape for error responses.
type errorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}
