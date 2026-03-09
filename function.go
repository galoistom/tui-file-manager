package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"crypto/md5"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func isKitty() bool {
	return os.Getenv("KITTY_WINDOW_ID") != ""
}

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
	out,err:=exec.Command("file", m.entries[m.cursor].path).Output()
	if err==nil{m.message=getValue(string(out))}
	if m.preview {
		return m.PreviewCmd(m.entries[m.cursor].path)
	}
	return nil
}

func getValue(s string) string {
	i := strings.Index(s, ":")
	if i == -1 {
		return ""
	}

	val := strings.TrimSpace(s[i+1:])

	if j := strings.Index(val, ","); j != -1 {
		val = val[:j]
	}

	return strings.TrimSpace(val)
}

func (m *module) Open(path string) tea.Cmd {
	return func() tea.Msg {

		cmd := exec.Command("xdg-open", path)

		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}

		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil

		_ = cmd.Start()

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
		return FetchFile(m.path,m.hide)
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
				os.Stdout.Write([]byte("\x1b_Ga=d\x1b\\"))
			}
		} else {
			m.message = "format incorrect"
		}
	case "rename":
		if len(insertCommand) == 2 {
			m.Rename(m.entries[m.cursor].path, insertCommand[1])
			return FetchFile(m.path,m.hide)
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
			if isKitty() {
				return style.Render("")
			}
			path, err := convertJPG(theFile.path, ext)
			if err != nil {
				return style.Render("failed to convert: " + err.Error())
			}
			cmd := exec.Command(
				"chafa",
				"-f", "symbols",
				"--animate=no",
				"--size", fmt.Sprintf("%dx%d", width-3, height),
				"--symbols", "block",
				path,
			)
			out, err := cmd.Output()
			if err != nil {
				return style.Render("failed to show: " + err.Error())
			}
			return style.Render(string(out))
		}
		path := theFile.path
		var index *exec.Cmd
		switch ext {
		case ".zip":
			index = exec.Command("unzip", "-l", path)
		case ".tar":
			index = exec.Command("tar", "-tf", path)
		case ".gz", ".tgz":
			index = exec.Command("tar", "-tzf", path)
		default:
			index = exec.Command(
				"bat",
				"--binary=no-printing",
				"--color=always",
				"--style=plain",
				"--paging=never",
				"-S", "-r", ":"+strconv.Itoa(height-5),
				path,
			)
		}
		out,err := index.Output()
		if err != nil {
			return "failed to cat: " + err.Error()
		}
		return style.Render(string(out))
	}
	return style.Render("unknow")
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
	os.Stdout.Write([]byte("\x1b_Ga=d\x1b\\"))
	if !isKitty() {
		return nil
	}
	ext := filepath.Ext(imagePath)
	if _, ok := needProcess[ext]; !ok {
		return nil
	}
	path, err := convertJPG(imagePath, ext)
	if err != nil {
		m.isError = true
		m.message = "failed to preview"
		return nil
	}
	yOffset := 2
	x := int(float64(m.width)*0.45) + 2
	cmd := exec.Command("kitty", "+kitten", "icat",
		"--z-index", "-1",
		"--place", fmt.Sprintf("%dx%d@%dx%d", m.width-x, m.height-3, x, yOffset),
		"--transfer-mode=file", path)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		m.isError = true
		m.message = err.Error()
	}
	return tea.ClearScreen
}
