package main

import (
	"fmt"
	"github.com/yobert/alsa"
	"os"
)

func main() {
	cards, err := alsa.OpenCards()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, card := range cards {
		devices, err := card.Devices()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Card:", card)
		for _, device := range devices {
			fmt.Println("Device:", device.Title, device.Path, device.Type, device.Play, device.Record)
		}
	}

	alsa.CloseCards(cards)
}
