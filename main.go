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

// Konfigurasi koneksi ke PostgreSQL
const (
	DB_USER     = "postgres"
	DB_PASSWORD = "1234"
	DB_NAME     = "productivity_hub"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
	API_KEY     = "7e8bd1a1817b902c910f11d573c50713" // Ganti dengan API key sendiri dari OpenWeatherMap
)

var db *sql.DB // Variabel global untuk menyimpan koneksi database

// Fungsi untuk inisialisasi koneksi ke database
func initDB() {
	var err error
	// Format string koneksi PostgreSQL
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	
	// Membuka koneksi ke database
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Gagal terhubung ke database:", err)
	}

	// Mengecek apakah koneksi berhasil
	if err = db.Ping(); err != nil {
		log.Fatal("Database tidak dapat diakses:", err)
	}

	fmt.Println("Koneksi database sukses!")
}

// Struktur untuk menyimpan data cuaca dari API OpenWeatherMap
type WeatherResponse struct {
	Name string `json:"name"` // Nama kota
	Sys  struct {
		Country string `json:"country"` // Negara
	} `json:"sys"`
	Main struct {
		Temp     float64 `json:"temp"`     // Suhu dalam derajat Celsius
		Humidity int     `json:"humidity"` // Kelembaban dalam persen
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"` // Deskripsi cuaca (contoh: Hujan ringan)
	} `json:"weather"`
	Wind struct {
		Speed float64 `json:"speed"` // Kecepatan angin dalam m/s
	} `json:"wind"`
}

// Fungsi untuk mengambil data cuaca dari API OpenWeatherMap
func fetchWeatherData(city string) (*WeatherResponse, error) {
	// Membuat URL request API
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric&lang=id", city, API_KEY)

	// Melakukan HTTP request ke API
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Membaca respon dari API
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parsing JSON dari API ke struct WeatherResponse
	var weatherData WeatherResponse
	if err := json.Unmarshal(body, &weatherData); err != nil {
		return nil, err
	}

	return &weatherData, nil
}

// Fungsi untuk menyimpan data cuaca ke database
func saveWeatherToDB(weather WeatherResponse) error {
	_, err := db.Exec(
		"INSERT INTO weather_data (city, country, temperature, humidity, weather_description, wind_speed) VALUES ($1, $2, $3, $4, $5, $6)",
		weather.Name, weather.Sys.Country, weather.Main.Temp, weather.Main.Humidity, weather.Weather[0].Description, weather.Wind.Speed,
	)
	return err
}

// Endpoint untuk mengambil data cuaca yang tersimpan di database
func getWeatherData(c *gin.Context) {
	rows, err := db.Query("SELECT id, city, country, temperature, humidity, weather_description, wind_speed, recorded_at FROM weather_data")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data"})
		return
	}
	defer rows.Close()

	// Menampung hasil query dalam bentuk slice of map
	var weatherList []map[string]interface{}
	for rows.Next() {
		var id int
		var city, country, description string
		var temperature, windSpeed float64
		var humidity int
		var recordedAt string

		// Scan setiap baris hasil query
		if err := rows.Scan(&id, &city, &country, &temperature, &humidity, &description, &windSpeed, &recordedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca data"})
			return
		}

		// Menyimpan hasil query ke dalam slice
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

	// Mengembalikan data dalam format JSON
	c.JSON(http.StatusOK, weatherList)
}

// Endpoint untuk mengambil data cuaca dari API dan menyimpannya ke database
func getAndSaveWeather(c *gin.Context) {
	// Mengambil parameter 'city' dari query
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter city diperlukan"})
		return
	}

	// Mengambil data cuaca dari OpenWeatherMap API
	weatherData, err := fetchWeatherData(city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data cuaca"})
		return
	}

	// Menyimpan data cuaca ke database
	if err := saveWeatherToDB(*weatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data"})
		return
	}

	// Mengembalikan respon sukses
	c.JSON(http.StatusOK, gin.H{"message": "Data berhasil disimpan!", "data": weatherData})
}

func main() {
	// Inisialisasi database
	initDB()
	defer db.Close()

	// Membuat router dengan Gin
	r := gin.Default()

	// Mendefinisikan endpoint API
	r.GET("/weather", getWeatherData)       // Mengambil data dari database
	r.GET("/weather/fetch", getAndSaveWeather) // Mengambil dari API & menyimpan ke database

	// Menjalankan server
	fmt.Println("Server berjalan di http://localhost:8080")
	r.Run(":8080")
}
