package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type swaybarMessageHeader struct {
	Version     int       `json:"version"`
	ClickEvents bool      `json:"click_events"`
	ContSignal  os.Signal `json:"cont_signal"`
	StopSignal  os.Signal `json:"stop_signal"`
}

/*
   ┌──────────────────────┬───────────────────┬────────────────────────────────────────────────────┐
   │      PROPERTY        │     DATA TYPE     │                    DESCRIPTION                     │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │      full_text       │      string       │ The text that will be displayed. If missing, the   │
   │                      │                   │ block will be skipped.                             │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │     short_text       │      string       │ If given and the text needs to be shortened due to │
   │                      │                   │ space, this will be displayed instead of full_text │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │        color         │      string       │ The text color to use in #RRGGBBAA or #RRGGBB no‐  │
   │                      │                   │ tation                                             │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │     background       │      string       │ The background color for the block in #RRGGBBAA or │
   │                      │                   │ #RRGGBB notation                                   │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │       border         │      string       │ The border color for the block in #RRGGBBAA or     │
   │                      │                   │ #RRGGBB notation                                   │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │     border_top       │      integer      │ The height in pixels of the top border. The de‐    │
   │                      │                   │ fault is 1                                         │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │    border_bottom     │      integer      │ The height in pixels of the bottom border. The de‐ │
   │                      │                   │ fault is 1                                         │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │     border_left      │      integer      │ The width in pixels of the left border. The de‐    │
   │                      │                   │ fault is 1                                         │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │    border_right      │      integer      │ The width in pixels of the right border. The de‐   │
   │                      │                   │ fault is 1                                         │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │      min_width       │ integer or string │ The minimum width to use for the block. This can   │
   │                      │                   │ either be given in pixels or a string can be given │
   │                      │                   │ to allow for it to be calculated based on the      │
   │                      │                   │ width of the string.                               │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │        align         │      string       │ If the text does not span the full width of the    │
   │                      │                   │ block, this specifies how the text should be       │
   │                      │                   │ aligned inside of the block. This can be left (de‐ │
   │                      │                   │ fault), right, or center.                          │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │        name          │      string       │ A name for the block. This is only used to iden‐   │
   │                      │                   │ tify the block for click events. If set, each      │
   │                      │                   │ block should have a unique name and instance pair. │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │      instance        │      string       │ The instance of the name for the block. This is    │
   │                      │                   │ only used to identify the block for click events.  │
   │                      │                   │ If set, each block should have a unique name and   │
   │                      │                   │ instance pair.                                     │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │       urgent         │      boolean      │ Whether the block should be displayed as urgent.   │
   │                      │                   │ Currently swaybar utilizes the colors set in the   │
   │                      │                   │ sway config for urgent workspace buttons. See      │
   │                      │                   │ sway-bar(5) for more information on bar color con‐ │
   │                      │                   │ figuration.                                        │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │      separator       │      boolean      │ Whether the bar separator should be drawn after    │
   │                      │                   │ the block. See sway-bar(5) for more information on │
   │                      │                   │ how to set the separator text.                     │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │separator_block_width │      integer      │ The amount of pixels to leave blank after the      │
   │                      │                   │ block. The separator text will be displayed cen‐   │
   │                      │                   │ tered in this gap. The default is 9 pixels.        │
   ├──────────────────────┼───────────────────┼────────────────────────────────────────────────────┤
   │       markup         │      string       │ The type of markup to use when parsing the text    │
   │                      │                   │ for the block. This can either be pango or none    │
   │                      │                   │ (default).                                         │
   └──────────────────────┴───────────────────┴────────────────────────────────────────────────────┘
*/

type color int

func colorToString(c color) string {
	return fmt.Sprintf("#%02d%02d%02d", c&0xFF, (c>>8)&0xFF, (c>>16)&0xFF)
}

type borderThickness struct {
	Top    int
	Bottom int
	Left   int
	Right  int
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

type fullSwaybarMessageBodyBlock struct {
	FullText     string `json:"full_text"`
	ShortText    string `json:"short_text,omitempty"`
	Color        string `json:"color,omitempty"`
	Background   string `json:"background,omitempty"`
	Border       string `json:"border,omitempty"`
	BorderTop    *int   `json:"border_top,omitempty"`
	BorderBottom *int   `json:"border_bottom,omitempty"`
	BorderLeft   *int   `json:"border_left,omitempty"`
	BorderRight  *int   `json:"border_right,omitempty"`
	MinWidth     *int   `json:"min_width,omitempty"` // or string whose length represents the desired width
	Align        string `json:"align,omitempty"`

	// blocks that respond to clicks should have a unique Name-Instance pair. Only name is needed to respond to clicks
	Name     string `json:"name,omitempty"`     // needed to receive click events
	Instance string `json:"instance,omitempty"` // Also identifies a clicker

	Urgent              *bool  `json:"urgent,omitempty"`
	Separator           *bool  `json:"separator,omitempty"`
	SeparatorBlockWidth *int   `json:"separator_block_width,omitempty"`
	Markup              string `json:"markup,omitempty"`
}

type swaybarMessageBody []swaybarMessageBodyBlock

func sendHeader(header swaybarMessageHeader) {
	bytes, err := json.Marshal(header)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bytes))
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

type monitorChan chan<- bool
type blockProvider interface {
	monitor(changeChan monitorChan)
	createBlock() fullSwaybarMessageBodyBlock
	name() string // if this is non-empty, then it will receive click events
	respondToClick(event clickEvent)
}

type weatherProvider struct {
	weatherStatus string
}

func (w *weatherProvider) monitor(changeChan monitorChan) {
	logger.Println("Weather")
	request, err := http.NewRequest("GET", "https://wttr.in?0&T&Q", nil)
	if err != nil {
		logger.Println("Cannot create request", err)
	}
	request.Header["User-Agent"] = []string{"curl/8.0.1"}

	logger.Println(request.Header)
	client := http.Client{}

	for {
		{ // This block is so that the goto doesn't complain about jumping over a variable declaration
			// response, err := http.Get("https://wttr.in?0&T&Q")
			response, _ := client.Do(request)

			status, err := strconv.ParseInt(response.Status[:3], 10, 32)
			if err != nil {
				logger.Println("Int parsing error", err)
				goto threadSleep
			}

			if status >= 200 && status < 300 {
				responseBodyBytes, err := io.ReadAll(response.Body)
				if err != nil {
					logger.Println("Error reading response body")
					goto threadSleep
				}
				responseBody := string(responseBodyBytes)
				logger.Println(responseBody)

				lines := strings.SplitN(responseBody, "\n", 3)
				firstValidCharacterIndex := 16
				line1 := strings.Trim(lines[0][firstValidCharacterIndex:], " \n\t")
				line2 := strings.Trim(lines[1][firstValidCharacterIndex:], " \n\t")
				w.weatherStatus = fmt.Sprintf("%s %s", line1, line2)
			} else {
				w.weatherStatus = fmt.Sprintf("wttr.in status code %d", status)
			}

			changeChan <- true
		}

	threadSleep:
		time.Sleep(1 * time.Hour)
	}
}

func (w *weatherProvider) createBlock() fullSwaybarMessageBodyBlock {
	var block fullSwaybarMessageBodyBlock

	block.FullText = w.weatherStatus

	return block
}

func (weatherProvider) name() string {
	return ""
}

func (weatherProvider) respondToClick(event clickEvent) {
}

// ---

type ipAddressProvider struct {
	text string
}

func (ip ipAddressProvider) monitor(changeChan monitorChan) {
	// This does not need to infinite-loop
}

func (ip *ipAddressProvider) createBlock() fullSwaybarMessageBodyBlock {
	var block fullSwaybarMessageBodyBlock

	if ip.text == "" {
		hostnameOutput, err := exec.Command("hostname", "-I").Output()
		if err != nil {
			return block
		}

		localIPAddress := strings.SplitN(string(hostnameOutput), " ", 2)[0]
		ip.text = fmt.Sprintf("IP:%s", localIPAddress)
		logger.Println("Set text to", ip.text)
	}

	block.FullText = ip.text
	logger.Println("Text is", ip.text)

	return block
}

func (ipAddressProvider) name() string {
	return ""
}

func (ipAddressProvider) respondToClick(clickEvent) {}

// ---

type timeMonitor struct{}

func (timeMonitor) monitor(changeChan monitorChan) {
	for {
		t := time.Now()
		diff := 60 - t.Second()
		time.Sleep(time.Duration(diff) * time.Second)
		changeChan <- true
	}
}

func (timeMonitor) createBlock() fullSwaybarMessageBodyBlock {
	block := fullSwaybarMessageBodyBlock{}
	t := time.Now()
	block.FullText = fmt.Sprintf("%s %s %02d, %d %02d:%02d", t.Weekday().String()[:3], t.Month().String()[:3], t.Day(), t.Year(), t.Hour(), t.Minute())
	return block
}

func (timeMonitor) name() string {
	return "" // Does not respond to clicks
}

func (timeMonitor) respondToClick(event clickEvent) {}

// ---

type notificationCenterState int

const (
	ncStateNone notificationCenterState = iota
	ncStateNotification
	ncStateDndNone
	ncStateDndNotification
)

func ncGetState(str string) notificationCenterState {
	// swaync-client -swb | while read -r line; do echo $line | jq '.class' | 's/none/ /p; s/notification/ ! /p; s/dnd-notification/ ! /p; s/dnd-none/ /p'
	switch str {
	case "none":
		return ncStateNone
	case "notification":
		return ncStateNotification
	case "dnd-notification":
		return ncStateDndNotification
	case "dnd-none":
		return ncStateDndNone
	default:
		return ncStateNone
	}
}

type notificationCenterMonitor struct {
	state  notificationCenterState
	isOpen bool
}

func (nc *notificationCenterMonitor) name() string {
	return "notification center"
}

func (nc *notificationCenterMonitor) respondToClick(event clickEvent) {
	// logger.Println("NC Received click", event)
	if event.Button == 1 {
		exec.Command("swaync-client", "-t", "-sw").Run()
	}
}

type ncClientOutput struct {
	Class any `json:"class"`
}

func (nc *notificationCenterMonitor) monitor(changeChan monitorChan) {
	ncMonitor := exec.Command("swaync-client", "-swb")
	stdout, err := ncMonitor.StdoutPipe()
	if err != nil {
		panic(err)
	}
	jsonDecoder := json.NewDecoder(stdout)
	ncMonitor.Start()

	for {
		var ncStateOutput ncClientOutput
		err = jsonDecoder.Decode(&ncStateOutput)
		if err != nil {
			panic(err)
		}

		oldState := nc.state
		nc.isOpen = false
		if str, ok := ncStateOutput.Class.(string); ok {
			nc.state = ncGetState(str)
		} else if arr, ok := ncStateOutput.Class.([]any); ok {
			nc.state = ncGetState(arr[0].(string))
			if len(arr) > 1 && arr[1].(string) == "cc-open" {
				nc.isOpen = true
			}
		}

		// logger.Printf("Got class %g (T = %T) | Changed state to %v | isOpen to %t", ncStateOutput.Class, ncStateOutput.Class, nc.state, nc.isOpen)
		// I don't think there's a reason to change the icon if the notification center is open
		if oldState != nc.state {
			changeChan <- true
		}
	}
}

func (nc *notificationCenterMonitor) createBlock() fullSwaybarMessageBodyBlock {
	var result fullSwaybarMessageBodyBlock

	if nc.state == ncStateNone {
		result.FullText = ""
	} else if nc.state == ncStateNotification {
		result.FullText = " !"
	} else if nc.state == ncStateDndNone {
		result.FullText = ""
	} else if nc.state == ncStateDndNotification {
		result.FullText = " !"
	}

	// if nc.isOpen {
	// 	result.FullText = "o " + result.FullText
	// }

	return result
}

/*
	┌───────────┬───────────┬────────────────────────────────────────────────────┐
	│ PROPERTY  │ DATA TYPE │                    DESCRIPTION                     │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│   name    │  string   │ The name of the block, if set                      │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│ instance  │  string   │ The instance of the block, if set                  │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│    x      │  integer  │ The x location that the click occurred at          │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│    y      │  integer  │ The y location that the click occurred at          │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│  button   │  integer  │ The x11 button number for the click. If the button │
	│           │           │ does not have an x11 button mapping, this will be  │
	│           │           │ 0.                                                 │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│  event    │  integer  │ The event code that corresponds to the button for  │
	│           │           │ the click                                          │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│relative_x │  integer  │ The x location of the click relative to the top-   │
	│           │           │ left of the block                                  │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│relative_y │  integer  │ The y location of the click relative to the top-   │
	│           │           │ left of the block                                  │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│  width    │  integer  │ The width of the block in pixels                   │
	├───────────┼───────────┼────────────────────────────────────────────────────┤
	│  height   │  integer  │ The height of the block in pixels                  │
	└───────────┴───────────┴────────────────────────────────────────────────────┘
*/

type clickEvent struct {
	Name      string `json:"name"`
	Instance  string `json:"instance"` // I don't currently set this
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Button    int    `json:"button"`
	Event     int    `json:"event"`
	RelativeX int    `json:"relative_x"`
	RelativeY int    `json:"relative_y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

func decodeClickEvent(eventString string) clickEvent {
	var result clickEvent

	if eventString[0] == ',' {
		eventString = eventString[1:]
	}

	err := json.Unmarshal([]byte(eventString), &result)
	if err != nil {
		panic(err)
	}

	return result
}

func updateFullBlockValues(fullBlockValues []fullSwaybarMessageBodyBlock, blockProviders []blockProvider) {
	for i, provider := range blockProviders {
		fullBlock := provider.createBlock()

		// Set name here to make sure that it responds to clicks if it needs to
		fullBlock.Name = provider.name()
		fullBlockValues[i] = fullBlock
	}
}

var logger *log.Logger

func main() {
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}

	directory := filepath.Dir(path)
	logsPath := filepath.Join(directory, "logs.txt")
	logsFile, err := os.OpenFile(logsPath, os.O_RDWR|os.O_CREATE, 0644)
	defer logsFile.Close()
	logsFile.Truncate(0)

	if err != nil {
		panic(err)
	}

	logger = log.New(logsFile, "", 0)

	defaultHeader := swaybarMessageHeader{
		Version:     1,
		ClickEvents: true,
		ContSignal:  syscall.SIGCONT,
		StopSignal:  syscall.SIGSTOP,
	}

	stdinChannel := make(chan clickEvent, 1)
	stdinNeverWriteToMe := make(chan clickEvent) // This channel is never written to and so it always blocks. This is in case stdinChannel is closed
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			buffer, err := reader.ReadString('\n')
			if err != nil { // Maybe log non io.EOF errors, if you want
				close(stdinChannel)
				return
			}

			trimmed := strings.Trim(buffer, " \n")
			if trimmed == "[" {
				// skip first [
			} else if trimmed == "]" {
				close(stdinChannel)
				return
			} else {
				stdinChannel <- decodeClickEvent(trimmed)
			}
		}
	}()

	weather := weatherProvider{}
	ipProvider := ipAddressProvider{}
	timeProvider := timeMonitor{}
	ncProvider := notificationCenterMonitor{}

	blockProviders := []blockProvider{
		// TODO
		// volume
		&weather,
		// ip address
		&ipProvider,
		// temperature
		// battery
		// Bluetooth
		// Wifi
		timeProvider,
		&ncProvider,
	}

	providersByName := make(map[string]int)
	for i, block := range blockProviders {
		name := block.name()
		if name != "" {
			providersByName[name] = i
		}
	}

	fullBlockValues := make([]fullSwaybarMessageBodyBlock, len(blockProviders))
	blockChanged := make(chan bool)

	// Update swaybar with initial info so you don't have to wait until a block updates
	go func() {
		blockChanged <- true
	}()

	for _, block := range blockProviders {
		go block.monitor(blockChanged)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCONT, syscall.SIGSTOP)

	sendHeader(defaultHeader)
	fmt.Print("[")

mainLoop:
	for {
		select {
		case event, isOpen := <-stdinChannel:
			if isOpen {
				providerIndex := providersByName[event.Name]
				blockProviders[providerIndex].respondToClick(event)
			} else {
				stdinChannel = stdinNeverWriteToMe
			}

		case signal := <-signals:
			if signal == syscall.SIGCONT {
				logger.Println("SIGCONT")
			} else if signal == syscall.SIGSTOP {
				logger.Println("SIGSTOP")
				break mainLoop
			}

		case <-blockChanged:
			updateFullBlockValues(fullBlockValues, blockProviders)
			bytes, err := json.Marshal(fullBlockValues)
			if err != nil {
				panic(err)
			}
			str := string(bytes)
			logger.Println("Data", str)
			fmt.Println(str, ",")
		}
	}

}
