package view

import (
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var LimaLocation *time.Location

func init() {
	var err error
	LimaLocation, err = time.LoadLocation("America/Lima")
	if err != nil {
		log.Fatalf("could not load America/Lima timezone: %v", err)
	}
}

func FormatInLima(t pgtype.Timestamptz, layout string) string {
	if !t.Valid {
		return ""
	}
	return t.Time.In(LimaLocation).Format(layout)
}
