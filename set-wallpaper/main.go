package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/exp/slices"
)

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
}

// TODO: Do image manipulation with an internal go library
func imageMagik(cmd ...string) {
	// NOTE convert is VERY slow
	convertCommand := exec.Command("convert", cmd...)
	convertCommand.Stdout = os.Stdout
	convertCommand.Stderr = os.Stderr

	err := convertCommand.Run()
	if err != nil {
		fmt.Println("Error executing first convert command", err)
		os.Exit(1)
	}
}

type messageType int

// Basic messages
const (
	IPC_COMMAND   = 0
	IPC_SUBSCRIBE = 2
	IPC_SEND_TICK = 10
	IPC_SYNC      = 11
)

// Queries
const (
	IPC_GET_WORKSPACES    = 1
	IPC_GET_OUTPUTS       = 3
	IPC_GET_TREE          = 4
	IPC_GET_MARKS         = 5
	IPC_GET_BAR_CONFIG    = 6
	IPC_GET_VERSION       = 7
	IPC_GET_BINDING_MODES = 8
	IPC_GET_CONFIG        = 9
	IPC_GET_BINDING_STATE = 12

	/* sway-specific command types */
	IPC_GET_INPUTS = 100
	IPC_GET_SEATS  = 101
)

// Events
const (
	IPC_EVENT_WORKSPACE        = ((1 << 31) | 0)
	IPC_EVENT_OUTPUT           = ((1 << 31) | 1)
	IPC_EVENT_MODE             = ((1 << 31) | 2)
	IPC_EVENT_WINDOW           = ((1 << 31) | 3)
	IPC_EVENT_BARCONFIG_UPDATE = ((1 << 31) | 4)
	IPC_EVENT_BINDING          = ((1 << 31) | 5)
	IPC_EVENT_SHUTDOWN         = ((1 << 31) | 6)
	IPC_EVENT_TICK             = ((1 << 31) | 7)

	/* sway-specific event types */
	IPC_EVENT_BAR_STATE_UPDATE = ((1 << 31) | 20)
	IPC_EVENT_INPUT            = ((1 << 31) | 21)
)

func swayMsgCommand(msgType messageType) []byte {
	const i3MagicString = "i3-ipc"
	const IPC_HEADER_SIZE = (uintptr(len(i3MagicString)) + 2*unsafe.Sizeof(int32(0)))

	var socketPath string = os.Getenv("SWAYSOCK")
	connection, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println("Unable to create connection", err)
		return []byte{}
	}

	length := int32(0)
	var lengthAndType [8]byte
	binary.LittleEndian.PutUint32(lengthAndType[0:4], uint32(length))
	binary.LittleEndian.PutUint32(lengthAndType[4:8], uint32(msgType))
	message := append([]byte(i3MagicString), lengthAndType[:]...)
	connection.Write(message)

	responseHeader := make([]byte, IPC_HEADER_SIZE)
	_, err = connection.Read(responseHeader)
	if err != nil {
		fmt.Println("Error when reading response header", err)
		return []byte{}
	}

	responseLength := binary.LittleEndian.Uint32(responseHeader[len(i3MagicString) : len(i3MagicString)+4])
	// responseType := binary.LittleEndian.Uint32(responseHeader[len(i3MagicString)+4:])

	response := make([]byte, responseLength)
	_, err = connection.Read(response)
	if err != nil {
		fmt.Println("Error when reading response payload", err)
		return []byte{}
	}

	return response
}

type SwayTreeJSON struct {
	Dimensions struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"rect"`
}

func getScreenDimensionsSway() (int, int) {
	jsonBytes := swayMsgCommand(IPC_GET_TREE)

	var swayTreeJson SwayTreeJSON
	err := json.Unmarshal(jsonBytes, &swayTreeJson)
	if err != nil {
		fmt.Println("Json parse error", err)
		os.Exit(1)
	}

	screenWidth, screenHeight := swayTreeJson.Dimensions.Width, swayTreeJson.Dimensions.Height

	return screenWidth, screenHeight
}

type SwayOutputJSON struct {
	Name string `json:"name"`
}

func getAllOutputs() []string {
	jsonBytes := swayMsgCommand(IPC_GET_OUTPUTS)

	var swayOutputs []SwayOutputJSON
	err := json.Unmarshal(jsonBytes, &swayOutputs)
	if err != nil {
		fmt.Println("Json parse error", err)
		os.Exit(1)
	}

	outputNames := []string{}
	for _, Output := range swayOutputs {
		outputNames = append(outputNames, Output.Name)
	}

	return outputNames
}

func getCurrentWallpaperDirectories() []string {
	homeDir, _ := os.UserHomeDir()
	defaultWallpaperDirectory := homeDir + "/wallpapers"
	result := []string{}
	wallpaperParentDirFile := homeDir + "/.config/wallpaper-directories"

	if _, err := os.Stat(wallpaperParentDirFile); !os.IsNotExist(err) {
		pathBytes, err := os.ReadFile(wallpaperParentDirFile)
		if err != nil {
			fmt.Println("Error when reading contents of", wallpaperParentDirFile, err)
			os.Exit(1)
		}

		paths := strings.Split(string(pathBytes), "\n")
		for _, path := range paths {
			if strings.TrimSpace(path) != "" {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					result = append(result, path)
				} else {
					// Soft error, fallback to default
					fmt.Println("Could not find directory at", path,
						"Read from", wallpaperParentDirFile,
						"original error:", err)
				}
			}
		}
	}

	if len(result) == 0 {
		result = []string{defaultWallpaperDirectory}
	}

	return result
}

func getAllWallpaperPaths(parentDir string, result *[]string) []string {
	files, err := os.ReadDir(parentDir)
	if err != nil {
		fmt.Println("Error when reading wallpaper directory", err)
		os.Exit(1)
	}

	for _, file := range files {
		fileName := file.Name()
		if !strings.HasPrefix(fileName, ".") {
			filePath := parentDir + "/" + fileName
			if stat, err := os.Stat(filePath); !os.IsNotExist(err) && stat.IsDir() {
				getAllWallpaperPaths(filePath, result)
			} else {
				*result = append(*result, filePath)
			}
		}
	}

	return *result
}

func setWallpaperForScreen(screen string, wallpaper string) {
	// Assume wallpaper exists

	homeDir, _ := os.UserHomeDir()
	processedWallpapersDir := homeDir + "/.local/processed-wallpapers"
	wallpaperOutputPath := processedWallpapersDir + "/wallpaper-" + screen + ".png"
	lockScreenWallpaperPath := processedWallpapersDir + "/lock-screen-" + screen + ".png"
	width, height := getScreenDimensionsSway()

	screenDimensions := fmt.Sprintf("%dx%d", width, height)

	os.Stderr.WriteString("Creating lock screen wallpaper\n")
	imageMagik(
		"-gravity", "center",
		"-crop", "16:9", // TODO: aspect ratio calculation
		"-resize", fmt.Sprintf("%dx%d", width/10, height/10),
		"-filter", "Gaussian",
		"-blur", "0x2.5",
		"-resize", screenDimensions+"^",
		wallpaper, lockScreenWallpaperPath)

	os.Stderr.WriteString("Creating desktop wallpaper\n")
	imageMagik(
		"-gravity", "center",
		"(", wallpaper, "-resize", screenDimensions, ")",
		"(", "+clone", "-background", "black", "-shadow", "60x10+20+20", ")",
		"+swap",
		"-compose", "over", "-composite",
		"(", lockScreenWallpaperPath, "-resize", screenDimensions+"^", ")",
		"+swap",
		"-compose", "over", "-composite",
		wallpaperOutputPath)

	fmt.Println("Updating output to", screen, wallpaperOutputPath)
	swayMsgCommand := exec.Command("swaymsg", "output", screen, "bg", wallpaperOutputPath, "fit")
	swayMsgCommand.Stdout = os.Stdout
	swayMsgCommand.Stderr = os.Stderr
	swayMsgCommand.Run()
}

func main() {
	outputs := getAllOutputs()
	wallpaperDirs := getCurrentWallpaperDirectories()

	wallpapers := []string{}
	for _, dir := range wallpaperDirs {
		getAllWallpaperPaths(dir, &wallpapers)
	}

	homeDir, _ := os.UserHomeDir()
	processedWallpapersDir := homeDir + "/.local/processed-wallpapers"
	ensureDirExists(processedWallpapersDir)

	if len(os.Args) <= 1 {
		if len(wallpapers) > 0 {
			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)

			for _, output := range outputs {
				setWallpaperForScreen(output, wallpapers[rng.Intn(len(wallpapers))])
			}
		}
	} else {
		output := os.Args[1]
		wallpaper := ""
		if len(os.Args) > 2 {
			wallpaper = os.Args[2]
		}

		if slices.Contains(outputs, output) {
			fmt.Println(output, "is not a valid output. Options are:", outputs)
			os.Exit(1)
		}

		if slices.Contains(wallpapers, wallpaper) {
			fmt.Println("Wallpaper", wallpaper, "does not exist in path")
			os.Exit(1)
		}

		setWallpaperForScreen(output, wallpaper)
	}
}
