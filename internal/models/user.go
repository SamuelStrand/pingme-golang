package models

import "time"

type User struct {
	ID         int       `db:"id" json:"id"`
	Email      string    `db:"email" json:"email"`
	Created_at time.Time `db:"cerated_at" json:"created_atx"`
}
