// Package models defines data structures used across the loyalty system.

package models

import "time"

// ========== data types ==========

type DataType string

const (
	DataTypeCredential DataType = "credential"
	DataTypeText       DataType = "text"
	DataTypeBinary     DataType = "binary"
	DataTypeCard       DataType = "card"
)

// LoginPassword is the struct containing login and password.
type LoginPassword struct {
	Login    string
	Password string
}

// SecretText is the struct containing secret text
type SecretText struct {
	Text string
}

// SecretBinary is the struct containing secret binary data
type SecretBinary struct {
	Binary string
}

// CardData is the struct containing secret card data.
type CardData struct {
	Number         string
	HolderName     string
	ExpirationDate time.Time
	CVV            string
}

// ========== database ==========

// DatabaseRow represents all data from single row of database table
type DatabaseRow struct {
	ID        string
	UserID    string
	Type      DataType
	Name      string
	Data      []byte
	Metadata  string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Deleted   bool
}
