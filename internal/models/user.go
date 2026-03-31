package models

import "time"

type User struct {
	ID         string    `db:"id" json:"id"`
	UserTG     string    `db:"user_tg" json:"user_tg"`
	Email      string    `db:"email" json:"email"`
	Password   string    `db:"password" json:"-"`
	Created_at time.Time `db:"cerated_at" json:"created_at"`
}
