package save

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slog"

	resp "main/cmd/internal/lib/api/response"
	"main/cmd/internal/lib/logger/sl"
	"main/cmd/internal/lib/random"
	"main/cmd/internal/storage"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
	// omitempty means that if that field is empty, it will not be included in the response, by default it would be included with an empty value, e.g., {status: xxx, error: ""} or with omitempty {status: xxx}
}

type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

const aliasLength = 6

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("Failed to decode request body into json", sl.Err(err))

			render.JSON(w, r, resp.Error("Failed to decode request body"))

			return
		}

		log.Info("Request body was decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			log.Error("Invalid request body", sl.Err(err))

			validateErr := err.(validator.ValidationErrors)

			render.JSON(w, r, resp.Error("Invalid request body"))
			render.JSON(w, r, resp.ValidationErrors(validateErr))

			return
		}

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLAlreadyExists) {
			log.Info("URL already exists", slog.String("url", req.URL))

			render.JSON(w, r, resp.Error("URL already exists"))

			return
		}
		if err != nil {
			log.Error("Failed to add URL", sl.Err(err))

			render.JSON(w, r, resp.Error("Failed to add URL"))

			return
		}

		log.Info("URL was added", slog.Int64("id", id))

		render.JSON(w, r, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
	}
}
