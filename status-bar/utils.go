package main

import (
	"encoding/json"
	"fmt"
)

type borderThickness struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

type color int

func colorToString(c color) string {
	return fmt.Sprintf("#%02d%02d%02d", c&0xFF, (c>>8)&0xFF, (c>>16)&0xFF)
}

type swaybarMessageBodyBlock struct {
	FullText            string
	ShortText           string
	shouldUseColor      byte // bits represent foreground, background and border
	ForegroundColor     color
	BackgroundColor     color
	BorderColor         color
	BorderThickness     borderThickness
	MinWidth            int // or string whose length represents the desired width
	Align               string
	Name                string
	Instance            string
	Urgent              bool
	Separator           bool
	SeparatorBlockWidth int
	Markup              string
}

// Test function
func sendToSwaybar(body swaybarMessageBody) {
	fullBodyArray := make([]fullSwaybarMessageBodyBlock, len(body))
	for i, y := range body {
		var bodyBlock fullSwaybarMessageBodyBlock

		// FullText is the only field that is required. Not providing it invalidates the whole block
		bodyBlock.FullText = y.FullText
		if y.ShortText != "" {
			bodyBlock.ShortText = y.ShortText
		}
		if (y.shouldUseColor & 0x1) != 0 {
			bodyBlock.Color = colorToString(y.ForegroundColor)
		}
		if (y.shouldUseColor & 0x2) != 0 {
			bodyBlock.Background = colorToString(y.BackgroundColor)
		}
		if (y.shouldUseColor & 0x4) != 0 {
			bodyBlock.Border = colorToString(y.BorderColor)
		}
		if y.BorderThickness.Top != 0 {
			bodyBlock.BorderTop = &y.BorderThickness.Top
		}
		if y.BorderThickness.Bottom != 0 {
			bodyBlock.BorderBottom = &y.BorderThickness.Bottom
		}
		if y.BorderThickness.Left != 0 {
			bodyBlock.BorderLeft = &y.BorderThickness.Left
		}
		if y.BorderThickness.Right != 0 {
			bodyBlock.BorderRight = &y.BorderThickness.Right
		}
		if y.MinWidth != 0 {
			bodyBlock.MinWidth = &y.MinWidth
		}
		if y.Align != "" {
			bodyBlock.Align = y.Align
		}
		if y.Name != "" {
			bodyBlock.Name = y.Name
			// Only need instance if you have a name
			if y.Instance != "" {
				bodyBlock.Instance = y.Instance
			}
		}
		if y.Urgent != false {
			bodyBlock.Urgent = &y.Urgent
		}
		if y.Separator != false {
			bodyBlock.Separator = &y.Separator
		}
		if y.SeparatorBlockWidth != 0 {
			bodyBlock.SeparatorBlockWidth = &y.SeparatorBlockWidth
		}
		if y.Markup != "" {
			bodyBlock.Markup = y.Markup
		}
		fullBodyArray[i] = bodyBlock
	}

	bytes, err := json.Marshal(fullBodyArray)
	if err != nil {
		panic(err)
	}

	str := string(bytes)
	fmt.Println(str, ",")
}
