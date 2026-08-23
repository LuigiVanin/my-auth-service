package utils

import (
	"time"

	"gorm.io/gorm"
)

const (
	OK      string = "OK"
	OFFLINE string = "OFFLINE"
	ERROR   string = "ERROR"
)

type DatabaseInfo struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Latency  time.Duration
}

func (this DatabaseInfo) ToJSON() JSON {
	return JSON{
		"status":   this.Status,
		"database": this.Database,
		"latency": JSON{
			"text":         this.Latency.String(),
			"miliseconds":  this.Latency.Milliseconds(),
			"microseconds": this.Latency.Microseconds(),
		},
	}
}

func GetDatabaseInfo(client *gorm.DB) DatabaseInfo {
	info := DatabaseInfo{
		Status:   OFFLINE,
		Database: "unknown",
		Latency:  0,
	}

	if db, err := client.DB(); err != nil && client == nil {
		info.Status = ERROR

	} else {
		start := time.Now()
		if err = db.Ping(); err != nil {
			info.Status = ERROR
		}
		info.Latency = time.Since(start)

		if err == nil {
			info.Database = client.Config.Dialector.Name()
			info.Status = OK
		}
	}

	return info
}
