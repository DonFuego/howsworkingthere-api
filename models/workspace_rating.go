package models

// WorkspaceRatingsRequest is the workspace ratings portion of a check-in payload.
type WorkspaceRatingsRequest struct {
	OutletsAtBar   bool   `json:"outlets_at_bar"`
	OutletsAtTable bool   `json:"outlets_at_table"`
	Crowdedness    int    `json:"crowdedness" validate:"required"`
	EaseOfWork     int    `json:"ease_of_work" validate:"required"`
	BestWorkType   string `json:"best_work_type" validate:"required"`
}
