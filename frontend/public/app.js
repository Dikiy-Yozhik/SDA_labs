const DEMO_MODE = new URLSearchParams(window.location.search).get("demo") === "1";
const form = document.querySelector("#cityForm");
const cityInput = document.querySelector("#cityInput");
const searchButton = document.querySelector("#searchButton");
const notice = document.querySelector("#notice");
const demoBanner = document.querySelector("#demoBanner");
const connectionStatus = document.querySelector("#connectionStatus");
const connectionDot = document.querySelector("#connectionDot");

let activeController = null;

const DEMO_WEATHER = {
  city: "Stockholm",
  country: "Sweden",
  updatedAt: new Date().toISOString(),
  current: {
    temperatureC: 12,
    feelsLikeC: 10,
    windSpeedKmh: 14,
    precipitationProbability: 20,
    weatherCode: 2,
  },
  hourly: Array.from({ length: 24 }, (_, index) => {
    const date = new Date();
    date.setHours(date.getHours() + index, 0, 0, 0);
    const hours = [12, 12, 11, 10, 9, 8, 7, 7];
    const codes = [2, 2, 3, 3, 61, 61, 3, 2];
    return {
      time: `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}T${String(date.getHours()).padStart(2, "0")}:00`,
      temperatureC: hours[index % hours.length],
      precipitationProbability: index >= 4 && index <= 6 ? 45 : 15,
      weatherCode: codes[index % codes.length],
    };
  }),
  recommendation: {
    title: "Лёгкая куртка пригодится",
    message: "Возьми лёгкую куртку. Сейчас довольно свежо, а ближе к вечеру возможен небольшой дождь.",
  },
};

const WEATHER_CODES = {
  0: { label: "Ясно", icon: "☀️", nightIcon: "🌙" },
  1: { label: "Преимущественно ясно", icon: "🌤️", nightIcon: "🌙" },
  2: { label: "Переменная облачность", icon: "⛅", nightIcon: "☁️" },
  3: { label: "Пасмурно", icon: "☁️", nightIcon: "☁️" },
  45: { label: "Туман", icon: "🌫️", nightIcon: "🌫️" },
  48: { label: "Изморозь и туман", icon: "🌫️", nightIcon: "🌫️" },
  51: { label: "Лёгкая морось", icon: "🌦️", nightIcon: "🌧️" },
  53: { label: "Морось", icon: "🌦️", nightIcon: "🌧️" },
  55: { label: "Сильная морось", icon: "🌧️", nightIcon: "🌧️" },
  56: { label: "Ледяная морось", icon: "🌧️", nightIcon: "🌧️" },
  57: { label: "Сильная ледяная морось", icon: "🌧️", nightIcon: "🌧️" },
  61: { label: "Небольшой дождь", icon: "🌦️", nightIcon: "🌧️" },
  63: { label: "Дождь", icon: "🌧️", nightIcon: "🌧️" },
  65: { label: "Сильный дождь", icon: "🌧️", nightIcon: "🌧️" },
  66: { label: "Ледяной дождь", icon: "🌧️", nightIcon: "🌧️" },
  67: { label: "Сильный ледяной дождь", icon: "🌧️", nightIcon: "🌧️" },
  71: { label: "Небольшой снег", icon: "🌨️", nightIcon: "🌨️" },
  73: { label: "Снег", icon: "❄️", nightIcon: "❄️" },
  75: { label: "Сильный снег", icon: "❄️", nightIcon: "❄️" },
  77: { label: "Снежные зёрна", icon: "🌨️", nightIcon: "🌨️" },
  80: { label: "Небольшой ливень", icon: "🌦️", nightIcon: "🌧️" },
  81: { label: "Ливень", icon: "🌧️", nightIcon: "🌧️" },
  82: { label: "Сильный ливень", icon: "🌧️", nightIcon: "🌧️" },
  85: { label: "Снегопад", icon: "🌨️", nightIcon: "🌨️" },
  86: { label: "Сильный снегопад", icon: "❄️", nightIcon: "❄️" },
  95: { label: "Гроза", icon: "⛈️", nightIcon: "⛈️" },
  96: { label: "Гроза с градом", icon: "⛈️", nightIcon: "⛈️" },
  99: { label: "Сильная гроза с градом", icon: "⛈️", nightIcon: "⛈️" },
};

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (character) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  })[character]);
}

function hasNumber(value) {
  return value !== null && value !== undefined && value !== "" && Number.isFinite(Number(value));
}

function rounded(value, suffix = "") {
  return hasNumber(value) ? `${Math.round(Number(value))}${suffix}` : "—";
}

function weatherInfo(code) {
  return WEATHER_CODES[Number(code)] || { label: "Условия неизвестны", icon: "🌡️", nightIcon: "🌡️" };
}

function formatHour(value) {
  if (typeof value !== "string") return "—";
  const match = value.match(/T(\d{2}:\d{2})/);
  return match ? match[1] : "—";
}

function setConnectionState(state, label) {
  connectionStatus.textContent = label;
  const headerMeta = connectionStatus.parentElement;
  headerMeta.classList.remove("is-online", "is-demo", "is-offline");
  if (state) headerMeta.classList.add(`is-${state}`);
}

function showNotice(message) {
  notice.textContent = message;
  notice.hidden = !message;
}

function renderWeather(data) {
  if (!data || !data.current || !Array.isArray(data.hourly) || !data.recommendation) {
    throw new Error("Ответ API имеет неожиданный формат. Сверьте его с frontend/README.md.");
  }

  const current = data.current;
  const info = weatherInfo(current.weatherCode);
  document.querySelector("#cityName").textContent = data.city || "Неизвестный город";
  document.querySelector("#countryName").textContent = data.country || "";
  document.querySelector("#currentTemp").innerHTML = `${escapeHtml(rounded(current.temperatureC))}<span>°</span>`;
  document.querySelector("#currentCondition").textContent = info.label;
  document.querySelector("#currentIcon").textContent = info.icon;
  document.querySelector("#feelsLike").textContent = `${rounded(current.feelsLikeC, "°")}`;
  document.querySelector("#windSpeed").textContent = `${rounded(current.windSpeedKmh)} км/ч`;
  document.querySelector("#rainChance").textContent = `${rounded(current.precipitationProbability, "%")}`;
  document.querySelector("#recommendationHeading").textContent = data.recommendation.title || "Совет на день";
  document.querySelector("#recommendationText").textContent = data.recommendation.message || "Рекомендаций пока нет.";

  const updatedLabel = data.updatedAt
    ? `Обновлено: ${new Date(data.updatedAt).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })}`
    : "Актуальные погодные данные";
  document.querySelector("#updatedAt").textContent = updatedLabel;

  const visibleHours = data.hourly.filter((_, index) => index % 3 === 0).slice(0, 8);
  const hourlyContainer = document.querySelector("#hourlyForecast");
  if (!visibleHours.length) {
    hourlyContainer.innerHTML = '<div class="forecast-placeholder">Почасовой прогноз пока недоступен.</div>';
    return;
  }

  hourlyContainer.innerHTML = visibleHours.map((hour) => {
    const hourInfo = weatherInfo(hour.weatherCode);
    const rain = hasNumber(hour.precipitationProbability)
      ? `<span class="forecast-rain">☂ ${escapeHtml(rounded(hour.precipitationProbability, "%"))}</span>`
      : '<span class="forecast-rain">&nbsp;</span>';
    return `<article class="forecast-item">
      <span class="forecast-time">${escapeHtml(formatHour(hour.time))}</span>
      <span class="forecast-icon" aria-hidden="true">${hourInfo.icon}</span>
      <strong class="forecast-temp">${escapeHtml(rounded(hour.temperatureC, "°"))}</strong>
      ${rain}
    </article>`;
  }).join("");
}

async function loadWeather(city) {
  if (DEMO_MODE) {
    showNotice("");
    demoBanner.hidden = false;
    setConnectionState("demo", "Демо-режим");
    cityInput.value = DEMO_WEATHER.city;
    renderWeather(DEMO_WEATHER);
    return;
  }

  if (activeController) activeController.abort();
  activeController = new AbortController();
  const controller = activeController;
  const timeoutId = window.setTimeout(() => controller.abort(), 12000);

  searchButton.disabled = true;
  searchButton.innerHTML = 'Загрузка…';
  setConnectionState("", "Получаем прогноз");
  showNotice("");
  demoBanner.hidden = true;

  try {
    const response = await fetch(`/api/weather?city=${encodeURIComponent(city)}`, {
      headers: { Accept: "application/json" },
      signal: controller.signal,
    });
    if (!response.ok) {
      let detail = "";
      try {
        const body = await response.json();
        detail = body.error || body.message || "";
      } catch { /* The server may return a non-JSON error page. */ }
      throw new Error(detail || `Сервис погоды ответил с кодом ${response.status}.`);
    }
    const data = await response.json();
    renderWeather(data);
    setConnectionState("online", "API подключён");
  } catch (error) {
    if (error.name === "AbortError") {
      showNotice("Запрос прерван или занял слишком много времени. Проверь, запущен ли Go-бэкенд.");
    } else {
      showNotice(`${error.message || "Не удалось получить прогноз."} Убедись, что Go API доступен через /api/weather.`);
    }
    setConnectionState("offline", "Нет связи с API");
  } finally {
    window.clearTimeout(timeoutId);
    if (activeController === controller) activeController = null;
    searchButton.disabled = false;
    searchButton.innerHTML = 'Показать <span aria-hidden="true">↗</span>';
  }
}

form.addEventListener("submit", (event) => {
  event.preventDefault();
  const city = cityInput.value.trim();
  if (!city) {
    showNotice("Введи название города.");
    cityInput.focus();
    return;
  }
  if (DEMO_MODE) {
    showNotice("В демо-режиме показан один пример — Стокгольм. Поиск других городов станет доступен после подключения Go API.");
    return;
  }
  loadWeather(city);
});

if (DEMO_MODE) {
  demoBanner.hidden = false;
  loadWeather("Stockholm");
} else {
  loadWeather(cityInput.value.trim() || "Stockholm");
}
