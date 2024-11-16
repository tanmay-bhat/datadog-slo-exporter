package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Response struct {
	Data Data `json:"data"`
}

type Data struct {
	Overall Overall `json:"overall"`
	Slo     Slo     `json:"slo"`
}

type Overall struct {
	Name string  `json:"name"`
	SLI  float64 `json:"sli_value"`
}

type Slo struct {
	Threshold float64 `json:"target_threshold"`
	Timeframe string  `json:"timeframe"`
}

var (
	DD_API_KEY          = os.Getenv("DD_API_KEY")
	DD_APP_KEY          = os.Getenv("DD_APP_KEY")
	DD_SLO_ID           = os.Getenv("DD_SLO_ID")
	PROMETHEUS_ENDPOINT = os.Getenv("PROMETHEUS_ENDPOINT")

	ctx       context.Context
	api       *datadogV1.ServiceLevelObjectivesApi
	apiClient *datadog.APIClient

	// Declare metrics
	DataDogSLOGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "datadog",
		Name:      "slo_uptime",
		Help:      "History details of a Datadog SLO"},
		[]string{"slo_name", "threshold", "window", "rolling_timeframe"},
	)

	DataDogAPIErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "datadog",
		Name:      "api_error_total",
		Help:      "Total Error count on requests to DataDog API"},
		[]string{"api_call", "status_code"},
	)

	logger *log.Logger
)

func InitDataDogClient() error {
	ctx = context.WithValue(context.Background(), datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {Key: DD_API_KEY},
			"appKeyAuth": {Key: DD_APP_KEY},
		},
	)

	configuration := datadog.NewConfiguration()
	configuration.RetryConfiguration.EnableRetry = true // defaults to 3 retries
	apiClient = datadog.NewAPIClient(configuration)
	api = datadogV1.NewServiceLevelObjectivesApi(apiClient)
	return nil
}

func GetSloData(sloDataID string, daysWindow int) (float64, string, float64, string, error) {
	fromTime := time.Now().AddDate(0, 0, -daysWindow).Unix()
	toTime := time.Now().Unix()

	resp, r, err := api.GetSLOHistory(ctx, sloDataID, fromTime, toTime)
	if err != nil {
		statusCode := 0
		if r != nil {
			statusCode = r.StatusCode
		}
		DataDogAPIErrorCounter.WithLabelValues("GetSLOHistory", strconv.Itoa(statusCode)).Inc()
		return 0, "", 0, "", fmt.Errorf("error when calling `ServiceLevelObjectivesApi.GetSLOHistory`: %v", err)
	}

	responseContent, err := json.Marshal(resp)
	if err != nil {
		return 0, "", 0, "", fmt.Errorf("error marshaling response: %v", err)
	}

	var response Response
	if err = json.Unmarshal(responseContent, &response); err != nil {
		return 0, "", 0, "", fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return response.Data.Overall.SLI, response.Data.Overall.Name, response.Data.Slo.Threshold, response.Data.Slo.Timeframe, nil
}

func main() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	if DD_API_KEY == "" || DD_APP_KEY == "" || DD_SLO_ID == "" || PROMETHEUS_ENDPOINT == "" {
		logger.Fatal("Required environment variables (DD_API_KEY, DD_APP_KEY, DD_SLO_ID, PROMETHEUS_ENDPOINT) are not set")
	}

	err := InitDataDogClient()
	if err != nil {
		logger.Fatalf("Failed to initialize Datadog client: %v\n", err)
		return
	}
	logger.Printf("Datadog client initialized successfully\n")

	// Use a custom registry for consistency and to drop default go metrics
	registry := prometheus.NewRegistry()
	registry.MustRegister(DataDogSLOGauge, DataDogAPIErrorCounter)

	days := []int{7, 30, 90}

	for _, day := range days {
		logger.Printf("Fetching SLO history for %d days\n", day)
		sliValue, sloName, threshold, rollingTimeframe, err := GetSloData(DD_SLO_ID, day)

		if err != nil {
			logger.Printf("Error getting SLO data: %v\n", err)
		} else {
			g := DataDogSLOGauge.WithLabelValues(
				sloName,
				fmt.Sprintf("%.2f", threshold),
				fmt.Sprintf("%dd", day),
				rollingTimeframe,
			)
			g.Set(sliValue)
		}
	}

	pusher := push.New(PROMETHEUS_ENDPOINT, "datadog-slo-exporter").Gatherer(registry)

	if err := pusher.Push(); err != nil {
		logger.Printf("Error pushing metrics to VictoriaMetrics: %v\n", err)
	} else {
		logger.Printf("Metrics pushed to VictoriaMetrics successfully\n")
	}
}
