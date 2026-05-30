package notify

import (
	"fmt"
	"io"
	"strings"
)

// SendTelegramMessage sends an HTML-formatted message to a Telegram chat via Bot API.
func SendTelegramMessage(botToken, chatId, text string) error {
	if botToken == "" || chatId == "" {
		return fmt.Errorf("telegram: botToken and chatId are required")
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := fmt.Sprintf(`{"chat_id":%q,"text":%q,"parse_mode":"HTML"}`, chatId, text)
	resp, err := httpClient.Post(apiURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
