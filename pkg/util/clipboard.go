package util

import "github.com/go-vgo/robotgo/clipboard"

func GetTextFromClipboard() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	return text, nil
}

func WriteTextToClipboard(t string) error {
	err := clipboard.WriteAll(t)
	return err
}
