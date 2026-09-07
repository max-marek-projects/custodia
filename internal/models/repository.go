// Package models defines data structures used across the loyalty system.

package models

import (
	"time"

	"github.com/max-marek-projects/custodia/pkg/proto"
)

// ========== data types ==========

type DataType string

const (
	DataTypeCredentials DataType = "credentials"
	DataTypeText        DataType = "text"
	DataTypeBinary      DataType = "binary"
	DataTypeCard        DataType = "card"
)

var ProtoDataTypeToString = map[proto.DataType]DataType{
	proto.DataType_DATA_TYPE_CREDENTIALS: DataTypeCredentials,
	proto.DataType_DATA_TYPE_TEXT:        DataTypeText,
	proto.DataType_DATA_TYPE_BINARY:      DataTypeBinary,
	proto.DataType_DATA_TYPE_CARD:        DataTypeCard,
}

func init() {
	StringDataTypeToProto := make(map[DataType]proto.DataType, len(ProtoDataTypeToString))
	for k, v := range ProtoDataTypeToString {
		StringDataTypeToProto[v] = k
	}
}

// Credentials is the struct containing login and password.
type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// CardData is the struct containing secret card data.
type CardData struct {
	Number         string `json:"number"`
	HolderName     string `json:"holder_name"`
	ExpirationDate string `json:"expiration_date"`
	CVV            string `json:"cvv"`
}

// ========== database ==========

// SecretData represents all data from single row of database table
type SecretData struct {
	ID        string
	UserID    int64
	Type      DataType
	Name      string
	Data      []byte
	Metadata  map[string]string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
