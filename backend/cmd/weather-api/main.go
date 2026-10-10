package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"weatherwise/backend/internal/weather"
)

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	weatherService := weather.NewService()
	mux := http.NewServeMux()

	// Проверка доступности бэкенда.
	mux.HandleFunc("GET /api/health", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "weather-api",
		})
	})

	// Получение прогноза для указанного города.
	mux.HandleFunc("GET /api/weather", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		city := strings.TrimSpace(r.URL.Query().Get("city"))

		if utf8.RuneCountInString(city) < 2 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "Укажи название города (не менее двух символов).",
			})
			return
		}

		if utf8.RuneCountInString(city) > 100 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: "Название города слишком длинное.",
			})
			return
		}

		ctx, cancel := context.WithTimeout(
			r.Context(), 11*time.Second,
		)
		defer cancel()

		result, err := weatherService.GetWeather(ctx, city)
		if err != nil {
			if errors.Is(err, weather.ErrCityNotFound) {
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error: "Не удалось найти такой город. " +
						"Проверь название и попробуй ещё раз.",
				})
				return
			}

			if isTimeout(err) ||
				errors.Is(ctx.Err(), context.DeadlineExceeded) {
				writeJSON(w, http.StatusGatewayTimeout, errorResponse{
					Error: "Сервис погоды отвечает слишком долго. " +
						"Попробуй ещё раз.",
				})
				return
			}

			log.Printf("weather request for %q failed: %v", city, err)
			writeJSON(w, http.StatusBadGateway, errorResponse{
				Error: "Не удалось получить данные от сервиса погоды.",
			})
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("WeatherWise API is listening on :8080")

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
