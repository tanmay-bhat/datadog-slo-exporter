# DataDog SLO Exporter

This project exports Service Level Objective (SLO) data from DataDog and pushes it to a Prometheus-compatible endpoint.

## Overview

The DataDog SLO Exporter is a Go application that:
1. Fetches SLO history from DataDog
2. Converts the data into Prometheus metrics
3. Pushes these metrics to a specified Prometheus endpoint

## Prerequisites

- Go 1.15 or higher
- DataDog account with API and application keys
- Prometheus Pushgateway or VictoriaMetrics server

## Setup

1. Clone the repository:

```
git clone https://github.com/your-username/datadog-slo-exporter.git
cd datadog-slo-exporter
```

2. Set up environment variables:

```
export DD_API_KEY=your_datadog_api_key
export DD_APP_KEY=your_datadog_app_key
export DD_SLO_ID=your_slo_id
export PROMETHEUS_ENDPOINT=http://localhost:9091
```

## Usage

Run the application:

```
PROMETHEUS_ENDPOINT="your_prom_push_endpoint" DD_API_KEY="xxx" DD_APP_KEY="xxx" go run main.go
```

The application will fetch SLO data for 7, 30, and 90-day windows and push the metrics to the specified Prometheus endpoint.

## Metrics

- `datadog_slo_uptime`: Gauge metric for SLO uptime
- `datadog_api_error_total`: Counter for DataDog API errors