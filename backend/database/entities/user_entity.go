package entities

import "github.com/jackc/pgx/v5/pgtype"

type User struct {
	ID           pgtype.UUID `db:"id"`
	Name         string      `db:"name"`
	Email        string      `db:"email"`
	PasswordHash string      `db:"password_hash" json:"-"`
	Timezone     string      `db:"timezone"`
	Currency     string      `db:"currency"`
	Timestamps
}
