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
	// "golang.org/x/sys/unix"
)

type swaybarMessageHeader struct {
	Version     int       `json:"version"`
	ClickEvents bool      `json:"click_events"`
	ContSignal  os.Signal `json:"cont_signal"`
	StopSignal  os.Signal `json:"stop_signal"`
}

func sendHeader(header swaybarMessageHeader) {
	bytes, err := json.Marshal(header)
	if err != nil {
		logger.Panic(err)
	}
	fmt.Println(string(bytes))
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

type fullSwaybarMessageBodyBlock struct {
	FullText            string `json:"full_text"`
	ShortText           string `json:"short_text,omitempty"`
	Color               string `json:"color,omitempty"`
	Background          string `json:"background,omitempty"`
	Border              string `json:"border,omitempty"`
	BorderTop           *int   `json:"border_top,omitempty"`
	BorderBottom        *int   `json:"border_bottom,omitempty"`
	BorderLeft          *int   `json:"border_left,omitempty"`
	BorderRight         *int   `json:"border_right,omitempty"`
	MinWidth            *int   `json:"min_width,omitempty"` // or string whose length represents the desired width
	Align               string `json:"align,omitempty"`
	Name                string `json:"name,omitempty"`     // needed to receive click events
	Instance            string `json:"instance,omitempty"` // Click event receivers should have a unique Name-Instance pair
	Urgent              *bool  `json:"urgent,omitempty"`
	Separator           *bool  `json:"separator,omitempty"`
	SeparatorBlockWidth *int   `json:"separator_block_width,omitempty"`
	Markup              string `json:"markup,omitempty"`
}

type blockChangedMessage struct {
	index int
}

type blockProvider interface {
	monitor(changeChan chan<- blockChangedMessage, index int)
	createBlock() fullSwaybarMessageBodyBlock
	name() string // if this is non-empty, then it will receive click events
	respondToClick(event clickEvent)
}

// Can't use SIGRTMIN for some reason
const VOLUME_CHANGED_SIGNAL = syscall.SIGUSR1

type volumeProvider struct {
	leftMuted   bool
	leftVolume  int
	rightMuted  bool
	rightVolume int
}

func (vol *volumeProvider) updateVolume() {
	volAndMuted := func(line string) (int, bool) {
		numIndex := strings.Index(line, "[") + 1
		percentIndex := strings.Index(line, "%")
		volume, err := strconv.Atoi(line[numIndex:percentIndex])
		if err != nil {
			logger.Panic(err)
		}

		lineAfterNum := line[percentIndex+2:]
		mutedIndex := strings.Index(lineAfterNum, "[") + 1
		closeBracketIndex := strings.Index(lineAfterNum, "]")
		isMuted := lineAfterNum[mutedIndex:closeBracketIndex] == "off"

		return volume, isMuted
	}

	output, err := exec.Command("amixer", "get", "Master").Output()
	if err != nil {
		logger.Panic(err)
	}

	lines := strings.Split(string(output), "\n")
	lines = lines[len(lines)-3:]

	vol.leftVolume, vol.leftMuted = volAndMuted(lines[0])
	vol.rightVolume, vol.rightMuted = volAndMuted(lines[1])
}

func (vol *volumeProvider) monitor(changeChan chan<- blockChangedMessage, index int) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, VOLUME_CHANGED_SIGNAL)
	vol.updateVolume()

	for {
		sig := <-signals
		if sig == VOLUME_CHANGED_SIGNAL {
			leftVol, leftMute, rightVol, rightMute := vol.leftVolume, vol.leftMuted, vol.rightVolume, vol.rightMuted
			vol.updateVolume()

			if vol.leftVolume != leftVol || vol.leftMuted != leftMute || vol.rightVolume != rightVol || vol.rightMuted != rightMute {
				changeChan <- blockChangedMessage{
					index: index,
				}
			}
		}
	}
}

func (vol *volumeProvider) createBlock() fullSwaybarMessageBodyBlock {
	getVolumeString := func(vol int, muted bool) string {
		if muted {
			return " mute"
		}
		return fmt.Sprintf(" %d%%", vol)
	}

	var block fullSwaybarMessageBodyBlock

	if vol.leftMuted == vol.rightMuted || vol.leftVolume == vol.rightVolume {
		block.FullText = getVolumeString(vol.leftVolume, vol.leftMuted)
	} else {
		block.FullText = fmt.Sprintf("L:%s R:%s", getVolumeString(vol.leftVolume, vol.leftMuted), getVolumeString(vol.rightVolume, vol.rightMuted))
	}

	return block
}

func (vol *volumeProvider) name() string {
	return "volume"
}

func (vol *volumeProvider) respondToClick(event clickEvent) {
	exec.Command("alacritty", "--class", "alsamixer", "-e", "alsamixer").Run()
}

// ---

type weatherProvider struct {
	weatherStatus string
}

func (w *weatherProvider) monitor(changeChan chan<- blockChangedMessage, index int) {
	request, err := http.NewRequest("GET", "https://wttr.in?0&T&Q", nil)
	if err != nil {
		logger.Println("Cannot create request", err)
		return
	}
	request.Header["User-Agent"] = []string{"curl/8.0.1"}

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

			changeChan <- blockChangedMessage{
				index: index,
			}
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

func (ip ipAddressProvider) monitor(changeChan chan<- blockChangedMessage, index int) {
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
	}

	block.FullText = ip.text

	return block
}

func (ipAddressProvider) name() string {
	return "network"
}

func (ipAddressProvider) respondToClick(event clickEvent) {
	exec.Command("alacritty", "--class", "network_manager", "-e", "nmtui").Run()
}

// ---

type temperatureProvider struct {
	text string
}

func (temp *temperatureProvider) monitor(changeChan chan<- blockChangedMessage, index int) {
	for {
		sensorInfo, err := exec.Command("sensors").Output()
		if err != nil {
			logger.Panic(err)
		}

		maxNum := 0
		maxString := ""
		for _, line := range strings.Split(string(sensorInfo), "\n") {
			if strings.HasPrefix(line, "Core") {
				numIndex := strings.Index(line, "+") + 1
				line = line[numIndex:]

				numEndIndex := strings.Index(line, ".")
				cIndex := strings.Index(line, "C") + 1

				num, err := strconv.Atoi(line[:numEndIndex])
				if err != nil {
					logger.Panic(err)
				}

				if num > maxNum {
					maxNum = num
					maxString = line[:cIndex]
				}

			}
		}

		if temp.text != maxString {
			temp.text = maxString
			changeChan <- blockChangedMessage{
				index: index,
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

func (temp *temperatureProvider) createBlock() fullSwaybarMessageBodyBlock {
	// /Core/ { X=substr($3, 2, 4)+0; if(X > M) M = X } END { print "  " M " °C " }
	var block fullSwaybarMessageBodyBlock

	block.FullText = "  " + temp.text

	return block
}

func (temp *temperatureProvider) name() string {
	return ""
}

func (temp *temperatureProvider) respondToClick(event clickEvent) {}

// ---

type timeMonitor struct{}

func (timeMonitor) monitor(changeChan chan<- blockChangedMessage, index int) {
	for {
		t := time.Now()
		diff := 60 - t.Second()
		time.Sleep(time.Duration(diff) * time.Second)
		changeChan <- blockChangedMessage{
			index: index,
		}
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

func (nc *notificationCenterMonitor) monitor(changeChan chan<- blockChangedMessage, index int) {
	ncMonitor := exec.Command("swaync-client", "-swb")
	stdout, err := ncMonitor.StdoutPipe()
	if err != nil {
		logger.Panic(err)
	}
	jsonDecoder := json.NewDecoder(stdout)
	ncMonitor.Start()

	for {
		var ncStateOutput ncClientOutput
		err = jsonDecoder.Decode(&ncStateOutput)
		if err != nil {
			logger.Panic(err)
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
			changeChan <- blockChangedMessage{
				index: index,
			}

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
		logger.Panic(err)
	}

	return result
}

func updateSingleBlock(fullBlockValues []fullSwaybarMessageBodyBlock, index int, provider blockProvider) {
	fullBlock := provider.createBlock()

	// Set name here to make sure that it responds to clicks if it needs to
	fullBlock.Name = provider.name()
	fullBlockValues[index] = fullBlock
}

func updateFullBlockValues(fullBlockValues []fullSwaybarMessageBodyBlock, blockProviders []blockProvider) {
	for i, provider := range blockProviders {
		updateSingleBlock(fullBlockValues, i, provider)
	}
}

func displayStatusBar(fullBlockValues []fullSwaybarMessageBodyBlock, blockProviders []blockProvider, indexToUpdate int) {
	if indexToUpdate < 0 {
		logger.Println("Updating all blocks")
		updateFullBlockValues(fullBlockValues, blockProviders)
	} else {
		logger.Println("Updating block", indexToUpdate)
		updateSingleBlock(fullBlockValues, indexToUpdate, blockProviders[indexToUpdate])
	}

	bytes, err := json.Marshal(fullBlockValues)
	if err != nil {
		logger.Panic(err)
	}
	str := string(bytes)
	logger.Println("Data", str)
	fmt.Println(str, ",")
}

func defaultHeader() swaybarMessageHeader {
	result := swaybarMessageHeader{
		Version:     1,
		ClickEvents: true,
		ContSignal:  syscall.SIGCONT,
		StopSignal:  syscall.SIGSTOP,
	}

	return result
}

func mainLoop(stdinChannel <-chan clickEvent, blockChanged <-chan blockChangedMessage, blockProviders []blockProvider) {
	stdinNeverWriteToMe := make(<-chan clickEvent) // This channel is never written to and so it always blocks. This is in case stdinChannel is closed
	fullBlockValues := make([]fullSwaybarMessageBodyBlock, len(blockProviders))

	providersByName := make(map[string]int)
	for i, block := range blockProviders {
		name := block.name()
		if name != "" {
			providersByName[name] = i
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCONT, syscall.SIGSTOP)

	header := defaultHeader()

	sendHeader(header)
	fmt.Print("[")

	displayStatusBar(fullBlockValues, blockProviders, -1)

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
				return
			}

		case changeInfo := <-blockChanged:
			displayStatusBar(fullBlockValues, blockProviders, changeInfo.index)
		}
	}
}

func setupStdinReader() <-chan clickEvent {
	stdinChannel := make(chan clickEvent, 1)
	go func(stdinChannel chan<- clickEvent) {
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
	}(stdinChannel)

	return stdinChannel
}

func setupBlockChangeNotifier(blockProviders []blockProvider) <-chan blockChangedMessage {
	blockChanged := make(chan blockChangedMessage)

	// Update swaybar with initial info so you don't have to wait until a block updates
	for index, block := range blockProviders {
		go block.monitor(blockChanged, index)
	}

	return blockChanged
}

var logger *log.Logger

func setupLogger() *os.File {
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}

	directory := filepath.Dir(path)
	logsPath := filepath.Join(directory, "logs.txt")
	logsFile, err := os.OpenFile(logsPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	logsFile.Truncate(0)

	logger = log.New(logsFile, "", 0)
	return logsFile
}

func main() {
	logsFile := setupLogger()
	defer logsFile.Close()

	volume := volumeProvider{}
	weather := weatherProvider{}
	ipProvider := ipAddressProvider{}
	temperature := temperatureProvider{}
	timeProvider := timeMonitor{}
	ncProvider := notificationCenterMonitor{}

	blockProviders := []blockProvider{
		&volume,
		&weather,
		&ipProvider,
		&temperature,
		// battery
		// Bluetooth
		timeProvider,
		&ncProvider,
	}

	stdinChannel := setupStdinReader()
	blockChanged := setupBlockChangeNotifier(blockProviders)

	mainLoop(stdinChannel, blockChanged, blockProviders)
}
