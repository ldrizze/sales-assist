package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	// Set timezone
	// time.Local, _ = time.LoadLocation("America/Sao_Paulo")

	// Get and validate flags
	var rabbitMqUri string
	flag.StringVar(&rabbitMqUri, "amqp", "-1", "the rabbitmq connection uri")

	var exchangeName string
	flag.StringVar(&exchangeName, "exchange", "-1", "the rabbitmq exchange name for routing")

	var openAIToken string
	flag.StringVar(&openAIToken, "openaitoken", "-1", "openai api key")

	var instanceId string
	flag.StringVar(&instanceId, "instanceid", "-1", "whatsapp instance id key")

	var evolutionApiKey string
	flag.StringVar(&evolutionApiKey, "evotoken", "-1", "evolution api auth token")

	var evoUrl string
	flag.StringVar(&evoUrl, "evourl", "-1", "evolution api url")

	var calendarEmail string
	flag.StringVar(&calendarEmail, "email", "-1", "google calendar email")

	var cronParams string
	flag.StringVar(&cronParams, "cron", "*/5 * * * *", "event list harvest cron")

	var pg string
	flag.StringVar(&pg, "pg", "-1", "postgres connection url")

	var ownerNumber string
	flag.StringVar(&ownerNumber, "number", "-1", "whatsapp number of owner")

	var forMe bool
	flag.BoolVar(&forMe, "me", false, "enable conversation in same number as instance")

	flag.Parse()

	// assert flags
	assertFlag(rabbitMqUri, "amqp://.*", "amqp")
	checkMissingFlag(exchangeName, "exchange")
	checkMissingFlag(openAIToken, "openaitoken")
	checkMissingFlag(instanceId, "instanceid")
	checkMissingFlag(evolutionApiKey, "evotoken")
	checkMissingFlag(ownerNumber, "number")
	assertFlag(evoUrl, "http.*", "evourl")
	assertFlag(pg, `(postgres(?:ql)?):\/\/(?:([^@\s]+)@)?([^\/\s]+)(?:\/(\w+))?(?:\?(.+))?`, "pg")

	// Catalog
	catalog, err := os.ReadFile("./Catalogo.pdf")
	failOnError(err, "Can't open catalog")

	// Initialize Vault Singleton
	Vault.OpenAIApiKey = openAIToken
	Vault.RabbitMQExchangeName = exchangeName
	Vault.InstanceID = instanceId
	Vault.EvolutionToken = evolutionApiKey
	Vault.EvolutionURL = evoUrl
	Vault.Conversations = map[string]*WhatsAppChat{}
	Vault.OwnerNumber = ownerNumber
	Vault.EnableForMe = forMe
	Vault.CatalogAttachment = catalog

	// Initilize scheduler
	// fmt.Println("Initializing scheduler")
	// PrepareCalendarClient()
	// cronScheduler, err := gocron.NewScheduler()
	// failOnError(err, "Failed to initialize cron scheduler")

	// _, err = cronScheduler.NewJob(
	// 	gocron.CronJob(cronParams, false),
	// 	gocron.NewTask(HarvestEventList),
	// )
	// failOnError(err, "Failed to initialize cron scheduler")
	// cronScheduler.Start()
	// defer cronScheduler.Shutdown()

	// Initialize system message
	fmt.Println("Loading default system message")
	systemMessage, err := os.ReadFile("./system-message.txt")
	failOnError(err, "Failed to open openai system message")
	Vault.SystemMessage = strings.ReplaceAll(string(systemMessage), "#today", time.Now().Format("02/01/2006"))

	// Initialize database
	fmt.Println("Connecting to database")
	Vault.PGX, err = pgx.Connect(context.Background(), pg)
	failOnError(err, "Failed to connect to database")
	defer Vault.PGX.Close(context.TODO())

	// Initialize RabbitMQ
	fmt.Println("Connecting to RabbitMQ")
	conn, err := amqp.Dial(rabbitMqUri)
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a communication channel")
	defer ch.Close()
	defer conn.Close()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Listen to RabbitMQ events
	// TODO listen for system maintence messages
	go listenMessagesUpsert(ch, exchangeName)

	fmt.Println("Application started. To exit press CTRL+C")
	sig := <-sigs
	fmt.Printf("\nReceived signal %s, exiting application.\n", sig)
}

func listenMessagesUpsert(ch *amqp.Channel, exchangeName string) {
	fmt.Println("Creating messages upsert queue and routing")
	queue, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to declare messages upsert queue")

	err = ch.QueueBind(
		queue.Name,
		"messages.upsert",
		exchangeName,
		false,
		nil,
	)
	failOnError(err, "Failed to bind the queue")

	msgs, err := ch.Consume(
		queue.Name,
		"",
		true,
		true,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to listen to messages upsert")

	for msg := range msgs {
		var conversation map[string]interface{}
		err = json.Unmarshal(msg.Body, &conversation)
		failOnError(err, "Can't unmarshal conversation")

		// TODO HOLY SHIT! make an struct for this \/!!!
		remoteJID := conversation["data"].(map[string]interface{})["key"].(map[string]interface{})["remoteJid"].(string)
		fromMe := conversation["data"].(map[string]interface{})["key"].(map[string]interface{})["fromMe"].(bool)

		// Skip self messages
		if fromMe && !Vault.EnableForMe {
			fmt.Println("Skipping self message")
			continue
		}

		// extract phone number
		reg, _ := regexp.Compile(`^(\d+)@.+`)
		match := reg.FindSubmatch([]byte(remoteJID))

		if match != nil {
			chat := GetOrCreateConversation(string(match[1]))

			messageType := conversation["data"].(map[string]interface{})["messageType"].(string)
			if messageType == "conversation" {
				remoteMessage := conversation["data"].(map[string]interface{})["message"].(map[string]interface{})["conversation"].(string)
				fmt.Printf("Received message from %s: %s\n", chat.Number, remoteMessage)
				chat.SendToOpenAI(remoteMessage, "user", nil)
			}

			if messageType == "imageMessage" || messageType == "documentMessage" {
				key := conversation["data"].(map[string]interface{})["key"].(map[string]interface{})["id"].(string)
				media := GetMediaBase64(key)

				if !slices.Contains([]string{"application/pdf", "image/jpeg", "image/jpg"}, media.MimeType) {
					fmt.Printf("Invalid file mime type %s", media.MimeType)
					chat.SendToOpenAI("O usuário enviou o comprovante porém não reconheci o formato.", "developer", nil)
					continue
				}

				chat.SendToOpenAI("comprovante de pagamento", "user", &media)
				chat.AllowSendReceipt = false
			}
		} else {
			fmt.Println("Warning: Can't extract the phone number from package")
		}
	}
}

func GetOrCreateConversation(phoneNumber string) *WhatsAppChat {
	if val, ok := Vault.Conversations[phoneNumber]; ok {
		return val
	}

	chat := &WhatsAppChat{
		Number:              phoneNumber,
		Messages:            []WhatsAppChatMessage{},
		LastInteractionTime: time.Now(),
		AllowSendReceipt:    false,
	}

	var suspendedData string
	err := Vault.PGX.QueryRow(context.Background(), "SELECT data FROM suspended_chats WHERE phone_number = $1", phoneNumber).Scan(&suspendedData)

	if err != nil && err != pgx.ErrNoRows {
		failOnError(err, "loading suspended chat error")
	}

	if suspendedData != "" {
		err := json.Unmarshal([]byte(suspendedData), &chat)
		failOnError(err, "error during unmarshal suspended data")
	}

	Vault.Conversations[phoneNumber] = chat
	return chat
}

func assertFlag(value string, must string, param string) {
	result, err := regexp.Match(must, []byte(value))
	failOnError(err, "Flag assertion regex error")

	if !result {
		failOnError(fmt.Errorf("assertion error %s", param), "Flag validation error")
	}
}

func checkMissingFlag(value string, param string) {
	if value == "-1" {
		failOnError(fmt.Errorf("missing flag %s", param), "Flag validation error")
	}
}
