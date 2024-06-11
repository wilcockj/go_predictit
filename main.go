package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const MinPercentChange = 0.02

var (
	cache           []NegRiskData
	cacheMutex      sync.Mutex
	cacheLastUpdate time.Time
	cacheDuration   = 120 * time.Second
)

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

func go_get_neg_risk(market Market, wg *sync.WaitGroup, resultArr []NegRiskData, index int) {
	defer wg.Done()
	resultArr[index] = GetNegativeRisk(market)
}

// postDataToURL posts string data to a specified URL and returns the response
func postDataToURL(url string, data string) (string, error) {
	// Create a new HTTP request. Use strings.NewReader to convert the string to an io.Reader
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return "", err
	}

	// Optionally, you can set the Content-Type header if needed (e.g., for plain text, use "text/plain")
	// req.Header.Set("Content-Type", "text/plain")

	// Create an HTTP client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func PostMessageToNtfy(Message string) {
	url := "https://ntfy.sh/predictitjohn"

	if Message != "" {
		response, err := postDataToURL(url, Message)
		if err != nil {
			fmt.Printf("Error posting data: %s\n", err)
			return
		}

		fmt.Printf("Response from server: %s\n", response)
	}

}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return !info.IsDir()
}

func GetMessageFromData(neg_risk []NegRiskData) string {
	var compare []NegRiskData
	GobExists := false
	if fileExists("neg_risk.gob") {
		GobExists = true
		file, err := os.Open("neg_risk.gob")
		if err != nil {
			log.Printf("Failed to open file: %v", err)
		}
		defer file.Close()

		// Create a new gob decoder.
		decoder := gob.NewDecoder(file)

		// Create a variable to hold the decoded data.

		// Decode the data into the variable.
		if err := decoder.Decode(&compare); err != nil {
			log.Printf("Failed to decode: %v", err)
			GobExists = false // failed to read gob
		}
	}

	var Message string
	fmtstring := "URL: %s has guaranteed profit: %.2f\n"

	for i, e := range neg_risk {
		fmt.Println(e)
		if e.LeastProfit >= 0 {
			// URL: has guaranteed profit:
			if !GobExists {
				Message += fmt.Sprintf(fmtstring, e.URL, e.LeastProfit)
			} else if e.LeastProfit > compare[i].LeastProfit*(1.0+MinPercentChange) || e.LeastProfit < compare[i].LeastProfit*(1.0-MinPercentChange) {
				Message += fmt.Sprintf(fmtstring, e.URL, e.LeastProfit)
			}
			fmt.Printf(fmtstring, e.URL, e.LeastProfit)
		}
	}

	return Message
}

func GetNegRiskFromPredictIt() []NegRiskData {
	var wg sync.WaitGroup

	body, err := get_json_response("https://www.predictit.org/api/marketdata/all/")
	if err != nil {
		log.Fatal(err)
	}

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
	return neg_risk

}

func SaveNegRiskToGob(neg_risk []NegRiskData) {
	file, err := os.Create("neg_risk.gob")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Create a new gob encoder.
	encoder := gob.NewEncoder(file)

	// Encode the struct and write to the file.
	if err := encoder.Encode(neg_risk); err != nil {
		log.Fatalf("Failed to encode struct: %v", err)
	}

}

func updateCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache = GetNegRiskFromPredictIt()
	cacheLastUpdate = time.Now()
}

func updateCachePeriodically() {
	for {
		updateCache()
		time.Sleep(cacheDuration)
	}
}

func NegRiskHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cache)
}

func main() {

	nonotifyPtr := flag.Bool("nonotify", false, "controls whether notifications will be sent to ntfy")
	flag.Parse()

	neg_risk := GetNegRiskFromPredictIt()

	Message := GetMessageFromData(neg_risk)

	if *nonotifyPtr {
		PostMessageToNtfy(Message)
	}

	SaveNegRiskToGob(neg_risk)

	go updateCachePeriodically()

	http.HandleFunc("/api/negrisk", NegRiskHandler)
	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
