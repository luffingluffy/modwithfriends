package main

import (
	"log"
	"modwithfriends/bot"
	"modwithfriends/http"
	"modwithfriends/postgres"
	"modwithfriends/smtp"
	"modwithfriends/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const (
	envPort             = "PORT"
	envDeploymentType   = "DEPLOYMENT_TYPE"
	envTelegramBotToken = "TELEGRAM_BOT_TOKEN"
	envDatabaseURL      = "DATABASE_URL"
	envPwd              = "PWD_LMAO"
	envFwensClientURL   = "FWENS_CLIENT_URL"
	envEmail            = "ENV_EMAIL"
	envEmailPassword    = "ENV_EMAIL_PASSWORD"
	envSMTPHost         = "ENV_SMTP_HOST"
	envSMTPPort         = "ENV_SMTP_PORT"
)

func main() {
	deploymentType, exist := os.LookupEnv(envDeploymentType)
	if !exist || deploymentType == "development" {
		err := utils.LoadEnvironmentVariables()
		if err != nil {
			log.Fatal(err)
		}
	}

	config, err := utils.GetConfig(
		envPort,
		envTelegramBotToken,
		envDatabaseURL,
		envPwd,
		envFwensClientURL,
		envEmail,
		envEmailPassword,
		envSMTPHost,
		envSMTPPort,
	)
	if err != nil {
		log.Fatal(err)
	}

	db, err := postgres.Open(config[envDatabaseURL])
	if err != nil {
		panic(err)
	}
	defer db.Close()

	us := &postgres.UserService{DB: db}
	ms := &postgres.ModuleService{DB: db}
	gs := &postgres.GroupService{DB: db}

	es := smtp.NewEmailClient(
		config[envEmail],
		config[envEmailPassword],
		config[envSMTPHost],
		utils.ToIntOrPanic(config[envSMTPPort]),
	)

	bot, err := bot.NewBot(config[envTelegramBotToken], bot.NewRoutes(us, ms, gs, es, config[envEmail]))
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{config[envFwensClientURL]}
	router.Use(cors.New(corsConfig))

	server := http.Server{
		Port:         utils.ToIntOrPanic(config[envPort]),
		Router:       router,
		Bot:          bot,
		UserService:  us,
		GroupService: gs,
		Pwd:          config[envPwd],
	}

	// Prevent Heroku from crashing by binding port to server.
	// go func() {
	// 	http.ListenAndServe(
	// 		fmt.Sprintf(":%d", utils.ToIntOrPanic(config[envPort])),
	// 		http.HandlerFunc(http.NotFound))
	// }()

	go bot.Start()
	log.Println("Bot is running 🤖")

	go server.Start()
	log.Println("Server is running 💻")

	// Gracefully shutdown when Heroku issues a SIGTERM due to dyno cycling.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	s := <-signals
	log.Println("Gracefully shutting down with signal:", s)
	os.Exit(0)
}
