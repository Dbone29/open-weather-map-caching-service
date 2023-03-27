package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type WeatherData struct {
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

type WeatherCache struct {
	sync.Mutex
	data       map[string]*WeatherData
	expiration time.Duration
}

func fetchWeatherData(apiKey, lat, lon string) (*WeatherData, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%s&lon=%s&appid=%s", lat, lon, apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var weatherData WeatherData
	err = json.Unmarshal(body, &weatherData)
	if err != nil {
		return nil, err
	}
	return &weatherData, nil
}

func (wc *WeatherCache) AddToCache(key string, weatherData *WeatherData) {
	wc.Lock()
	defer wc.Unlock()
	wc.data[key] = weatherData
	go func() {
		time.Sleep(wc.expiration)
		wc.Lock()
		defer wc.Unlock()
		delete(wc.data, key)
	}()
}

func (wc *WeatherCache) GetFromCache(apiKey, lat, lon string) (*WeatherData, error) {
	wc.Lock()
	defer wc.Unlock()
	key := fmt.Sprintf("%s,%s", lat, lon)
	if data, ok := wc.data[key]; ok {
		return data, nil
	}
	weatherData, err := fetchWeatherData(apiKey, lat, lon)
	if err != nil {
		return nil, err
	}
	wc.AddToCache(key, weatherData)
	return weatherData, nil
}

func initConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetDefault("cache.expiration", "10m")
	viper.SetDefault("server.port", "8080")

	viper.SetEnvPrefix("MYAPP")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error in config file: %s", err))
	}
}

func main() {
	initConfig()

	apiKey := viper.GetString("openweathermap.api_key")
	cacheExpiration, _ := time.ParseDuration(viper.GetString("cache.expiration"))
	cache := WeatherCache{data: make(map[string]*WeatherData), expiration: cacheExpiration}

	router := gin.Default()

	router.GET("/weather", func(c *gin.Context) {
		lat := c.Query("lat")
		lon := c.Query("lon")
		if lat == "" || lon == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and longitude query parameters 'lat' and 'lon' are required"})
			return
		}
		weatherData, err := cache.GetFromCache(apiKey, lat, lon)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, weatherData)
	})

	serverPort := viper.GetString("server.port")
	router.Run(":" + serverPort)
}
