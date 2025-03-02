package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Konfigurasi PostgreSQL
const (
	DB_USER     = "postgres"
	DB_PASSWORD = "1234"
	DB_NAME     = "productivity_hub"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
	API_KEY     = "7e8bd1a1817b902c910f11d573c50713" // Ganti dengan API key sendiri
)

var db *sql.DB

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Gagal terhubung ke database:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Database tidak dapat diakses:", err)
	}
	fmt.Println("✅ Koneksi database sukses!")
}

type WeatherResponse struct {
	Name string `json:"name"`
	Sys  struct {
		Country string `json:"country"`
	} `json:"sys"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`
}

func fetchWeatherData(city string) (*WeatherResponse, error) {
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=id", city, API_KEY)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var weatherData WeatherResponse
	if err := json.Unmarshal(body, &weatherData); err != nil {
		return nil, err
	}
	return &weatherData, nil
}

func saveWeatherToDB(weather WeatherResponse) error {
	_, err := db.Exec(
		"INSERT INTO weather_data (city, country, temperature, humidity, weather_description, wind_speed) VALUES ($1, $2, $3, $4, $5, $6)",
		weather.Name, weather.Sys.Country, weather.Main.Temp, weather.Main.Humidity, weather.Weather[0].Description, weather.Wind.Speed,
	)
	return err
}

func getWeatherData(c *gin.Context) {
	rows, err := db.Query("SELECT id, city, country, temperature, humidity, weather_description, wind_speed, recorded_at FROM weather_data")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data"})
		return
	}
	defer rows.Close()

	var weatherList []map[string]interface{}
	for rows.Next() {
		var id int
		var city, country, description string
		var temperature, windSpeed float64
		var humidity int
		var recordedAt string
		if err := rows.Scan(&id, &city, &country, &temperature, &humidity, &description, &windSpeed, &recordedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca data"})
			return
		}
		weatherList = append(weatherList, map[string]interface{}{
			"id":                  id,
			"city":                city,
			"country":             country,
			"temperature":         temperature,
			"humidity":            humidity,
			"weather_description": description,
			"wind_speed":          windSpeed,
			"recorded_at":         recordedAt,
		})
	}
	c.JSON(http.StatusOK, weatherList)
}

func getAndSaveWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter city diperlukan"})
		return
	}
	weatherData, err := fetchWeatherData(city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data cuaca"})
		return
	}
	if err := saveWeatherToDB(*weatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data berhasil disimpan!", "data": weatherData})
}

func main() {
	initDB()
	defer db.Close()
	r := gin.Default()
	r.GET("/weather", getWeatherData)
	r.GET("/weather/fetch", getAndSaveWeather)

	fmt.Println("🚀 Server berjalan di http://localhost:8080")
	r.Run(":8080")
}
