// Package models defines data structures used across the loyalty system.

package models

import (
	"fmt"

	"github.com/max-marek-projects/custodia/internal/utils"
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
var StringDataTypeToProto map[DataType]proto.DataType

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

// NewCardData creates a new CardData instance with validation.
// It validates all fields using the corresponding validators from the utils package.
// Returns the CardData pointer and a validation error, or nil if all fields are valid.
func NewCardData(number, holderName, expirationDate, cvv string) (*CardData, error) {
	if err := utils.ValidateCardNumber(number); err != nil {
		return nil, fmt.Errorf("invalid card number: %w", err)
	}
	if err := utils.ValidateCardHolder(holderName); err != nil {
		return nil, fmt.Errorf("invalid card holder name: %w", err)
	}
	if err := utils.ValidateExpirationDate(expirationDate); err != nil {
		return nil, fmt.Errorf("invalid expiration date: %w", err)
	}
	if err := utils.ValidateCVV(cvv); err != nil {
		return nil, fmt.Errorf("invalid CVV: %w", err)
	}
	return &CardData{
		Number:         number,
		HolderName:     holderName,
		ExpirationDate: expirationDate,
		CVV:            cvv,
	}, nil
}
