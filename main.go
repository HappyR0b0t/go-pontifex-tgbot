package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	userStates        = make(map[int64]string)
	userStatesMu      sync.Mutex
	textForDecipher   = make(map[int64]string)
	textForDecipherMu sync.Mutex

	// Menu texts
	firstMenu = "<b>Main menu</b>\n\nSelect an option"
	// secondMenu = "<b>Cipher a text message menu</b>\n\nSelect an option"
	// thirdMenu  = "<b>Decipher a text message menu</b>\n\nSelect an option"
	// fourthMenu = "<b>About</b>\n\nAbout Pontifex algorithm..."

	// Button texts
	aboutButton        = "About"
	backButton         = "Back"
	cipherButton       = "Cipher text"
	decipherButton     = "Decipher text"
	generatedeckButton = "Generate deck"

	// Keyboard layout for the first menu. One button, one row
	firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(aboutButton, aboutButton),
			tgbotapi.NewInlineKeyboardButtonData(cipherButton, cipherButton),
			tgbotapi.NewInlineKeyboardButtonData(decipherButton, decipherButton),
			tgbotapi.NewInlineKeyboardButtonData(generatedeckButton, generatedeckButton),
		),
	)
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

	// fmt.Println(body)

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

func handleGenerateCommand() string {
	url := "http://localhost:8080/generate"
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Ошибка!", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Ошибка при чтении ответа:", err)
		return ""
	}
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

	for update := range updates {
		// Если обновление не содержит сообщение, пропускаем его
		if update.Message == nil {
			continue
		}
		go handleMessage(bot, update.Message)
	}
}

func stateChanger(chatID int64, state string) {
	userStatesMu.Lock()
	userStates[chatID] = state
	userStatesMu.Unlock()
}

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	userStatesMu.Lock()
	state, ok := userStates[chatID]
	if !ok {
		state = "started"
		userStates[chatID] = state
	}
	userStatesMu.Unlock()

	if msg.IsCommand() {
		switch msg.Command() {
		case "cipher":
			reply := tgbotapi.NewMessage(chatID, "Enter message to encrypt!")
			bot.Send(reply)
			stateChanger(chatID, "waiting_for_text_to_cipher")
			return

		case "decipher":
			reply := tgbotapi.NewMessage(chatID, "Enter message to decrypt!")
			bot.Send(reply)
			stateChanger(chatID, "waiting_for_text_to_decipher")
			return

		case "generate":
			// reply := tgbotapi.NewMessage(chatID, "Enter message to decrypt!")
			generated := handleGenerateCommand()
			reply := tgbotapi.NewMessage(chatID, generated)
			bot.Send(reply)

			return

		case "start":
			reply := tgbotapi.NewMessage(chatID, "Use /cipher or /decipher to start.")
			bot.Send(reply)
			stateChanger(chatID, "started")
			return
		}
	}

	if !msg.IsCommand() && msg.Text != "" {
		switch state {
		case "started":
			reply := tgbotapi.NewMessage(chatID, "Please use /cipher or /decipher first.")
			bot.Send(reply)

		case "waiting_for_text_to_cipher":
			ciphered := handleCipherCommand(msg.Text)

			reply := tgbotapi.NewMessage(chatID, ciphered)
			bot.Send(reply)

			stateChanger(chatID, "started")

		case "waiting_for_text_to_decipher":
			textForDecipherMu.Lock()
			textForDecipher[chatID] = msg.Text
			textForDecipherMu.Unlock()

			reply := tgbotapi.NewMessage(chatID, "Now send the deck to use for deciphering:")
			bot.Send(reply)

			stateChanger(chatID, "waiting_for_deck_to_decipher")

		case "waiting_for_deck_to_decipher":
			textForDecipherMu.Lock()
			originalText := textForDecipher[chatID]
			textForDecipherMu.Unlock()

			deciphered := handleDecipherCommand(originalText, msg.Text)
			reply := tgbotapi.NewMessage(chatID, deciphered)
			bot.Send(reply)

			stateChanger(chatID, "started")

		}
	}
}
