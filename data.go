package main

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
