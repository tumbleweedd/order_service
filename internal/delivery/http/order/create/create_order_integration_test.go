package create

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mockCache "github.com/tumbleweedd/two_services_system/order_service/internal/cacheImpl/mocks"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	mockServices "github.com/tumbleweedd/two_services_system/order_service/internal/repository/mocks"
	"github.com/tumbleweedd/two_services_system/order_service/internal/services/order/create"
)

func TestCreateOrders(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	repoCreator := mockServices.NewMockOrderCreator(ctl)

	cache := mockCache.NewMockCacheI(ctl)

	createSvc := create.New(log, cache, repoCreator)

	h := NewHandler(log, createSvc)

	serverFunc := h.Create

	type mockBehavior func(
		mockRepo *mockServices.MockOrderCreator,
		toSave models.Order,
		expectedResponse uuid.UUID,
	)

	//TODO: подумать, как убрать хардкод
	userUUID := uuid.New()
	productUUIDs := []uuid.UUID{uuid.New(), uuid.New()}

	tCases := []struct {
		name           string
		toSave         models.Order
		mockBehavior   mockBehavior
		reqBody        string
		wantErr        bool
		expectedResult uuid.UUID
	}{
		{
			name: "OK",
			toSave: models.Order{
				UserUUID: userUUID,
				Products: []models.Product{
					{UUID: productUUIDs[0], Amount: 350},
					{UUID: productUUIDs[1], Amount: 1000},
				},
				Status:      1,
				PaymentType: 1,
			},
			mockBehavior: func(mockRepo *mockServices.MockOrderCreator, toSave models.Order, expResponse uuid.UUID) {
				mockRepo.EXPECT().Create(ctx, &toSave).Return(expResponse, nil)
			},
			reqBody: `
				{
					"user_uuid": "%s",
					"products":[
						{
							"uuid": "%s",
							"amount": 350
						},
						{
							"uuid": "%s",
							"amount": 1000
						}
					],
					"payment_type": "card"
				}
			`,
			wantErr:        false,
			expectedResult: uuid.New(),
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			tCase.mockBehavior(repoCreator, tCase.toSave, tCase.expectedResult)

			rec := httptest.NewRecorder()

			req := httptest.NewRequest(
				http.MethodPost,
				"/order",
				bytes.NewBuffer([]byte(fmt.Sprintf(tCase.reqBody, userUUID, productUUIDs[0], productUUIDs[1]))),
			)
			defer req.Body.Close()

			req.Header.Set("Content-Type", "Application/Json")

			serverFunc(rec, req)

			res := rec.Result()

			data, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			expected := fmt.Sprintf("{\"order_uuid\":\"%s\"}\n", tCase.expectedResult.String())

			require.Equal(t, expected, string(data))
		})
	}
}
