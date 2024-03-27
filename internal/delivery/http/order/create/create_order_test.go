package create

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidate(t *testing.T) {
	tCases := []struct {
		name  string
		input *CreateOrderRequest
	}{
		{
			name: "payment_card",
			input: &CreateOrderRequest{
				UserUUID: uuid.New().String(),
				Products: []Products{
					{
						UUID:   uuid.New().String(),
						Amount: 320,
					},
				},
				PaymentType: "card",
			},
		},
		{
			name: "payment_points",
			input: &CreateOrderRequest{
				UserUUID: uuid.New().String(),
				Products: []Products{
					{
						UUID:   uuid.New().String(),
						Amount: 320,
					},
				},
				PaymentType: "points",
				WithPoints:  200,
			},
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			err := tCase.input.validate()
			require.NoError(t, err)
		})
	}
}

func TestValidateError(t *testing.T) {
	tCases := []struct {
		name   string
		input  *CreateOrderRequest
		expErr error
	}{
		{
			name:   "bad_user_uuid",
			input:  &CreateOrderRequest{UserUUID: ""},
			expErr: errInvalidUserUUID,
		},
		{
			name: "bad_product_uuid",
			input: &CreateOrderRequest{
				UserUUID:    uuid.New().String(),
				PaymentType: "card",
				Products: []Products{
					{
						UUID: "",
					},
				},
			},
			expErr: errInvalidProductUUID,
		},
		{
			name: "bad_product_amount",
			input: &CreateOrderRequest{
				UserUUID:    uuid.New().String(),
				PaymentType: "card",
				Products: []Products{
					{UUID: uuid.New().String(), Amount: 0},
				},
			},
			expErr: errInvalidAmount,
		},
		{
			name: "no_products",
			input: &CreateOrderRequest{
				UserUUID:    uuid.New().String(),
				PaymentType: "card",
			},
			expErr: errEmptyProducts,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			err := tCase.input.validate()
			require.Error(t, err)
			require.EqualError(t, tCase.expErr, err.Error())
		})
	}
}
