package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	log.SetFlags(0)
}

func main() {
	var (
		listenAddress = flag.String("web.listen-address", ":9999", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		mqttBroker    = flag.String("mqtt.broker", "tcp://localhost:1883", "MQTT broker URI.")
		namespace     = flag.String("prometheus.namespace", "mqtt", "Prometheus metric namespace (prefix).")
		configFile    = flag.String("config.file", "mqtt.yml", "Path to configuration file.")
		debug         = flag.Bool("debug", false, "Enable debug logging.")
	)

	flag.Parse()

	// Load configuration, parse templates

	log.Println("Loading configuration file", *configFile)

	cfg, err := loadConfig(*configFile, *namespace)
	if err != nil {
		log.Fatalln("Error loading configuration file:", err)
	}

	// Create handlers with metrics and compile templates / regexes

	topics := make([]string, len(cfg))
	handlers := make([]mqtt.MessageHandler, len(cfg))

	i := 0
	for topic, metrics := range cfg {
		topics[i] = topic
		handler, err := NewMessageHandler(metrics, *namespace)
		if err != nil {
			log.Fatalln("Error creating handler:", err)
		}
		handlers[i] = handler
		i++
	}

	// Initialize MQTT, connect to broker

	if *debug {
		mqtt.CRITICAL = log.New(os.Stderr, "MQTT  CRIT: ", 0)
		mqtt.ERROR = log.New(os.Stderr, "MQTT ERROR: ", 0)
		mqtt.WARN = log.New(os.Stderr, "MQTT  WARN: ", 0)
		mqtt.DEBUG = log.New(os.Stdout, "MQTT DEBUG: ", 0)
	}

	options := mqtt.NewClientOptions().AddBroker(*mqttBroker)
	if username := os.Getenv("MQTT_USERNAME"); username != "" {
		options.SetUsername(username)
	}
	if password := os.Getenv("MQTT_PASSWORD"); password != "" {
		options.SetPassword(password)
	}

	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalln("Error connecting to MQTT broker:", token.Error())
	}

	// Set up metrics and subscriptions
	for i, topic := range topics {
		client.Subscribe(topic, 0, handlers[i])
	}

	// Start HTTP server
	http.Handle(*metricsPath, promhttp.Handler())
	log.Println("Listening on", *listenAddress)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		 <head><title>IoT Exporter</title></head>
		 <body>
		 <h1>IoT Exporter</h1>
		 <p><a href='` + *metricsPath + `'>Metrics</a></p>
		 </body>
		 </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
