package models

import "time"


type DeviceType struct {
    ID            int       `json:"id"`
    Name        string      `json:"name" validate:"required"`
    IPAddress     string    `json:"ip_address" validate:"required"`
    Device        string    `json:"device" validate:"required"`
    Error         bool      `json:"error"`
    Description   string    `json:"description"`
    DownTime      time.Time `json:"down_time" validate:"required"`
    Type          string    `json:"type" validate:"required"`
    StatusMessage string    `json:"status_message"`
}
