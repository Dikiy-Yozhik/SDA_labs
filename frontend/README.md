# WeatherWise frontend

Static frontend for the WeatherWise lab project. It uses plain HTML, CSS, and JavaScript; no Node.js build is required.

## Preview without the Go API

From PowerShell, run this from the repository root:

```powershell
py -m http.server 5500 --directory frontend/public
```

Open <http://localhost:5500/?demo=1>. The page displays explicitly labelled sample data in demo mode. To stop the server, press `Ctrl+C` in PowerShell.

## Expected API

The app makes requests to `GET /api/weather?city=Stockholm` and expects JSON in this shape:

```json
{
  "city": "Stockholm",
  "country": "Sweden",
  "updatedAt": "2026-10-10T14:00:00+02:00",
  "current": {
    "temperatureC": 12.4,
    "feelsLikeC": 10.8,
    "windSpeedKmh": 14.2,
    "precipitationProbability": 20,
    "weatherCode": 3
  },
  "hourly": [
    {
      "time": "2026-10-10T14:00",
      "temperatureC": 12.0,
      "precipitationProbability": 20,
      "weatherCode": 3
    }
  ],
  "recommendation": {
    "title": "A light jacket could be useful",
    "message": "Bring a light jacket; little precipitation is expected."
  }
}
```

The example numbers and dates are illustrative only. `hourly` should ideally contain the next 24 hourly forecast points. The UI displays up to eight points sampled at three-hour intervals.

The app also expects `GET /api/health` to be implemented later for manual health checks, but it does not call that endpoint automatically.

## Nginx / Compose

The Dockerfile serves `public/` with Nginx. `nginx.conf` proxies `/api/...` to the Compose service named `backend` on port `8080`. The backend container therefore needs to listen on `0.0.0.0:8080` inside the Compose network. Publish Nginx's port 80 from the root Compose file, for example `8080:80`.

## Demo mode

Add `?demo=1` to the page URL to preview the layout with sample data. Without this parameter, the frontend calls the Go API; it never silently replaces API errors with fabricated weather data.
