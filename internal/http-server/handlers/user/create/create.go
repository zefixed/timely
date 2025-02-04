package create

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"io"
	"log/slog"
	"net/http"
	resp "timely/internal/lib/api/response"
	"timely/internal/lib/logger/sl"
)

type Request struct {
	Name        string `json:"name"  validate:"required"`
	Description string `json:"description,omitempty"`
}

type Response struct {
	resp.Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.52.1 --name=UserCreator
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

		if err := validator.New().Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			errors.As(err, &validateErr)
			log.Error("invalid request", sl.Err(err))
			render.JSON(w, r, resp.ValidationError(validateErr))
			return
		}

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
