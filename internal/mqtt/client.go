package mqttclient

import (
	"ServerRoom/internal/storage"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var Phone = "+99364936679"

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
		}
	}
	return defaultVal
}

func Start(broker string) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("GoFiberMQTTClient")

	opts.OnConnect = func(c mqtt.Client) {
		for _, topic := range topics {
			if token := c.Subscribe(topic, 0, messageHandler); token.Wait() && token.Error() != nil {
				fmt.Println("Topic subscription error:", topic, token.Error())
			}
		}
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		fmt.Printf("MQTT connection lost: %v\n", err)
	}

	client := mqtt.NewClient(opts)

	for retries := 0; retries < 5; retries++ {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			fmt.Printf("MQTT Connection error: %v, retrying in 5s...\n", token.Error())
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
		fmt.Println("JSON formatting error:", err)
		return
	}

	select {
	case storage.BroadcastCh <- string(jsonMessage):
	default:
		fmt.Println("Broadcast channel full, dropping message")
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
			fmt.Printf("Database save error: %v, retrying...\n", err)
			time.Sleep(time.Second * time.Duration(retries+1))
			continue
		}
		return
	}
	fmt.Println("Failed to save to database after retries")
}

var smsSuccessCount int
var smsFailureCount int

func sendSMS(number, message string) {
	// Gammu komutunun doğru şekilde yapılandırılması
	cmd := exec.Command("gammu", "--device", "/dev/ttyUSB3", "--sendsms", "TEXT", number, "-text", message)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Eğer bir hata olursa, hata mesajını yazdır
		fmt.Printf("Failed to send SMS: %v, Output: %s\n", err, string(output))
		smsFailureCount++ // Başarısız SMS sayısını artır
		return
	}

	// Eğer mesaj gönderildiyse, "Message reference" kelimesi çıktıda olmalı
	if strings.Contains(string(output), "Message reference") {
		fmt.Println("SMS sent successfully")
		smsSuccessCount++ // Başarılı SMS sayısını artır
	} else {
		// Eğer mesaj gönderilemediyse, çıktıyı yazdır
		fmt.Printf("Failed to send SMS. Output: %s\n", string(output))
		smsFailureCount++ // Başarısız SMS sayısını artır
	}

	// Başarı ve başarısızlık sayılarını yazdır
	fmt.Printf("SMS Sent: Success %d, Failure %d\n", smsSuccessCount, smsFailureCount)
}

func handleTemperature(payload, timestamp string) {
	temp, err := strconv.ParseFloat(payload, 64)
	if err != nil || temp < -50 || temp > 100 {
		fmt.Println("Invalid temperature data:", payload)
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
		fmt.Println("Invalid humidity data:", payload)
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
		fmt.Println("Invalid fire sensor data:", payload)
		return
	}
	saveEventToDB(TopicFire, strconv.Itoa(fire), timestamp)

	if fire == 1 {
		sendSMS(Phone, "🚨 Server otagynda ýangyn ýüze çykdy! Gözegçilik ediň.")
	} else {
		sendSMS(Phone, "✅ Server otagyndaky ýangyn ýagdaýy adaty ýagdaýa geldi.")
	}
}

func handleDoor(payload, timestamp string) {
	door, err := strconv.Atoi(payload)
	if err != nil || (door != 0 && door != 1) {
		fmt.Println("Invalid door sensor data:", payload)
		return
	}
	saveEventToDB(TopicDoor, strconv.Itoa(door), timestamp)

	if door == 1 {
		sendSMS(Phone, "📢 Server otagynyň gapysy açyldy! Gözegçilik ediň.")
	} else {
		sendSMS(Phone, "✅ Server otagynyň gapysy ýapyldy.")
	}
}

func handleMotion(payload, timestamp string) {
	motion, err := strconv.Atoi(payload)
	if err != nil || (motion != 0 && motion != 1) {
		fmt.Println("Invalid motion sensor data:", payload)
		return
	}
	saveEventToDB(TopicMotion, strconv.Itoa(motion), timestamp)

	if motion == 1 {
		sendSMS(Phone, "⚠️ Server otagynda hereket bar! Gözegçilik ediň.")
	}
}
