package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func handleCipherCommand(request string) string {
	url := "http://localhost:8080/cipher"

	type CipherRequest struct {
		Message string   `json:"message"`
		Deck    []string `json:"deck"`
	}

	data := CipherRequest{request, []string{}}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Println("Ошибка!", err)
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return string(body)
}

func handleDecipherCommand(message string, deck string) string {
	url := "http://localhost:8080/decipher"

	var deckArr []string
	json.Unmarshal([]byte(deck), &deckArr)

	type DecipherRequest struct {
		Message string   `json:"message"`
		Deck    []string `json:"deck"`
	}

	data := DecipherRequest{message, deckArr}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Println("Ошибка!", err)
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return string(body)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env file couldn't be loaded")
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TG_TOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)

	var state = "started"

	var textForDecipher string
	// var textForCipher string

	for update := range updates {
		// Если обновление не содержит сообщение, пропускаем его
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "cipher" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter message for encrypt!")
			// Отправляем сообщение обратно пользователю
			bot.Send(msg)
			state = "waiting_for_enter_for_cipher"
		}

		if update.Message.IsCommand() && update.Message.Command() == "decipher" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter message for decrypt!")
			// Отправляем сообщение обратно пользователю
			bot.Send(msg)
			state = "waiting_for_enter_for_decipher"
		}

		if !update.Message.IsCommand() && update.Message.Text != "" {
			if state == "started" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter command cipher or decipher!")
				bot.Send(msg)
			}
			if state == "waiting_for_enter_for_cipher" {
				ciphered_text := handleCipherCommand(update.Message.Text)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, ciphered_text)
				bot.Send(msg)
				state = "started"
			} else if state == "waiting_for_enter_for_decipher" {
				textForDecipher = update.Message.Text
				state = "waiting_for_deck_to_decipher"
				// deciphered_text := handleDecipherCommand(update.Message.Text)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Enter deck for deciphering")
				bot.Send(msg)
				// state = "started"
			} else if state == "waiting_for_deck_to_decipher" {
				deciphered_text := handleDecipherCommand(textForDecipher, update.Message.Text)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, deciphered_text)
				bot.Send(msg)
				state = "started"
			}
		}
	}
}
