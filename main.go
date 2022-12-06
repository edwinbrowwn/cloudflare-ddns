package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	AuthEmail      string `json:"authEmail"`
	AuthKey        string `json:"authKey"`
	ZoneIdentifier string `json:"zoneIdentifier"`
	RecordName     string `json:"recordName"`
	EnableProxy    bool   `json:"proxy"`
}

type UpdateDNSBody struct {
	ZoneIdentifier string `json:"id"`
	RecordType     string `json:"type"`
	EnableProxy    bool   `json:"proxied"`
	RecordName     string `json:"name"`
	IPAddress      string `json:"content"`
	TTL            int16  `json:"ttl"`
}

// Get current addr
func getCurrentAddr() string {
	resp, err := http.Get("https://api.ipify.org/")

	if err != nil {
		log.Printf("Error fetching current ip addr: %s", err.Error())
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error fetching current ip addr: %s", err.Error())
		return ""
	}

	resp.Body.Close()

	return string(body)
}

func getPreviousAddr(name *string) string {
	file, err := os.Open(*name)
	if err != nil {
		log.Printf("Error opening ip addr file: %s", err.Error())
	}

	addr, err := ioutil.ReadAll(file)

	if err != nil {
		log.Printf("Error opening ip addr file: %s", err.Error())
		return ""
	}

	return string(addr)
}

func getDNSRecord(config *Config) string {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", config.ZoneIdentifier, config.RecordName), nil)

	if err != nil {
		log.Printf("Error fetching dns record: %s", err.Error())
		return ""
	}

	req.Header.Add("X-Auth-Email", config.AuthEmail)
	req.Header.Add("X-Auth-Key", config.AuthKey)
	req.Header.Add("Content-Type", "application/json")

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching dns record: %s", err.Error())
		return ""
	}

	d := json.NewDecoder(resp.Body)

	var resJSON map[string]interface{}
	err = d.Decode(&resJSON)

	if err != nil {
		log.Printf("Error fetching dns record: %s", err.Error())
		return ""
	}

	if resJSON["success"].(bool) {
		var result = resJSON["result"].([]interface{})
		if len(result) > 0 {
			var recordMap = result[0].(map[string]interface{})
			dnsRecord := recordMap["id"].(string)
			addr := strings.Trim(recordMap["content"].(string), "")
			log.Printf("Fetched ipv4 address: %s", addr)
			// update ip address to dns
			currentAddr := getCurrentAddr()
			err = updateDNSRecord(config, currentAddr, dnsRecord)
			if err != nil {
				log.Fatalf("Error updating dns record: %s", err.Error())
			}

			ipAddressBuffer := []byte(addr)
			err := ioutil.WriteFile(config.RecordName, ipAddressBuffer, 0644)
			if err != nil {
				log.Fatalf("Error writing to addr file: %s", err.Error())
			}

			return dnsRecord
		}
		log.Printf("Error fetching dns record: %s", err.Error())
		return ""
	}

	log.Fatalf("Error fetching dns record: %s", err.Error())

	resp.Body.Close()
	return ""
}

func updateDNSRecord(config *Config, ip string, dnsIdent string) error {
	var dNSUpdateRequest = UpdateDNSBody{
		IPAddress:      ip,
		EnableProxy:    config.EnableProxy,
		RecordName:     config.RecordName,
		RecordType:     "A",
		ZoneIdentifier: config.ZoneIdentifier,
		TTL:            120,
	}
	dNSUpdateRequestJSON, err := json.Marshal(dNSUpdateRequest)

	if err != nil {
		log.Printf("Error updating dns record: %s", err.Error())
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
		config.ZoneIdentifier,
		dnsIdent),
		bytes.NewBuffer(dNSUpdateRequestJSON))
	if err != nil {
		log.Printf("Error updating dns record: %s", err.Error())
		return err
	}

	req.Header.Add("X-Auth-Email", config.AuthEmail)
	req.Header.Add("X-Auth-Key", config.AuthKey)
	req.Header.Add("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching dns record: %s", err.Error())
		return err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var responseJSON map[string]interface{}
	err = decoder.Decode(&responseJSON)

	if err != nil {
		log.Printf("Error fetching dns record: %s", err.Error())
		return err
	}

	if !responseJSON["success"].(bool) {
		return fmt.Errorf("Error updating dns record: %v", responseJSON)
	}

	return nil
}

func tryUpdate(config *Config) {
	var err error
	// get current ip address
	currentAddr := getCurrentAddr()
	if currentAddr == "" {
		log.Fatalf("Error getting current ip: %s", err.Error())
	}
	log.Printf("Current public ipv4 address: %s", currentAddr)

	// get ip address previously set to cloudflare
	previousAddr := getPreviousAddr(&config.RecordName)
	// if previousAddr == "" {
	// 	log.Printf("Error getting previous ip: %s", previousAddr)
	// } else {
	// 	log.Printf("Current previous ipv4 address: %s", previousAddr)
	// }

	if strings.Trim(previousAddr, "") != strings.Trim(currentAddr, "") {
		dnsRecord := getDNSRecord(config)
		if dnsRecord == "" {
			log.Fatalf("Error getting dns record identifier: %s", err.Error())
		}
		log.Printf("DNS record id: %s", dnsRecord)
	} else {
		log.Println("Current and previous ip addresses match, exiting...")
	}
}

func main() {
	log.Println("Starting cloudflare-ddns...")
	var configs []Config
	var err error

	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config.json: %s", err.Error())
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configs)
	if err != nil {
		log.Fatalf("Error decoding config.json: %s", err.Error())
	}

	ticker := time.NewTicker(10000 * time.Millisecond) // Check for addr changes every 20 seconds
	pollAddr := time.NewTicker(300 * time.Second)      // Check for addr updates every 5 min
	done := make(chan bool)
	donePoll := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				for i := range configs {
					tryUpdate(&configs[i])
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-donePoll:
				return
			case <-pollAddr.C:
				for i := range configs {
					getDNSRecord(&configs[i])
				}
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for sig := range c {
		log.Println(sig.String())
		ticker.Stop()
		done <- true
		pollAddr.Stop()
		donePoll <- true
		log.Println("Stopped")
		break
	}

	log.Println("Exiting cloudflare-ddns")
}
