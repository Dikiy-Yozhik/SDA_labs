package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrCityNotFound = errors.New("city not found")

// Service получает прогноз от Open-Meteo.
type Service struct {
	httpClient *http.Client
}

func NewService() *Service {
	return &Service{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Response — формат ответа нашего API для фронтенда.
type Response struct {
	City           string          `json:"city"`
	Country        string          `json:"country"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Current        CurrentWeather  `json:"current"`
	Hourly         []HourlyWeather `json:"hourly"`
	Recommendation Recommendation  `json:"recommendation"`
}

type CurrentWeather struct {
	TemperatureC             float64  `json:"temperatureC"`
	FeelsLikeC               float64  `json:"feelsLikeC"`
	WindSpeedKmh             float64  `json:"windSpeedKmh"`
	PrecipitationProbability *float64 `json:"precipitationProbability"`
	WeatherCode              int      `json:"weatherCode"`
}

type HourlyWeather struct {
	Time                     string   `json:"time"`
	TemperatureC             float64  `json:"temperatureC"`
	PrecipitationProbability *float64 `json:"precipitationProbability"`
	WeatherCode              int      `json:"weatherCode"`
}

type Recommendation struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// Структуры для ответа Geocoding API.

type geocodingResponse struct {
	Results []location `json:"results"`
}

type location struct {
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

// Структура для нужной нам части ответа Forecast API.

type forecastResponse struct {
	Current struct {
		Time                string  `json:"time"`
		Temperature2m       float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WindSpeed10m        float64 `json:"wind_speed_10m"`
		WeatherCode         int     `json:"weather_code"`
	} `json:"current"`

	Hourly struct {
		Time                     []string   `json:"time"`
		Temperature2m            []float64  `json:"temperature_2m"`
		PrecipitationProbability []*float64 `json:"precipitation_probability"`
		WeatherCode              []int      `json:"weather_code"`
	} `json:"hourly"`
}

// GetWeather находит город, получает прогноз и строит рекомендацию.
func (s *Service) GetWeather(
	ctx context.Context,
	city string,
) (Response, error) {
	city = strings.TrimSpace(city)
	if city == "" {
		return Response{}, errors.New("city must not be empty")
	}

	loc, err := s.findLocation(ctx, city)
	if err != nil {
		return Response{}, err
	}

	forecast, err := s.getForecast(ctx, loc)
	if err != nil {
		return Response{}, err
	}

	start, err := findCurrentHour(
		forecast.Current.Time,
		forecast.Hourly.Time,
	)
	if err != nil {
		return Response{}, err
	}

	// Проверяем, что API действительно вернул
	// минимум 24 часа согласованных данных.
	if len(forecast.Hourly.Time)-start < 24 ||
		start >= len(forecast.Hourly.Temperature2m) ||
		start >= len(forecast.Hourly.PrecipitationProbability) ||
		start >= len(forecast.Hourly.WeatherCode) {
		return Response{}, errors.New(
			"Open-Meteo returned incomplete hourly data",
		)
	}

	result := Response{
		City:      loc.Name,
		Country:   loc.Country,
		UpdatedAt: time.Now().UTC(),
		Current: CurrentWeather{
			TemperatureC:             forecast.Current.Temperature2m,
			FeelsLikeC:               forecast.Current.ApparentTemperature,
			WindSpeedKmh:             forecast.Current.WindSpeed10m,
			PrecipitationProbability: forecast.Hourly.PrecipitationProbability[start],
			WeatherCode:              forecast.Current.WeatherCode,
		},
		Hourly: make([]HourlyWeather, 0, 24),
	}

	for i := start; i < start+24; i++ {
		if i >= len(forecast.Hourly.Temperature2m) ||
			i >= len(forecast.Hourly.PrecipitationProbability) ||
			i >= len(forecast.Hourly.WeatherCode) {
			return Response{}, errors.New(
				"Open-Meteo returned inconsistent hourly data",
			)
		}

		result.Hourly = append(result.Hourly, HourlyWeather{
			Time:                     forecast.Hourly.Time[i],
			TemperatureC:             forecast.Hourly.Temperature2m[i],
			PrecipitationProbability: forecast.Hourly.PrecipitationProbability[i],
			WeatherCode:              forecast.Hourly.WeatherCode[i],
		})
	}

	result.Recommendation = makeRecommendation(result.Current)

	return result, nil
}

// Находим координаты города по его названию.
func (s *Service) findLocation(
	ctx context.Context,
	city string,
) (location, error) {
	query := url.Values{}
	query.Set("name", city)
	query.Set("count", "10")
	query.Set("language", "ru")
	query.Set("format", "json")

	endpoint := "https://geocoding-api.open-meteo.com/v1/search?" +
		query.Encode()

	var result geocodingResponse
	if err := s.getJSON(ctx, endpoint, &result); err != nil {
		return location{}, fmt.Errorf(
			"geocoding request failed: %w", err,
		)
	}

	if len(result.Results) == 0 {
		return location{}, ErrCityNotFound
	}

	// Берём наиболее подходящее совпадение, выбранное сервисом.
	return result.Results[0], nil
}

// Получаем текущую погоду и почасовой прогноз.
func (s *Service) getForecast(
	ctx context.Context,
	loc location,
) (forecastResponse, error) {
	query := url.Values{}
	query.Set("latitude", fmt.Sprintf("%f", loc.Latitude))
	query.Set("longitude", fmt.Sprintf("%f", loc.Longitude))
	query.Set(
		"current",
		"temperature_2m,apparent_temperature,wind_speed_10m,weather_code",
	)
	query.Set(
		"hourly",
		"temperature_2m,precipitation_probability,weather_code",
	)
	query.Set("forecast_days", "2")

	if loc.Timezone != "" {
		query.Set("timezone", loc.Timezone)
	} else {
		query.Set("timezone", "auto")
	}

	endpoint := "https://api.open-meteo.com/v1/forecast?" +
		query.Encode()

	var result forecastResponse
	if err := s.getJSON(ctx, endpoint, &result); err != nil {
		return forecastResponse{}, fmt.Errorf(
			"forecast request failed: %w", err,
		)
	}

	return result, nil
}

// Выполняет HTTP-запрос и декодирует JSON.
func (s *Service) getJSON(
	ctx context.Context,
	endpoint string,
	target any,
) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, endpoint, nil,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	start := time.Now()
	log.Printf("Requesting Open-Meteo: %s", endpoint)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf(
			"Open-Meteo request failed after %s: %v",
			time.Since(start), err,
		)
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf(
		"Open-Meteo returned %s after %s",
		resp.Status, time.Since(start),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf(
			"Open-Meteo returned %s: %s",
			resp.Status, message,
		)
	}

	if err := json.NewDecoder(
		io.LimitReader(resp.Body, 2<<20),
	).Decode(target); err != nil {
		log.Printf("Failed to decode Open-Meteo response: %v", err)
		return fmt.Errorf("decode response: %w", err)
	}

	log.Printf("Open-Meteo response decoded successfully")
	return nil
}

// Текущий час должен совпадать с одним из часов прогноза.
func findCurrentHour(
	currentTime string,
	hourlyTimes []string,
) (int, error) {
	const layout = "2006-01-02T15:04"

	current, err := time.Parse(layout, currentTime)
	if err != nil {
		return 0, fmt.Errorf(
			"parse current time %q: %w",
			currentTime, err,
		)
	}

	// Почасовой прогноз начинается с начала часа.
	current = current.Truncate(time.Hour)

	for i, hour := range hourlyTimes {
		forecastHour, err := time.Parse(layout, hour)
		if err != nil {
			continue
		}

		if forecastHour.Equal(current) {
			return i, nil
		}
	}

	return 0, fmt.Errorf(
		"hour %q was not present in hourly forecast",
		current.Format(layout),
	)
}

// Простая бизнес-логика рекомендаций.
func makeRecommendation(c CurrentWeather) Recommendation {
	if c.PrecipitationProbability != nil &&
		*c.PrecipitationProbability >= 50 {
		return Recommendation{
			Title: "Не забудь зонт",
			Message: "Вероятность осадков высокая. " +
				"Возьми зонт или непромокаемую куртку.",
		}
	}

	if isPrecipitationCode(c.WeatherCode) {
		return Recommendation{
			Title: "Возможны осадки",
			Message: "Возможны дождь или снег. " +
				"На всякий случай захвати зонт или капюшон.",
		}
	}

	if c.WindSpeedKmh >= 35 {
		return Recommendation{
			Title: "На улице ветрено",
			Message: "Сильный ветер может быть некомфортным. " +
				"Выбирай одежду, которая защищает от ветра.",
		}
	}

	if c.FeelsLikeC <= 0 {
		return Recommendation{
			Title: "Одевайся теплее",
			Message: "По ощущениям около нуля или холоднее. " +
				"Тёплая куртка и закрытая обувь будут кстати.",
		}
	}

	if c.FeelsLikeC < 10 {
		return Recommendation{
			Title: "Лёгкая куртка пригодится",
			Message: "На улице прохладно. Возьми лёгкую куртку " +
				"или дополнительный слой одежды.",
		}
	}

	if c.TemperatureC >= 27 {
		return Recommendation{
			Title: "Не забудь воду",
			Message: "Довольно тепло. Пей воду и в солнечные " +
				"часы старайся находиться в тени.",
		}
	}

	return Recommendation{
		Title: "Хорошего дня!",
		Message: "Серьёзных погодных препятствий по текущим " +
			"данным не видно. Проверь прогноз перед выходом.",
	}
}

func isPrecipitationCode(code int) bool {
	return (code >= 51 && code <= 67) ||
		(code >= 71 && code <= 77) ||
		(code >= 80 && code <= 82) ||
		(code >= 85 && code <= 86) ||
		(code >= 95 && code <= 99)
}
