package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type App struct {
	Name      string
	Version   string
	Scheduler *cron.Cron
}

type Response struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

var prev = false

func NewApp() *App {
	return &App{Name: "Exchage Rate E-mail Application", Version: "1.0.0"}
}

func (app *App) Run() error {
	SetLogFormat()
	LoadEnvironmentVariable()
	log.Info("Starting Exchage Rate E-mail Application")
	GetExchangeRate()
	app.SetupJobs()
	return nil
}

func (app *App) SetupJobs() {
	app.Scheduler = cron.New(cron.WithSeconds())

	app.Scheduler.AddFunc("0 30 * * * *", func() {
		GetExchangeRate()
	})

	app.Scheduler.Start()
	select {}
}

func LoadEnvironmentVariable() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func GetExchangeRate(Test ...string) {
	response, err := http.Get("https://openexchangerates.org/api/latest.json?app_id=" + getEnv("APP_ID"))

	if err != nil {
		log.Error(err)
		return
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
		return
	}

	var responseObject Response
	err = json.Unmarshal(responseData, &responseObject)
	if err != nil {
		log.Error(err)
		return
	}

	message := fmt.Sprintf("Exchange Rate : 1 USD is %v Baht", responseObject.Rates["THB"])

	log.Info(message)

	threshold, err := strconv.ParseFloat(getEnv("THRESHOLD"), 64)

	if err != nil {
		log.Error(err)
		return
	}
	if (responseObject.Rates["THB"] <= threshold && responseObject.Rates["THB"] != 0 && !prev) || len(Test) == 1 {
		SendMail(message, getEnv("RECEIVER"))
		prev = true
	} else if responseObject.Rates["THB"] > threshold && responseObject.Rates["THB"] != 0 {
		prev = false
	}
}

func SendMail(messageString string, receiver string) {
	from := getEnv("SENDER")
	password := getEnv("PASSWORD")
	to := []string{
		receiver,
	}

	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Message.
	message := []byte("To:" + receiver + "\r\n" +
		"Subject: Exchange Rate Notification\r\n" +
		"\r\n" +
		messageString + "\r\n")

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info(fmt.Sprintf("A notification e-mail is sent to %s", receiver))
}

func SetLogFormat() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	app := NewApp()
	if err := app.Run(); err != nil {
		log.Error(err)
		log.Fatal("The application can't be start")
	}

}
