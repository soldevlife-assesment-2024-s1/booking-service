package entity

import (
	"database/sql"
	"time"
)

type Booking struct {
	ID                string       `db:"id"` // UUID
	UserID            int64        `db:"user_id"`
	TicketDetailID    int64        `db:"ticket_detail_id"`
	TotalTickets      int          `db:"total_tickets"`
	FullName          string       `db:"full_name"`
	PersonalID        string       `db:"personal_id"`
	BookingDate       time.Time    `db:"booking_date"`
	BookingExpiration time.Time    `db:"booking_expiration"`
	CreatedAt         time.Time    `db:"created_at"`
	UpdatedAt         sql.NullTime `db:"updated_at"`
	DeletedAt         sql.NullTime `db:"deleted_at"`
}

type Payment struct {
	ID                int64        `db:"id"`
	BookingID         string       `db:"booking_id"`
	Amount            float64      `db:"amount"`
	Currency          string       `db:"currency"`
	Status            string       `db:"status"`
	PaymentMethod     string       `db:"payment_method"`
	PaymentDate       time.Time    `db:"payment_date"`
	PaymentExpiration time.Time    `db:"payment_expiration"`
	CreatedAt         time.Time    `db:"created_at"`
	UpdatedAt         sql.NullTime `db:"updated_at"`
	DeletedAt         sql.NullTime `db:"deleted_at"`
}
