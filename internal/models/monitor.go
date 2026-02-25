package models

import "time"

type Monitor struct {
	ID         int       `db:"id" json:"id"`
	User_ID    int       `db:"user_id" json:"user_id"`
	URL        string    `db:"url" json:"url"`
	Created_at time.Time `db:"created_at" json:"created_atx"`
}
