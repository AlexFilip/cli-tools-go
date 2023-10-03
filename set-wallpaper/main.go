package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

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

func swayMsgCommand(cmd string) []byte {
	dimensionCommand := exec.Command("swaymsg", "-t", cmd)
	jsonOutput, err := dimensionCommand.StdoutPipe()
	if err != nil {
		fmt.Println("Error when getting stdout of swaymsg command", err)
		os.Exit(1)
	}

	err = dimensionCommand.Start()
	if err != nil {
		fmt.Println("Error starting swaymsg command", err)
		os.Exit(1)
	}

	jsonBytes, _ := io.ReadAll(jsonOutput)

	err = dimensionCommand.Wait()
	if err != nil {
		fmt.Println("Error executing swaymsg command", cmd, err)
		os.Exit(1)
	}

	return jsonBytes
}

type SwayTreeJSON struct {
	Dimensions struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"rect"`
}

func getScreenDimensionsSway() (int, int) {
	jsonBytes := swayMsgCommand("get_tree")

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
	jsonBytes := swayMsgCommand("get_outputs")

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

func getCurrentWallpaperDirectory() []string {
	homeDir, _ := os.UserHomeDir()
	defaultWallpaperDirectory := homeDir + "/wallpapers"
	result := []string{defaultWallpaperDirectory}

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
						"falling back to default path:", defaultWallpaperDirectory,
						"original error:", err)
				}
			}
		}
	}

	return result
}

func getAllWallpaperPaths(parentDir string, result *[]string) []string {
	files, err := ioutil.ReadDir(parentDir)
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
	wallpaperDir := getCurrentWallpaperDirectory()

	wallpapers := []string{}
	for _, wallpaper := range wallpaperDir {
		getAllWallpaperPaths(wallpaper, &wallpapers)
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
