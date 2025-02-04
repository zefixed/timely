package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"timely/internal/http-server/handlers/user/create/mocks"
	"timely/internal/lib/logger/handlers/slogdiscard"
)

func TestCreateHandler(t *testing.T) {
	cases := []struct {
		name            string
		userName        string
		userDescription string
		respError       string
		mockError       error
	}{
		{
			name:            "Success",
			userName:        "user name",
			userDescription: "user description",
		},
		{
			name:            "Empty user description",
			userName:        "user name",
			userDescription: "",
		},
		{
			name:            "Empty user name",
			userName:        "",
			userDescription: "user description",
			respError:       "field Name is a required field",
		},
		{
			name:            "SaveURL Error",
			userName:        "user name",
			userDescription: "user description",
			respError:       "failed to create user",
			mockError:       errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userCreatorMock := mocks.NewUserCreator(t)

			if tc.respError == "" || tc.mockError != nil {
				userCreatorMock.On("CreateUser", tc.userName, tc.userDescription).
					Return(tc.mockError).
					Once()
			}

			handler := New(slogdiscard.NewDiscardLogger(), userCreatorMock)

			input := fmt.Sprintf(`{"name": "%s", "description": "%s"}`, tc.userName, tc.userDescription)

			req, err := http.NewRequest(http.MethodPost, "/user/create", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, rr.Code, http.StatusOK)

			body := rr.Body.String()

			var resp Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

			// TODO: add more checks
		})
	}
}
