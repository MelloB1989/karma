package ai

func (f *F) EnableGrokLiveSearch(opts struct {
	ReturnCitations  bool             `json:"return_citations"`
	MaxSearchResults int              `json:"max_search_results"`
	Sources          []map[string]any `json:"sources"`
}) {
	search_params := map[string]any{
		"mode":             "auto",
		"return_citations": opts.ReturnCitations,
	}
	if len(opts.Sources) > 0 {
		search_params["sources"] = opts.Sources
	}
	if opts.MaxSearchResults == 0 {
		opts.MaxSearchResults = 5
	}
	f.optionalFields["search_parameters"] = search_params
}

/* Example:
kai.EnableGrokLiveSearch(struct {
	ReturnCitations  bool             `json:"return_citations"`
	MaxSearchResults int              `json:"max_search_results"`
	Sources          []map[string]any `json:"sources"`
}{
	ReturnCitations:  true,
	MaxSearchResults: 10,
	Sources: []map[string]any{
		{"type": "web", "country": "IN"},
		{"type": "x", "included_x_handles": []string{"lyzn_ai", "mellob1989"}},
	},
})
*/
