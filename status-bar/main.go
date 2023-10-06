package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	// "time"
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

type swaybarMessageBodyBlock struct {
	FullText            string
	ShortText           string
	Color               string
	Background          string
	Border              string
	BorderTop           int
	BorderBottom        int
	BorderLeft          int
	BorderRight         int
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

		bodyBlock.FullText = y.FullText
		if y.ShortText != "" {
			bodyBlock.ShortText = y.ShortText
		}
		if y.Color != "" {
			bodyBlock.Color = y.Color
		}
		if y.Background != "" {
			bodyBlock.Background = y.Background
		}
		if y.Border != "" {
			bodyBlock.Border = y.Border
		}
		if y.BorderTop != 0 {
			bodyBlock.BorderTop = &y.BorderTop
		}
		if y.BorderBottom != 0 {
			bodyBlock.BorderBottom = &y.BorderBottom
		}
		if y.BorderLeft != 0 {
			bodyBlock.BorderLeft = &y.BorderLeft
		}
		if y.BorderRight != 0 {
			bodyBlock.BorderRight = &y.BorderRight
		}
		if y.MinWidth != 0 {
			bodyBlock.MinWidth = &y.MinWidth
		}
		if y.Align != "" {
			bodyBlock.Align = y.Align
		}
		if y.Name != "" {
			bodyBlock.Name = y.Name
		}
		if y.Instance != "" {
			bodyBlock.Instance = y.Instance
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

type monitorChan chan bool
type blockProvider interface {
	monitor(changeChan monitorChan)
	createBlock() fullSwaybarMessageBodyBlock
	name() string // if this is non-empty, then it will receive click events
	respondToClick(event clickEvent)
}

type timeMonitor struct {
}

func (timeMonitor) monitor(changeChan monitorChan) {
	// TODO: create timer that will fire on the minute

	for {
		t := time.Now()
		diff := 60 - t.Second()
		time.Sleep(time.Duration(diff) * time.Second)
		changeChan <- true
	}
}

func (timeMonitor) createBlock() fullSwaybarMessageBodyBlock {
	block := fullSwaybarMessageBodyBlock{}
	// TODO: Print time to FullText
	t := time.Now()
	block.FullText = fmt.Sprintf("%s %s %02d, %d %d:%d", t.Weekday().String()[:3], t.Month().String()[:3], t.Day(), t.Year(), t.Hour(), t.Minute())
	return block
}

func (timeMonitor) name() string {
	return "" // Does not respond to clicks
}

func (timeMonitor) respondToClick(event clickEvent) {}

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
		jsonDecoder.Decode(&ncStateOutput)

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

	stdinChannel := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			buffer, err := reader.ReadString('\n')
			if err != nil { // Maybe log non io.EOF errors, if you want
				close(stdinChannel)
				return
			}
			stdinChannel <- buffer
			// time.Sleep(1 * time.Second) // To avoid spamming stdin
		}
	}()

	timeProvider := timeMonitor{}
	ncMonitor := notificationCenterMonitor{}

	blockProviders := []blockProvider{
		timeProvider,
		&ncMonitor,
	}

	providersByName := make(map[string]int)
	for i, block := range blockProviders {
		name := block.name()
		if name != "" {
			providersByName[name] = i
		}
	}

	fullBlockValues := make([]fullSwaybarMessageBodyBlock, len(blockProviders))
	blockChanged := make(monitorChan)

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
		case str := <-stdinChannel:
			trimmed := strings.Trim(str, " \n")
			if trimmed == "[" {
				// skip first [
			} else if trimmed == "]" {
				// No more stdin. Stop stdin-reading goroutine?
				// break mainLoop
			} else {
				event := decodeClickEvent(str)
				providerIndex := providersByName[event.Name]
				blockProviders[providerIndex].respondToClick(event)
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
