package websocket

import "time"

type wsInput struct {
	Token         string `json:"token"`
	ApplicationID uint   `json:"application_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
}
