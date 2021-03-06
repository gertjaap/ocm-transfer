package main

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/DusanKasan/parsemail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/gorilla/mux"
	"github.com/mhale/smtpd"
)

var telegramWaiters sync.Map
var mailWaiters sync.Map

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/t/{from}", telegramHandler)
	r.HandleFunc("/m/{addr}", mailHandler)
	srv := &http.Server{
		Handler: r,
		Addr:    ":8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 65 * time.Second,
		ReadTimeout:  65 * time.Second,
	}

	go telegramLoop()
	go mailLoop()

	srv.ListenAndServe()
}

func telegramLoop() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_API_KEY"))
	if err != nil {
		log.Panic(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Copied to OCM!")
		msg.ReplyToMessageID = update.Message.MessageID

		if update.Message.Text == "/start" {
			msg.Text = "Send me an LN invoice to fill into the send screen in OCM"
		} else {
			c, ok := telegramWaiters.Load(strings.ToLower(update.Message.From.UserName))
			if ok {
				ch, ok := c.(chan string)
				if ok {

					ch <- extractFirstLNInvoice(update.Message.Text)
				}
				telegramWaiters.Delete(strings.ToLower(update.Message.From.UserName))
			}
		}

		bot.Send(msg)
	}
}

func extractFirstLNInvoice(t string) string {
	start := strings.Index(t, "lnbc1")
	if start > -1 {
		t = t[start:]
	}
	end := strings.IndexAny(t, " \r\n\t<")
	if end > -1 {
		t = t[:end]
	}
	return t
}

func smtpHandler(origin net.Addr, from string, to []string, data []byte) error {
	email, err := parsemail.Parse(bytes.NewReader(data)) // returns Email struct and error
	if err != nil {
		// handle error
	}

	t := to[0]
	t = strings.ToLower(t)
	t = strings.ReplaceAll(t, "@ocm-backend.blkidx.org", "")
	c, ok := mailWaiters.Load(t)
	if ok {
		ch, ok := c.(chan string)
		if ok {
			lninv := extractFirstLNInvoice(email.Subject)
			lninv2 := extractFirstLNInvoice(email.TextBody)
			if len(lninv2) > len(lninv) {
				lninv = lninv2
			}
			if lninv == "" {
				lninv2 = extractFirstLNInvoice(email.HTMLBody)
				if len(lninv2) > len(lninv) {
					lninv = lninv2
				}
			}
			ch <- lninv
			mailWaiters.Delete(t)
		}
	}
	return nil
}

func mailLoop() {
	smtpd.ListenAndServe(":2525", smtpHandler, "OCM Tranfer", "")
}

func telegramHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c := make(chan string, 1)
	telegramWaiters.Store(strings.ToLower(vars["from"]), c)

	value := ""
	select {
	case value = <-c:
	case <-time.After(1 * time.Minute):
	}

	w.Write([]byte(value))
}

func mailHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	c := make(chan string, 1)
	mailWaiters.Store(strings.ToLower(vars["addr"]), c)

	value := ""
	select {
	case value = <-c:
	case <-time.After(1 * time.Minute):
	}

	w.Write([]byte(value))
}
