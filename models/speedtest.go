package models

// SpeedTestRequest is the speed test portion of a check-in payload.
type SpeedTestRequest struct {
	DownloadSpeedMbps     float64     `json:"download_speed_mbps"`
	UploadSpeedMbps       float64     `json:"upload_speed_mbps"`
	LatencyMs             int         `json:"latency_ms"`
	Jitter                float64     `json:"jitter"`
	ISPName               *string     `json:"isp_name"`
	IPAddress             *string     `json:"ip_address"`
	NetworkType           string      `json:"network_type" validate:"required"`
	PacketLossPercent     *float64    `json:"packet_loss_percent"`
	TimeToFirstByteMs     int         `json:"time_to_first_byte_ms"`
	DownloadTransferredMB float64     `json:"download_transferred_mb"`
	UploadTransferredMB   float64     `json:"upload_transferred_mb"`
	Server                *ServerInfo `json:"server"`
	SpeedTestID           string      `json:"speed_test_id" validate:"required"`
	Skipped               bool        `json:"skipped"`
}

// ServerInfo holds the speed test server details.
type ServerInfo struct {
	Domain  *string `json:"domain"`
	City    *string `json:"city"`
	Country *string `json:"country"`
}
