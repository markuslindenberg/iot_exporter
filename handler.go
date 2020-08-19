package main

import (
	"bytes"
	"encoding/json"
	"log"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

type TemplateData struct {
	Payload interface{}
	Matches []string
}

func NewMessageHandler(metrics []*Metric, namespace string) (mqtt.MessageHandler, error) {
	var err error
	names := make([]string, len(metrics))
	regexps := make([]*regexp.Regexp, len(metrics))
	valueTemplates := make([]*template.Template, len(metrics))
	labelNames := make([][]string, len(metrics))
	labelTemplates := make([][]*template.Template, len(metrics))
	collectors := make([]*prometheus.GaugeVec, len(metrics))

	for i, metric := range metrics {
		names[i] = metric.Name

		// compile match regexp
		if metric.Match != "" {
			regexps[i], err = regexp.Compile(metric.Match)
			if err != nil {
				return nil, err
			}
		}

		// parse value template
		if metric.Value != "" {
			valueTemplates[i], err = template.New("value").Parse(metric.Value)
			if err != nil {
				return nil, err
			}
		}

		// parse label templates
		labelNames[i] = make([]string, 0, len(metric.Labels))
		labelTemplates[i] = make([]*template.Template, 0, len(metric.Labels))
		for label, tplString := range metric.Labels {
			labelNames[i] = append(labelNames[i], label)
			tpl, err := template.New("label").Parse(tplString)
			if err != nil {
				return nil, err
			}
			labelTemplates[i] = append(labelTemplates[i], tpl)
		}

		// create prometheus collector
		collectors[i] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      names[i],
		}, labelNames[i])

		err = prometheus.Register(collectors[i])
		if err != nil {
			return nil, err
		}

	}

	// return the handler function for mqtt
	return func(client mqtt.Client, msg mqtt.Message) {
		var payload interface{}
		var payloadIsFloat bool

		// try to parse payload as float64 or JSON, else use as string
		stringVal := string(msg.Payload())
		floatVal, err := strconv.ParseFloat(stringVal, 64)
		if err == nil {
			payload = floatVal
			payloadIsFloat = true
		} else {
			mapVal := make(map[string]interface{})
			err = json.Unmarshal(msg.Payload(), &mapVal)
			if err == nil {
				payload = mapVal
			} else {
				payload = stringVal
			}
		}

		// repeat for every configured metric
		for i := range names {

			// check if topic matches filter
			var matches []string
			if regexps[i] != nil {
				matches = regexps[i].FindStringSubmatch(msg.Topic())
				if matches == nil {
					continue
				}
			}

			// provide .Payload and .Matches to templates
			data := &TemplateData{
				Payload: payload,
				Matches: matches,
			}

			// render template for value, parse as float64
			var buf bytes.Buffer
			if valueTemplates[i] != nil {
				err = valueTemplates[i].Execute(&buf, data)
				if err != nil {
					log.Println(err)
					continue
				}
				floatVal, err = strconv.ParseFloat(strings.TrimSpace(buf.String()), 64)
				if err != nil {
					continue
				}
			} else {
				if !payloadIsFloat {
					// payload is not numeric and no value template was given: skip
					continue
				}
			}

			// render template(s) for labels
			labels := make(prometheus.Labels)
			for j, tpl := range labelTemplates[i] {
				buf.Reset()
				err = tpl.Execute(&buf, data)
				if err != nil {
					log.Println(err)
					continue
				}
				labels[labelNames[i][j]] = buf.String()
			}

			// update collector
			collectors[i].With(labels).Set(floatVal)
		}
	}, nil
}
