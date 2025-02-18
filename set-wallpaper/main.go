package main

// TODO
//  Get from environment variable
//   - config file that specifies all wallpaper directories (or just the directories themselves)
//   - processed-wallpapers directory
//   - wallpapers directory

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	// "image/color"
	"image/png"
	"math/rand"
	"net"
	"os"
	"path"
	"strings"
	"time"
	"unsafe"

	"github.com/disintegration/gift"
	"golang.org/x/exp/slices"
)

func swap[T any](first, second *T) {
	temp := *first
	*first = *second
	*second = temp
}

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
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

func swayMsgCommand(msgType messageType, payload string) []byte {
	const i3MagicString = "i3-ipc"
	const IPC_HEADER_SIZE = (uintptr(len(i3MagicString)) + 2*unsafe.Sizeof(int32(0)))

	socketPath := os.Getenv("SWAYSOCK")
	connection, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println("Unable to create connection", err)
		return []byte{}
	}

	length := uint32(len(payload))
	var lengthAndType [8]byte
	binary.LittleEndian.PutUint32(lengthAndType[0:4], length)
	binary.LittleEndian.PutUint32(lengthAndType[4:8], uint32(msgType))
	message := append([]byte(i3MagicString), lengthAndType[:]...)
	connection.Write(message)
	connection.Write([]byte(payload))

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
	jsonBytes := swayMsgCommand(IPC_GET_TREE, "")

	var swayTreeJson SwayTreeJSON
	err := json.Unmarshal(jsonBytes, &swayTreeJson)
	if err != nil {
		fmt.Println("Json parse error", err)
		os.Exit(1)
	}

	return swayTreeJson.Dimensions.Width, swayTreeJson.Dimensions.Height
}

type SwayOutputJSON struct {
	Name string `json:"name"`
}

func getAllOutputs() []string {
	jsonBytes := swayMsgCommand(IPC_GET_OUTPUTS, "")

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
	defaultWallpaperDirectory := path.Join(homeDir, "wallpapers")
	result := []string{}
	wallpaperParentDirFile := path.Join(homeDir, ".config/wallpaper-directories")

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
			filePath := path.Join(parentDir, fileName)
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

	fmt.Printf("Using %s for %s\n", wallpaper, screen)
	// homeDir, _ := os.UserHomeDir()
	processedWallpapersRelativeDir := ".local/processed-wallpapers"
	wallpaperOutputPath := path.Join(processedWallpapersRelativeDir, "wallpaper-"+screen+".png")
	lockScreenWallpaperPath := path.Join(processedWallpapersRelativeDir, "lock-screen-"+screen+".png")

	screenWidth, screenHeight := getScreenDimensionsSway()

	os.Stderr.WriteString("Creating lock screen wallpaper\n")
	file, err := os.Open(wallpaper)
	if err != nil {
		fmt.Printf("Could not load file \"%s\" with error: %+v\n", wallpaper, err)
		os.Exit(1)
	}
	defer file.Close()

	img, _ /* format_name */, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Could not decode image \"%s\" with error: %+v\n", wallpaper, err)
		os.Exit(1)
	}

	imgBounds := img.Bounds()

	newDesktopHeight := screenHeight
	newDesktopWidth := (imgBounds.Dx() * screenHeight) / imgBounds.Dy()

	newLockScreenWidth := screenWidth
	newLockScreenHeight := (imgBounds.Dy() * screenWidth) / imgBounds.Dx()

	if newLockScreenHeight < screenHeight {
		fmt.Println("Swapping locks screen and desktop dims")
		swap(&newDesktopHeight, &newLockScreenHeight)
		swap(&newDesktopWidth, &newLockScreenWidth)
	}

	screenRect := image.Rectangle{
		Min: image.Pt(0, 0),
		Max: image.Pt(screenWidth, screenHeight),
	}

	// Draw lock screen image
	lockScreenFilter := gift.New(
		gift.GaussianBlur(5.0),
		gift.Resize(newLockScreenWidth, newLockScreenHeight, gift.LinearResampling),
		gift.CropToSize(screenWidth, screenHeight, gift.CenterAnchor),
	)

	lockScreenOutputImage := image.NewRGBA(screenRect)
	lockScreenFilter.Draw(lockScreenOutputImage, img)

	lockScreenFile, err := os.Create(lockScreenWallpaperPath)
	if err != nil {
		fmt.Printf("Could not create image at \"%s\". Error: %+v\n", lockScreenWallpaperPath, err)
		os.Exit(1)
	}
	defer lockScreenFile.Close()

	png.Encode(lockScreenFile, lockScreenOutputImage)

	// Draw Desktop Image
	os.Stderr.WriteString("Creating desktop wallpaper\n")
	desktopFilter := gift.New(gift.Resize(newDesktopWidth, newDesktopHeight, gift.LinearResampling))

	desktopOutputImage := image.NewRGBA(screenRect)

	lockScreenFilter.Draw(desktopOutputImage, img)

	centeredOrigin := image.Pt(screenWidth/2-newDesktopWidth/2, screenHeight/2-newDesktopHeight/2)
	desktopFilter.DrawAt(desktopOutputImage, img, centeredOrigin, gift.OverOperator)

	fmt.Printf("              Image dims: (%d, %d)\n", imgBounds.Dx(), imgBounds.Dy())
	fmt.Printf("             Screen dims: (%d, %d)\n", screenWidth, screenHeight)
	fmt.Printf("        Lock screen dims: (%d, %d)\n", newLockScreenWidth, newLockScreenHeight)
	fmt.Printf("            Desktop dims: (%d, %d)\n", newDesktopWidth, newDesktopHeight)
	fmt.Printf("Lock screen image bounds: %+v\n", lockScreenOutputImage.Bounds())
	fmt.Printf("    Desktop image bounds: %+v\n", desktopOutputImage.Bounds())

	fmt.Printf("  Lock screen bounds after filter: %+v\n", lockScreenFilter.Bounds(imgBounds))
	fmt.Printf("Desktop image bounds after filter: %+v\n", desktopFilter.Bounds(imgBounds))

	desktopFile, err := os.Create(wallpaperOutputPath)
	if err != nil {
		fmt.Printf("Could not create image at \"%s\". Error: %+v\n", wallpaperOutputPath, err)
		os.Exit(1)
	}
	defer desktopFile.Close()
	png.Encode(desktopFile, desktopOutputImage)

	// TODO: Drop shadow
	// maybeDropShadowFilter := gift.New(
	// 	gift.GaussianBlur(5.0), // Apply a blur to simulate shadow
	// 	gift.ColorFunc(func(r, g, b, a float32) (rf, gf, bf, af float32) {
	// 		return float32(0), float32(0), float32(0), 1.0
	// 	}),
	// )

	fmt.Println("Updating output to", screen, wallpaperOutputPath)
	swayMsgCommand(IPC_COMMAND, fmt.Sprintf("output \"%s\" bg \"%s\" fit", screen, wallpaperOutputPath))
}

func main() {
	outputs := getAllOutputs()
	wallpaperDirs := getCurrentWallpaperDirectories()

	wallpapers := []string{}
	for _, dir := range wallpaperDirs {
		getAllWallpaperPaths(dir, &wallpapers)
	}

	homeDir, _ := os.UserHomeDir()
	processedWallpapersDir := path.Join(homeDir, ".local/processed-wallpapers")
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
