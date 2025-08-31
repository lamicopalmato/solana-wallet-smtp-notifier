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
)

var (
	solanaRPC   = os.Getenv("SOLANA_RPC")
	walletAddrs []string
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

func readLastTx(wallet string) string {
	fileName := fmt.Sprintf("/opt/solana-notifier/last_tx_%s.sig", wallet)
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Printf("No last tx file found for wallet %s, starting fresh", wallet)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeLastTx(wallet string, tx string) {
	fileName := fmt.Sprintf("/opt/solana-notifier/last_tx_%s.sig", wallet)
	err := os.WriteFile(fileName, []byte(tx), 0644)
	if err != nil {
		log.Printf("Error writing last tx file for wallet %s: %v", wallet, err)
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

func checkTransactions(wallet string, lastTxSig string) (string, bool) {
	log.Printf("Checking new transactions for wallet %s...", wallet)

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getSignaturesForAddress",
		"params":  []interface{}{wallet, map[string]int{"limit": 1}},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(solanaRPC, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error making Solana request for %s: %v", wallet, err)
		return lastTxSig, false
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	var result solanaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding Solana response for %s: %v", wallet, err)
		return lastTxSig, false
	}

	if len(result.Result) == 0 {
		log.Printf("No transactions found for wallet %s", wallet)
		return lastTxSig, false
	}

	latest := result.Result[0]
	if latest.Signature != lastTxSig {
		url := "https://solscan.io/tx/" + latest.Signature
		log.Printf("New transaction found for %s: %s", wallet, latest.Signature)

		err := sendEmail(
			"📥 New Solana Transaction!",
			fmt.Sprintf("New transaction on wallet %s\n\nDetails: %s", wallet, url),
		)

		if err != nil {
			log.Printf("Error sending email: %v", err)
			return lastTxSig, false
		}

		log.Printf("Email sent successfully for wallet %s", wallet)
		return latest.Signature, true
	}

	log.Printf("No new transaction for wallet %s since last check.", wallet)
	return lastTxSig, false
}

func main() {
	log.Println("Starting Solana wallet notifier...")

	walletsEnv := os.Getenv("WALLET_ADDRESS")
	if walletsEnv == "" {
		log.Fatal("WALLET_ADDRESS env variable not set")
	}

	// Split wallet addresses by comma and trim whitespace
	walletAddrs = strings.Split(walletsEnv, ",")
	for i, wallet := range walletAddrs {
		walletAddrs[i] = strings.TrimSpace(wallet)
	}

	log.Printf("Monitoring %d wallet addresses", len(walletAddrs))

	if os.Getenv("SMTP_USER") == "" || os.Getenv("SMTP_PASS") == "" || os.Getenv("EMAIL_TO") == "" {
		log.Fatal("SMTP_USER, SMTP_PASS, and EMAIL_TO env variables must be set")
	}

	// Load last transaction for each wallet
	lastTxSigs := make(map[string]string)
	for _, wallet := range walletAddrs {
		lastTxSigs[wallet] = readLastTx(wallet)
		log.Printf("Loaded last transaction for wallet %s: %s", wallet, lastTxSigs[wallet])
	}

	// Main monitoring loop
	for {
		for _, wallet := range walletAddrs {
			lastSig, updated := checkTransactions(wallet, lastTxSigs[wallet])
			if updated {
				lastTxSigs[wallet] = lastSig
				writeLastTx(wallet, lastSig)
			}
		}
		time.Sleep(pollInterval)
	}
}
