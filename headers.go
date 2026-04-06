package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

var (
	temp        int
	yank        []string
	cache       string
	needProcess = map[string]struct{}{
		".pdf":  {},
		".mp4":  {},
		".mkv":  {},
		".docx": {},
		".xlsx": {},
		".xls":  {},
		".jpg":  {},
		".svg":  {},
		".png":  {},
		".gif":  {},
	}
)

type Myerror struct {
	err     error
	message string
}

func (err Myerror) Error() string {
	return strings.Join([]string{err.message}, err.err.Error())
}

func HandleCreateMap(s string) int {
	switch s {
	case "f", "enter":
		return 1
	case "d":
		return 2
	case "s":
		return 3
	}
	return -1
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeCommand
	modeCreate
	modeTyping
	modeDelete
	modeRename
	modeBookmark
)

var (
	// 基础颜色和边框
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#5A56E0")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	// 命令输入框样式
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
	errorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF0000")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)
)

type fileitm struct {
	name string
	path string
	mode string
}

type module struct {
	cursor      int
	selected    map[int]struct{}
	entries     []fileitm
	path        string
	height      int
	width       int
	offset      int
	ti          textinput.Model
	searching   bool
	message     string
	isError     bool
	currentMode mode
	tempFile    string
	preview     bool
	hide        bool
}

type itemsMsg []fileitm
type editorMsg struct{}
type redrawMsg struct{}

type Config struct {
	BOOKMARK map[string]string `json:"bookmark"`
	SHELL    string            `json:"shell"`
	EDITOR   string            `json:"editor"`
	GAP      int               `json:"gap"`
	CONFIG   string            `json:"config"`
}

var Configs = Config{
	SHELL:  "bash",
	EDITOR: "vim",
	GAP:    10,
	CONFIG: "~/.config/tui-fm/config.json",
}

func init() {
	file := ExpandPath(Configs.CONFIG)
	_, err := os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Println("failed to get config status: ", err)
			os.Exit(3)
		} else {
			err = exec.Command("mkdir", "-p", filepath.Dir(file)).Run()
			data, err := json.MarshalIndent(Configs, "", "    ")
			if err != nil {
				fmt.Println("failed to write config: ", err)
				os.Exit(3)
			}
			os.WriteFile(file, data, 0644)
		}
	}
	fileContent, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("Error occoured when reading: ", err)
		os.Exit(1)
	}
	err = json.Unmarshal(fileContent, &Configs)
	if err != nil {
		fmt.Println("Error unamarshalling JSON: ", err)
		os.Exit(1)
	}
	Configs.CONFIG= ExpandPath(Configs.CONFIG)
	for a:=range Configs.BOOKMARK{
		Configs.BOOKMARK[a]= ExpandPath(Configs.BOOKMARK[a])
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	cache = filepath.Join(cacheDir, "tui-fm")
	os.Mkdir(cache, 0750)
}
