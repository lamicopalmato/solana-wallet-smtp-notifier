package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

const (
	pollInterval = 30 * time.Second
	lastTxFile   = "/opt/solana-notifier/last_tx.sig"
)

var (
	solanaRPC     = os.Getenv("SOLANA_RPC")
	walletAddress = os.Getenv("WALLET_ADDRESS")
	lastTxSig     string
)

type solanaResponse struct {
	Result []struct {
		Signature string  `json:"signature"`
		Slot      uint64  `json:"slot"`
		Err       *string `json:"err"`
		Memo      *string `json:"memo"`
		BlockTime *int64  `json:"blockTime"`
	} `json:"result"`
	ID int `json:"id"`
}

func readLastTx() string {
	data, err := os.ReadFile(lastTxFile)
	if err != nil {
		log.Println("No last tx file found, starting fresh")
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeLastTx(tx string) {
	err := os.WriteFile(lastTxFile, []byte(tx), 0644)
	if err != nil {
		log.Printf("Error writing last tx file: %v\n", err)
	}
}

func sendEmail(subject, body string) error {
	server := mail.NewSMTPClient()
	server.Host = "smtp.gmail.com"
	server.Port = 587
	server.Username = os.Getenv("SMTP_USER")
	server.Password = os.Getenv("SMTP_PASS")
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		return fmt.Errorf("SMTP connect error: %w", err)
	}

	email := mail.NewMSG()
	email.SetFrom("Solana Notifier <"+server.Username+">").
		AddTo(os.Getenv("EMAIL_TO")).
		SetSubject(subject).
		SetBody(mail.TextPlain, body)

	err = email.Send(smtpClient)
	if err != nil {
		return fmt.Errorf("SMTP send error: %w", err)
	}
	return nil
}

func checkTransactions() {
	log.Println("Checking new transactions...")

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getSignaturesForAddress",
		"params":  []interface{}{walletAddress, map[string]int{"limit": 1}},
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(solanaRPC, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error making Solana request: %v\n", err)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var result solanaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding Solana response: %v\n", err)
		return
	}

	if len(result.Result) == 0 {
		log.Println("No transactions found")
		return
	}

	latest := result.Result[0]
	if latest.Signature != lastTxSig {
		url := "https://solscan.io/tx/" + latest.Signature
		log.Printf("New transaction found: %s\n", latest.Signature)

		err := sendEmail("📥 New Solana Transaction!", fmt.Sprintf("New transaction on wallet %s\n\nDetails: %s", walletAddress, url))
		if err != nil {
			log.Printf("Error sending email: %v\n", err)
		} else {
			log.Println("Email sent successfully.")
			lastTxSig = latest.Signature
			writeLastTx(lastTxSig)
		}
	} else {
		log.Println("No new transaction since last check.")
	}
}

func main() {
	log.Println("Starting Solana wallet notifier...")

	if walletAddress == "" {
		log.Fatal("WALLET_ADDRESS env variable not set")
	}
	if os.Getenv("SMTP_USER") == "" || os.Getenv("SMTP_PASS") == "" || os.Getenv("EMAIL_TO") == "" {
		log.Fatal("SMTP_USER, SMTP_PASS, and EMAIL_TO env variables must be set")
	}

	lastTxSig = readLastTx()
	log.Printf("Loaded last transaction from file: %s\n", lastTxSig)

	for {
		checkTransactions()
		time.Sleep(pollInterval)
	}
}
