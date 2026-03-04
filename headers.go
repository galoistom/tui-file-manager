
package main

import (
	"strings"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"	
)


const (
	GAP= 10
	EDITOR="vim"
	SHELL="zsh"
)
var (
	temp int
	yank []string
)

type Myerror struct{
	err error
	message string
}

func (err Myerror) Error() string{
	return strings.Join([]string{err.message}, err.err.Error())
}

func HandleCreateMap (s string) int{
	switch s{
	case "f","enter": return 1
	case "d": return 2
	case "s": return 3
	}
	return -1
}

type mode int
const (
	modeNormal mode= iota
	modeSearch
	modeCommand
	modeCreate
	modeTyping
	modeDelete
	modeRename
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

type fileitm struct{
	name string
	path string
	mode string
}

type module struct{
	cursor int
	selected map[int]struct{}
	entries []fileitm
	path string
	height int
	offset int
	ti textinput.Model
	searching bool
	message string
	isError bool
	currentMode mode
	tempFile string
}

type itemsMsg []fileitm
type editorMsg struct{}
