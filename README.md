# Solana Wallet Notifier

Questo programma monitora le transazioni in entrata (o in uscita) su un wallet Solana specifico e invia un'email quando viene rilevata una nuova transazione.

## 🔧 Requisiti

- Go installato sulla macchina (`go 1.18+`)
- Variabili d'ambiente configurate:
    - `SOLANA_RPC`: indirizzo rpc Solana
    - `WALLET_ADDRESS`: indirizzo del wallet Solana da monitorare
    - `SMTP_USER`: email Gmail usata per inviare le notifiche
    - `SMTP_PASS`: password o App Password della Gmail
    - `EMAIL_TO`: email destinataria delle notifiche

## 📂 File importanti

- `main.go`: codice sorgente del notifier
- `/opt/solana-notifier/last_tx.sig`: file persistente che memorizza l'ultima transazione per evitare notifiche duplicate

## 🛠️ Installazione

1. Clona o copia il file `main.go` nella tua macchina.
2. Crea la directory necessaria:
   ```bash
   sudo mkdir -p /opt/solana-notifier
   ```

## 🚀 Esecuzione

### 🔹 Esecuzione diretta

Esporta le variabili d’ambiente:
```bash
export SOLANA_RPC=https://api.mainnet-beta.solana.com
export WALLET_ADDRESS=tuo_wallet
export SMTP_USER=tuo@email.com
export SMTP_PASS=la_tua_password_app
export EMAIL_TO=email_destinatario
```

E poi esegui:
```bash
go run main.go
```

### 🔹 Esecuzione compilata

Puoi anche compilare l'eseguibile e avviarlo così:

```bash
go build -o solana-notifier main.go
```

Poi lancia il binario:
```bash
WALLET_ADDRESS=tuo_wallet \
SMTP_USER=tuo@email.com \
SMTP_PASS=la_tua_password_app \
EMAIL_TO=email_destinatario \
./solana-notifier
```

Puoi anche creare un semplice `systemd` service per avviarlo automaticamente.

## 🔁 Funzionamento

- Ogni 30 secondi lo script interroga l'RPC di Solana per controllare l'ultima transazione.
- Se la transazione è nuova rispetto all'ultima salvata in `last_tx.sig`, viene inviata una mail e il file viene aggiornato.

## 📬 Email inviata

- Oggetto: `📥 New Solana Transaction!`
- Corpo: contiene l'indirizzo monitorato e un link alla transazione su SolScan.

## 🔒 Sicurezza

Se si utilizza Gmail, si consiglia di usare una **Google App Password** per `SMTP_PASS` e non la password diretta dell’account Gmail.

## 📄 Licenza

Script fornito "as is", senza garanzia. Usare a proprio rischio.

---
