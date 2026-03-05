package main

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (m *module) GotoFile(n int) {
	m.message = ""
	m.isError = false
	if n > len(m.entries)-1 || n < 0 {
		return
	}
	if n > m.cursor {
		m.offset = min(max(len(m.entries)-m.height+4, 0),
			max(m.offset, n+Configs.GAP-m.height))
	} else if n < m.cursor {
		m.offset = max(0, min(m.offset, n-Configs.GAP))
	}
	m.cursor = n
}

func (m *module) Open(path string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command("xdg-open", path).Start()
		if err != nil {
			return err
		}
		return nil
	}
}

func OpenShell(path string, command string) tea.Cmd {
	c := exec.Command("sh", "-c", command)
	c.Dir = path
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return err
		}
		return editorMsg{}
	})
}

func matchSimple(fileName, pattern string) bool {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return false
	}
	return re.MatchString(fileName)
}

func (m *module) RenameUpdate() {
	context, err := os.ReadFile(m.tempFile)
	if err != nil {
		m.message = "failed to update renaming: " + err.Error()
		m.isError = true
		return
	}
	lines := strings.Lines(string(context))
	for line := range lines {
		l := strings.Fields(line)
		if len(l) > 2 {
			current, err := strconv.Atoi(l[0])
			if err != nil {
				m.message = "number convertion fialed: " + err.Error()
				m.isError = true
				os.Remove(m.tempFile)
				continue
			}
			oldName := m.entries[current].path
			newName := filepath.Join(m.path, strings.Join(l[2:], " "))
			m.Rename(oldName, newName)
			os.Remove(m.tempFile)
		}
	}
	m.message = "completes"
	os.Remove(m.tempFile)
}

func (m *module) Rename(oldName, newName string) {
	if oldName == newName {
		return
	}
	if _, err := os.Stat(newName); err == nil {
		newName += "_" + strconv.Itoa(rand.Intn(100))
	}
	if err := os.Rename(oldName, newName); err != nil {
		m.message = "fialed to rename: " + err.Error()
		m.isError = true
		return
	}
}

func (m *module) Search(pattern string, place int, mod bool) int {
	if !mod {
		for i := place - 1; i >= 0; i-- {
			if matchSimple(m.entries[i].name, pattern) {
				return i
			}
		}
	} else {
		end := len(m.entries)
		for i := place + 1; i < end; i++ {
			if matchSimple(m.entries[i].name, pattern) {
				return i
			}
		}
	}
	return -1
}

func (m *module) ExecCommand() {
	insertCommand := strings.Fields(m.ti.Value())
	switch insertCommand[0] {
	case "go":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return
		}
		m.GotoFile(n)
	case "down":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return
		}
		m.GotoFile(n + m.cursor)
	case "up":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return
		}
		m.GotoFile(m.cursor - n)
	case "sh":
		command := exec.Command(Configs.SHELL, "-c", strings.Join(insertCommand[1:], " "))
		command.Dir = m.path
		if err := command.Run(); err != nil {
			m.message = "fialed to execute: " + err.Error()
			m.isError = true
			return

		}
	case "goto":
		if len(insertCommand) == 2 {
			path := expandPath(insertCommand[1])
			f, err := os.Stat(path)
			if err != nil {
				m.message = "not a correct path:" + err.Error()
				m.isError = true
				return
			}
			if f.IsDir() {
				m.path = path
				m.cursor = 0
				m.offset = 0
			}
		} else {
			m.message = "format incorrect"
		}
	case "rename":
		if len(insertCommand) == 2 {
			m.Rename(m.entries[m.cursor].path, insertCommand[1])
		} else {
			m.message = "format incorrect"
		}
	case "copyto":
		if len(insertCommand) == 2 {
			path := expandPath(insertCommand[1])
			f, err := os.Stat(path)
			if err != nil {
				m.message = "not a correct path:" + err.Error()
				m.isError = true
				return
			}
			if !f.IsDir() {
				return
			}
			if len(m.selected) == 0 {
				cmd := exec.Command("cp", "-r", m.entries[m.cursor].path, path)
				if err = cmd.Run(); err != nil {
					m.message = "failed to pase: " + err.Error()
					m.isError = true
					return
				}
			} else {
				for i := range m.selected {
					cmd := exec.Command("cp", "-r", m.entries[i].path, path)
					if err = cmd.Run(); err != nil {
						m.message = "failed to pase: " + err.Error()
						m.isError = true
					}
				}
			}
		} else {
			m.message = "format incorrect"
		}
	default:
		m.message = fmt.Sprintf("Unknow command: %s", insertCommand[0])
	}
}

func expandPath(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if path[1] == '/' || path[1] == '\\' {
		return filepath.Join(home, path[1:])
	}
	return path
}

func (m *module) Creatf(mod int) {
	path := filepath.Join(m.path, m.ti.Value())
	switch mod {
	case 1:
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			m.isError = true
			m.message = fmt.Sprintf("fialed to create: %v", err)
		} else {
			f.Close()
			m.message = "file created successfully"
		}
	case 2:
		if err := os.MkdirAll(path, 0750); err != nil {
			m.isError = true
			m.message = fmt.Sprintf("fialed to create: %v", err)
		}
	case 3:
		if err := os.Symlink(m.entries[m.cursor].path, path); err != nil {
			m.isError = true
			m.message = fmt.Sprintf("fialed to create: %v", err)
		}
	}
}

func (m module) Preview(width int, height int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Padding(1).
		Border(lipgloss.NormalBorder(), false, false, false, true)
	if m.cursor == 0 {
		return style.Render("up a dictionary")
	}
	f, err := os.Stat(m.entries[m.cursor].path)
	if err != nil {
		return "failed to preview: " + err.Error()
	}
	if f.IsDir() {
		out, err := exec.Command("tree", "-L",
			"3", m.entries[m.cursor].path).Output()
		if err != nil {
			return "failed to review tree: " + err.Error()
		}
		return style.Render(string(out))
	}
	if f.Mode().String()[0] == '-' {
		path := m.entries[m.cursor].path
		index := exec.Command("cat", "-n", path)
		restrict := exec.Command("head", "-n", strconv.Itoa(height-5))
		pipe, err := index.StdoutPipe()
		if err != nil {
			return "failed to pipe: " + err.Error()
		}
		restrict.Stdin = pipe
		err = index.Start()
		if err != nil {
			return "failed to cat: " + err.Error()
		}
		out, err := restrict.Output()
		if err != nil {
			return "failed to get out: " + err.Error()
		}
		return style.Render(highlightCode(path, string(out)))
	}
	return style.Render("unknow")
}

func highlightCode(path string, content string) string {
	// 1. 根据文件名获取对应的词法分析器 (Lexer)
	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// 4. 进行高亮处理
	var buf bytes.Buffer
	iterator, _ := lexer.Tokenise(nil, content)
	formatter.Format(&buf, style, iterator)

	return buf.String()
}
