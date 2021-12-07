# iot_exporter

MQTT metric exporter for Prometheus Monitoring

This is a simple [Prometheus](https://prometheus.io/) exporter for data exposed over MQTT.

* Supports arbitrary topic names and payload data structures.
* Payload has to be a number, string or JSON structure.
* All metrics need to be configured in a simple YAML configuration file.
* Only supports gauges for now, metrics are simply cached in `prometheus.GaugeVec` instances.

## Building

To build iot_exporter, the [Go distribution](https://golang.org/doc/install) is needed.

```bash
go get -u github.com/markuslindenberg/iot_exporter
```

## Installation

See `iot_exporter -h` for usage. For authenticated MQTT connections the environment variables `MQTT_USERNAME` and `MQTT_PASSWORD` must be provided.

## Configuration

Every topic pattern to subscribe and the resulting prometheus metrics have to be configured in a simple YAML configuration file, containing of just a map of topics as keys and a list of metric configurations as value.

### Example

```yaml
'zigbee2mqtt/+':
  - name: zigbee_link_quality
    match: '^zigbee2mqtt/([^/]+)$'
    value: '{{ .Payload.linkquality }}'
    labels:
      instance: '{{ index .Matches 1 }}'
  - name: zigbee_update_available
    match: '^zigbee2mqtt/([^/]+)$'
    value: '{{ if .Payload.update_available }}1{{ else }}0{{ end }}'
    labels:
      instance: '{{ index .Matches 1 }}'
'esphome/+/sensor/+/state':
  - name: sensor_temperature_celsius
    match: '^esphome/([^/]+)/sensor/temperatur_([^/]+)/state$'
    labels:
      instance: '{{ index .Matches 1 }}'
      sensor: '{{ index .Matches 2 }}'
'emsesp/+/boiler_data':
  - name: ems_boiler_ignition_on
    match: '^emsesp/([^/]+)/boiler_data$'
    value: '{{ if eq .Payload.ignWork "on" }}1{{ else }}0{{ end }}'
    labels:
      instance: '{{ index .Matches 1 }}'
```

### Metrics

Metric configurations can contain the following keys:

* `name` (string) is the resulting prometheus metric name. It will be prefixed by the namespace given on the command line using `-prometheus.namespace` (default namespace is `mqtt`).
* `match` (string, regex, optional) is a [RE2 regular expression](https://github.com/google/re2/wiki/Syntax) that filters topics and optionally captures subgroups for use in value / label templates. When `match` is empty, default is to not filter messages.
* `value` (string, template, optional) is a string containing a [Go template](https://golang.org/pkg/text/template/) + [Sprig](http://masterminds.github.io/sprig/) to extract a value from the MQTT message payload. The result of the rendered template should be parseable by [strconv.ParseFloat](https://golang.org/pkg/strconv/#ParseFloat). When no value template is given, the raw payload will be parsed.
  * Template context: Inside the template, the Variables `{{ .Payload }}` and `{{ .Matches }}` are available.
    * `.Payload` contains the parsed message payload: A float64 if the payload can be parsed as number, a data structure if the payload can be parsed as JSON or a string otherwise.
    * `.Matches` contains the matched topic (at index 0) and all groups captured using `()` from index 1 on.
* `labels` (map, template, optional) is a map containing labels to be added to the metrics. The label values can be templates using the same data available to the `value` template.
