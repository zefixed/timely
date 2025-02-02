package create

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	resp "timely/internal/lib/api/response"
	"timely/internal/lib/logger/sl"
)

type Request struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Response struct {
	resp.Response
}

type UserCreator interface {
	CreateUser(name, desc string) error
}

func New(log *slog.Logger, userCreator UserCreator) http.HandlerFunc {
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

		err = userCreator.CreateUser(req.Name, req.Description)
		if err != nil {
			log.Error("failed to create user", sl.Err(err))
			render.JSON(w, r, resp.Error("failed to create user"))
			return
		}

		log.Info("user created")

		render.JSON(w, r, Response{
			Response: resp.OK(),
		})
	}
}
