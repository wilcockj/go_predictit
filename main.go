package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type MarketData struct {
	Markets []Market `json:"markets"`
}
type Contract struct {
	ID              int     `json:"id"`
	DateEnd         string  `json:"dateEnd"`
	Image           string  `json:"image"`
	Name            string  `json:"name"`
	ShortName       string  `json:"shortName"`
	Status          string  `json:"status"`
	LastTradePrice  float64 `json:"lastTradePrice"`
	BestBuyYesCost  float64 `json:"bestBuyYesCost"`
	BestBuyNoCost   float64 `json:"bestBuyNoCost"`
	BestSellYesCost float64 `json:"bestSellYesCost"`
	BestSellNoCost  float64 `json:"bestSellNoCost"`
	LastClosePrice  float64 `json:"lastClosePrice"`
	DisplayOrder    int     `json:"displayOrder"`
}
type Market struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	ShortName string     `json:"shortName"`
	Image     string     `json:"image"`
	URL       string     `json:"url"`
	Contracts []Contract `json:"contracts"`
	TimeStamp string     `json:"timeStamp"`
	Status    string     `json:"status"`
}

type NegRiskData struct {
	ContractsToBuy int
	LowestNo       float64
	HighestNo      float64
	NegRisk        float64
	LeastProfit    float64
	MaxProfit      float64
	URL            string
}

func get_json_response(url string) ([]byte, error) {
	// Create a new request using http
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, errors.New("Failed to create request")
	}

	// Set the request header to accept only JSON
	req.Header.Set("Accept", "application/json")

	// Make the request using the http client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, errors.New("Failed to make request")
	}
	defer resp.Body.Close()

	// Check if the response status code is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Non-OK HTTP status:", resp.StatusCode)
		return nil, errors.New("Response code not OK")
	}

	// Read the body of the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, errors.New("Error reading response")
	}
	return body, nil

}

func GetNegativeRisk(market Market) NegRiskData {
	// for contract in market
	// get bestBuyNoCost 1-that summed
	// and then return that * 850(max bet)/cost of most expensive no contract

	var data NegRiskData
	max_no_price := 0.0
	lowest_no_price := 1.0
	neg_risk := 0.0
	for _, contract := range market.Contracts {
		if contract.BestBuyNoCost > float64(max_no_price) {
			max_no_price = contract.BestBuyNoCost
		}

		if contract.BestBuyNoCost < float64(lowest_no_price) && contract.BestBuyNoCost > 0.00 {
			lowest_no_price = contract.BestBuyNoCost
		}

		if contract.BestBuyNoCost < 0.01 {
			continue
		}
		neg_risk += 1 - contract.BestBuyNoCost
	}
	data.NegRisk = neg_risk

	// least risk is profit from all other contracts
	// - price of most expensive * 0.9
	var MaxContractBetTotal float64 = 850
	ProfitFee := 0.1
	ContractsCount := MaxContractBetTotal / max_no_price

	HighestLoss := max_no_price * ContractsCount
	LowestLoss := lowest_no_price * ContractsCount

	data.LeastProfit = ((neg_risk-(1.0-max_no_price))*(ContractsCount)*(1-ProfitFee) - HighestLoss)
	data.MaxProfit = ((neg_risk-(1.0-lowest_no_price))*(ContractsCount)*(1-ProfitFee) - LowestLoss)
	data.ContractsToBuy = int(ContractsCount)
	data.URL = market.URL
	data.LowestNo = lowest_no_price
	data.HighestNo = max_no_price

	return data
}

func go_get_neg_risk(market Market, wg *sync.WaitGroup, resultArr []NegRiskData, index int) {
	defer wg.Done()
	resultArr[index] = GetNegativeRisk(market)
}

func main() {

	var wg sync.WaitGroup

	body, err := get_json_response("https://www.predictit.org/api/marketdata/all/")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("Response body: %s\n", body)
	var MarketsData MarketData
	err = json.Unmarshal(body, &MarketsData)
	if err != nil {
		log.Fatalf("Error parsing JSON: %s", err)
	}

	resultArr := make([]NegRiskData, len(MarketsData.Markets))

	for i, market := range MarketsData.Markets {
		wg.Add(1)
		go go_get_neg_risk(market, &wg, resultArr, i)
	}

	wg.Wait()

	var neg_risk []NegRiskData
	for _, nr := range resultArr {
		neg_risk = append(neg_risk, nr)
	}

	for _, e := range neg_risk {
		fmt.Println(e)
	}

	/*
		prettyJSON, err := json.MarshalIndent(MarketsData, "", "    ")
		if err != nil {
			log.Fatalf("Failed to generate pretty print JSON: %s", err)
		}
		fmt.Printf("Pretty JSON:\n%s\n", prettyJSON)
	*/
}
