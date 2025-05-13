package mqttclient

import (
	"ServerRoom/internal/storage"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func GetPhoneNumberFromDB() (string, error) {
	const query = `SELECT phonenumber FROM phones LIMIT 1`
	var phone string
	err := storage.DbPool.QueryRow(context.Background(), query).Scan(&phone)
	if err != nil {
		return "", fmt.Errorf("error fetching phone number: %w", err)
	}
	return phone, nil
}

var Phone string

const (
	TopicDoor        = "topic/door"
	TopicFire        = "topic/fire"
	TopicMotion      = "topic/motion"
	TopicTemperature = "topic/temprature"
	TopicHumidity    = "topic/humintity"
)

var topics = []string{
	TopicDoor,
	TopicFire,
	TopicMotion,
	TopicTemperature,
	TopicHumidity,
}

type Config struct {
	TempMin     float64
	TempMax     float64
	HumidityMin float64
	HumidityMax float64
}

var config = Config{
	TempMin:     getEnvAsFloat("TEMP_MIN", 18.0),
	TempMax:     getEnvAsFloat("TEMP_MAX", 27.0),
	HumidityMin: getEnvAsFloat("HUMIDITY_MIN", 30.0),
	HumidityMax: getEnvAsFloat("HUMIDITY_MAX", 60.0),
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		} else {
			log.Printf("Invalid float for env %s: %s, using default %.2f", key, val, defaultVal)
		}
	}
	return defaultVal
}

func Start(broker string) {
	var err error
	Phone, err = GetPhoneNumberFromDB()
	if err != nil {
		log.Printf("Phone number fetch error: %v, defaulting to fallback number", err)
		Phone = "+99364936679"
	}
	log.Println("Using phone number:", Phone)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("GoFiberMQTTClient")
	opts.OnConnect = func(c mqtt.Client) {
		for _, topic := range topics {
			if token := c.Subscribe(topic, 0, messageHandler); token.Wait() && token.Error() != nil {
				log.Printf("Topic subscription error: %s: %v", topic, token.Error())
			}
		}
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	for retries := 0; retries < 5; retries++ {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("MQTT Connection error: %v, retrying in 5s...", token.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
}

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()

	topic := msg.Topic()
	payload := string(msg.Payload())
	timestamp := time.Now().UTC().Format(time.RFC3339)

	wsMessage := map[string]string{
		"topic":     topic,
		"value":     payload,
		"timestamp": timestamp,
	}

	storage.SensorData[topic] = wsMessage

	jsonMessage, err := json.Marshal(wsMessage)
	if err != nil {
		log.Println("JSON formatting error:", err)
		return
	}

	select {
	case storage.BroadcastCh <- string(jsonMessage):
	case <-time.After(1 * time.Second):
		log.Println("Broadcast timeout, dropping message")
	}

	checkForRisk(topic, payload, timestamp)
}

func checkForRisk(topic, payload, timestamp string) {
	switch topic {
	case TopicTemperature:
		handleTemperature(payload, timestamp)
	case TopicHumidity:
		handleHumidity(payload, timestamp)
	case TopicFire:
		handleFire(payload, timestamp)
	case TopicDoor:
		handleDoor(payload, timestamp)
	case TopicMotion:
		handleMotion(payload, timestamp)
	}
}

func saveEventToDB(topic, value, timestamp string) {
	for retries := 0; retries < 3; retries++ {
		if err := storage.SaveEventToDB(topic, value, timestamp); err != nil {
			log.Printf("Database save error: %v, retrying...", err)
			time.Sleep(time.Second * time.Duration(retries+1))
			continue
		}
		return
	}
	log.Println("Failed to save to database after retries")
}

var (
	smsSuccessCount int
	smsFailureCount int
	lastSMSSent     = make(map[string]time.Time)
)

func shouldSendSMS(key string, cooldown time.Duration) bool {
	now := time.Now()
	if last, ok := lastSMSSent[key]; ok {
		if now.Sub(last) < cooldown {
			return false
		}
	}
	lastSMSSent[key] = now
	return true
}

func sendSMS(number, message string) {

	go func() {
		number, err := GetPhoneNumberFromDB()
		if err != nil {
			fmt.Printf("Error fetching phone number: %v\n", err)
			return
		}
		cmd := exec.Command("gammu", "sendsms", "TEXT", number, "-text", message)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(number)
			log.Printf("SMS sending error: %v, Output: %s", err, string(output))
			smsFailureCount++
			return
		}

		if strings.Contains(string(output), "Message reference") {
			log.Println("SMS sent successfully.")
			smsSuccessCount++
		} else {
			log.Printf("SMS sending failed. Output: %s", string(output))
			smsFailureCount++
		}

		log.Printf("SMS Sent: Success %d, Failure %d, Phone %s", smsSuccessCount, smsFailureCount, number)
	}()
}

func handleTemperature(payload, timestamp string) {
	temp, err := strconv.ParseFloat(payload, 64)
	if err != nil || temp < -50 || temp > 100 {
		log.Printf("Invalid temperature data: '%s', err: %v", payload, err)
		return
	}
	value := fmt.Sprintf("%.2f", temp)
	if temp > config.TempMax || temp < config.TempMin {
		saveEventToDB(TopicTemperature, value, timestamp)
	}
}

func handleHumidity(payload, timestamp string) {
	cleaned := strings.TrimSuffix(payload, "%")
	humidity, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || humidity < 0 || humidity > 100 {
		log.Printf("Invalid humidity data: '%s', err: %v", payload, err)
		return
	}
	value := fmt.Sprintf("%.2f", humidity)
	if humidity > config.HumidityMax || humidity < config.HumidityMin {
		saveEventToDB(TopicHumidity, value, timestamp)
	}
}

func handleFire(payload, timestamp string) {
	fire, err := strconv.Atoi(payload)
	if err != nil || (fire != 0 && fire != 1) {
		log.Printf("Invalid fire sensor data: '%s'", payload)
		return
	}
	saveEventToDB(TopicFire, strconv.Itoa(fire), timestamp)

	if fire == 1 && shouldSendSMS("fire", 1*time.Minute) {
		sendSMS(Phone, "Server otagynda ýangyn ýüze çykdy! Gözegçilik ediň.")
	} else if fire == 0 {
		sendSMS(Phone, "Server otagyndaky ýangyn ýagdaýy adaty ýagdaýa geldi.")
	}
}

func handleDoor(payload, timestamp string) {
	door, err := strconv.Atoi(payload)
	if err != nil || (door != 0 && door != 1) {
		log.Printf("Invalid door sensor data: '%s'", payload)
		return
	}
	saveEventToDB(TopicDoor, strconv.Itoa(door), timestamp)

	if door == 1 && shouldSendSMS("door", 1*time.Minute) {
		sendSMS(Phone, "Server otagynyň gapysy açyldy! Gözegçilik ediň.")
	} else if door == 0 {
		sendSMS(Phone, "Server otagynyň gapysy ýapyldy.")
	}
}

func handleMotion(payload, timestamp string) {
	motion, err := strconv.Atoi(payload)
	if err != nil || (motion != 0 && motion != 1) {
		log.Printf("Invalid motion sensor data: '%s'", payload)
		return
	}
	saveEventToDB(TopicMotion, strconv.Itoa(motion), timestamp)

	if motion == 1 && shouldSendSMS("motion", 1*time.Minute) {
		sendSMS(Phone, "Server otagynda hereket bar! Gözegçilik ediň.")
	}
}
