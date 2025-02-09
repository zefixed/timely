package get

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	resp "timely/internal/lib/api/response"
	"timely/internal/lib/logger/sl"
	"timely/internal/storage"
	"timely/internal/storage/postgres"
)

type Request struct {
	UID int `json:"uid"`
}

type Response struct {
	resp.Response
	Users []postgres.User `json:"users"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.52.1 --name=UserGetter
type UserGetter interface {
	GetUser(id int) ([]postgres.User, error)
}

func New(log *slog.Logger, userGetter UserGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			log.Error("request body is empty")
			render.JSON(w, r, resp.Error("empty request"))
			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		userIDString := chi.URLParam(r, "id")
		if userIDString == "" {
			userIDString = "-1"
		}

		userID, err := strconv.Atoi(userIDString)
		if err != nil {
			log.Error("failed to convert user id to int")
			render.JSON(w, r, resp.Error("id is not a number"))
			return
		}

		users, err := userGetter.GetUser(userID)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Error("user not found")
				render.JSON(w, r, resp.Error("user not found"))
				return
			}
			log.Error("failed to get user")
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		log.Info("user received")

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Users:    users,
		})
	}
}
