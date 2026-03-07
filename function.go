package main

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"crypto/md5"
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

func (m *module) GotoFile(n int) tea.Cmd {
	os.Stdout.Write([]byte("\x1b_Ga=d\x1b\\"))
	m.message = ""
	m.isError = false
	if n > len(m.entries)-1 || n < 0 {
		return nil
	}
	if n > m.cursor {
		m.offset = min(max(len(m.entries)-m.height+4, 0),
			max(m.offset, n+Configs.GAP-m.height))
	} else if n < m.cursor {
		m.offset = max(0, min(m.offset, n-Configs.GAP))
	}
	m.cursor = n
	if m.preview {
		return m.PreviewCmd(m.entries[m.cursor].path)
	}
	return nil
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

func (m *module) ExecCommand() tea.Cmd {
	insertCommand := strings.Fields(m.ti.Value())
	switch insertCommand[0] {
	case "go":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return nil
		}
		m.GotoFile(n)
	case "down":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return nil
		}
		m.GotoFile(n + m.cursor)
	case "up":
		n, err := strconv.Atoi(insertCommand[1])
		if err != nil {
			m.message = "Not numbers"
			m.isError = true
			return nil
		}
		m.GotoFile(m.cursor - n)
	case "sh":
		command := exec.Command(Configs.SHELL, "-c", strings.Join(insertCommand[1:], " "))
		command.Dir = m.path
		if err := command.Run(); err != nil {
			m.message = "fialed to execute: " + err.Error()
			m.isError = true
			return nil
		}
		m.message = "Succeed!"
		return FetchFile(m.path)
	case "goto":
		if len(insertCommand) == 2 {
			path := ExpandPath(insertCommand[1])
			f, err := os.Stat(path)
			if err != nil {
				m.message = "not a correct path:" + err.Error()
				m.isError = true
				return nil
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
			return FetchFile(m.path)
		} else {
			m.message = "format incorrect"
		}
	case "copyto":
		if len(insertCommand) == 2 {
			path := ExpandPath(insertCommand[1])
			f, err := os.Stat(path)
			if err != nil {
				m.message = "not a correct path:" + err.Error()
				m.isError = true
				return nil
			}
			if !f.IsDir() {
				return nil
			}
			if len(m.selected) == 0 {
				cmd := exec.Command("cp", "-r", m.entries[m.cursor].path, path)
				if err = cmd.Run(); err != nil {
					m.message = "failed to pase: " + err.Error()
					m.isError = true
					return nil
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
	return nil
}

func ExpandPath(path string) string {
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

func (m *module) Creatf(mod int) tea.Cmd {
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
	return nil
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
		theFile := m.entries[m.cursor]
		ext := filepath.Ext(theFile.name)
		if _, ok := needProcess[ext]; ok {
			// path, err := convertJPG(theFile.path, ext)
			// if err != nil {
			// 	return style.Render("failed to convert: " + err.Error())
			// }
			// cmd := exec.Command(
			// 	"chafa",
			// 	"-f", "symbols",
			// 	"--animate=no",
			// 	"--size", fmt.Sprintf("%dx%d", width-3, height),
			// 	"--symbols", "block",
			// 	path,
			// )
			// out, err := cmd.Output()
			// if err != nil {
			// 	return style.Render("failed to show: " + err.Error())
			// }
			// return style.Render(string(out))
			return style.Render("file")
		}
		path := theFile.path
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

func getCacheName(path string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(path)))
	return filepath.Join(cache, hash)
}

func convertJPG(path string, t string) (string, error) {
	cachePath := getCacheName(path) + ".jpg"
	if _, err := os.Stat(cachePath); err == nil {
		return getCacheName(path) + ".jpg", nil
	}
	var cmd exec.Cmd
	switch t {
	case ".pdf":
		cmd = *exec.Command("pdftoppm", "-jpeg", "-f", "1", "-singlefile", path, getCacheName(path))
	case ".doc", ".xls", ".docx", ".xlsx", "pptx":
		cmd = *exec.Command("libreoffice", "--convert-to", "jpg", path, "--outdir", filepath.Dir(cachePath))
		if err := cmd.Run(); err != nil {
			return "", err
		}
		baseName := filepath.Base(path)
		ext := filepath.Ext(baseName)
		defaultOutputName := strings.TrimSuffix(baseName, ext) + ".jpg"
		defaultOutputPath := filepath.Join(cache, defaultOutputName)

		err := os.Rename(defaultOutputPath, cachePath)
		if err != nil {
			return "", fmt.Errorf("rename failed: %v", err)
		}
	case ".mp4", ".mkv", ".mov":
		cmd = *exec.Command("ffmpeg", "-i", path, "-frames:v", "1", cachePath)
	default:
		return path, nil
	}
	if err := cmd.Run(); err != nil {
		return "", Myerror{message: getCacheName(path) + " " + path, err: err}
	}
	return cachePath, nil
}

func (m *module) PreviewCmd(imagePath string) tea.Cmd {
	ext := filepath.Ext(imagePath)
	if _,ok:=needProcess[ext];!ok{
		return nil
	}
	path, err := convertJPG(imagePath, ext)
	if err != nil {
		m.isError = true
		m.message = "failed to preview"
		return nil
	}
	return func() tea.Msg {
		yOffset := 2
		x := int(float64(m.width) * 0.45)+1 // 图片起始列
		cmd := exec.Command("kitty", "+kitten", "icat",
			"--z-index", "1",
			"--place", fmt.Sprintf("%dx%d@%dx%d",m.width-x,m.height,x,yOffset),
			"--transfer-mode=file", path)
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr= os.Stderr
		err := cmd.Start()
		if err != nil {
			return err.Error()
		}
		return nil
	}
}

// args := []string{
//     "+kitten", "icat",
//     "--silent",
// 	"--transfer-mode", "file",
// 	"--z-index", "1",
//     "--place", fmt.Sprintf("%dx@%dx%d", w, x, yOffset),
//     path,
// }
