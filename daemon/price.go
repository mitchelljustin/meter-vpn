package daemon

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

const (
	MonthlyCostUSD = 4.00
	DailyCostUSD   = MonthlyCostUSD / 30.5
	HourlyCostUSD  = DailyCostUSD / 24
)

type PriceTracker struct {
	LatestUSDBTC float64
}

type durationSnapshot struct {
	Month float64 `json:"month"`
	Day   float64 `json:"day"`
	Hour  float64 `json:"hour"`
}

type PricesSnapshot struct {
	Satoshi durationSnapshot `json:"satoshi"`
	USD     durationSnapshot `json:"usd"`
}

type coindeskCurrentPrice struct {
	Bpi *struct {
		USD *struct {
			RateFloat *float64 `json:"rate_float"`
		} `json:"USD"`
	}
}

func priceToSatoshi(priceUsd, rate float64) float64 {
	return priceUsd / rate * 1e8
}

func (pt *PriceTracker) UpdateLatestRate() error {
	resp, err := http.Get("https://api.coindesk.com/v1/bpi/currentprice/USD.json")
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	var curPrice coindeskCurrentPrice
	if err := json.NewDecoder(resp.Body).Decode(&curPrice); err != nil {
		return err
	}
	if curPrice.Bpi == nil || curPrice.Bpi.USD == nil || curPrice.Bpi.USD.RateFloat == nil {
		return errors.New("got nil from JSON decoded BTC price")
	}
	pt.LatestUSDBTC = *curPrice.Bpi.USD.RateFloat
	return nil
}

func (pt *PriceTracker) RetrieveSnapshot() PricesSnapshot {
	if err := pt.UpdateLatestRate(); err != nil {
		log.Printf("Error getting latest rate: %v, using stale rate", err)
	}
	return PricesSnapshot{
		Satoshi: durationSnapshot{
			Month: priceToSatoshi(MonthlyCostUSD, pt.LatestUSDBTC),
			Day:   priceToSatoshi(DailyCostUSD, pt.LatestUSDBTC),
			Hour:  priceToSatoshi(HourlyCostUSD, pt.LatestUSDBTC),
		},
		USD: durationSnapshot{
			Month: MonthlyCostUSD,
			Day:   DailyCostUSD,
			Hour:  HourlyCostUSD,
		},
	}
}
