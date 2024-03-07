package order_service_http

import (
	"bytes"
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tumbleweedd/two_services_system/order_service/internal/domain/models"
	mock_services "github.com/tumbleweedd/two_services_system/order_service/internal/repository/mocks"
	"github.com/tumbleweedd/two_services_system/order_service/internal/services"
	"github.com/tumbleweedd/two_services_system/order_service/internal/services/mocks"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCreateOrders(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	repoCreator := mock_services.NewMockOrderCreator(ctl)
	repoGetter := mock_services.NewMockOrderGetter(ctl)
	repoCancaler := mock_services.NewMockOrderCancaler(ctl)

	cache := mock_cache_imp.NewMockCacheI(ctl)

	service := services.NewService(log, repoCreator, repoGetter, repoCancaler, cache)

	h := NewHandler(log, service)
	h.InitRoutes()

	serverFunc := h.createOrder

	type mockBehavior func(
		mockRepo *mock_services.MockOrderCreator,
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
		reqBody        []byte
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
			mockBehavior: func(mockRepo *mock_services.MockOrderCreator, toSave models.Order, expResponse uuid.UUID) {
				mockRepo.EXPECT().Create(ctx, &toSave).Return(expResponse, nil)
			},
			reqBody: []byte(`
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
			`),
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
				bytes.NewBuffer(tCase.reqBody),
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
