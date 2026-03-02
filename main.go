package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const GAP= 10

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
	typing bool
	message string
	isError bool
}

type itemsMsg []fileitm
type editorMsg struct{}

func initialModel(path string) module {
	ti:= textinput.New()
	ti.Placeholder= "input command"
	ti.Prompt=""
	ti.Focus()
	return module{
		ti: ti,
		selected: make(map[int]struct{}),
		cursor: 0,
		path: path,
	}
}

func (m module) Init() tea.Cmd {
	return fetchFile(m.path)
}

func fetchFile(path string) tea.Cmd {
	return func() tea.Msg {
		ent, err:= os.ReadDir(path)
		if err!=nil{
			return err
		}
		items:= []fileitm{{
			name: "../",
			path: filepath.Dir(path),
			mode: "d---------",
		}}
		for _, entry:= range ent {
			info, _:= entry.Info()
			items= append(items, fileitm{
				name: entry.Name(),
				path: filepath.Join(path, entry.Name()),
				mode: info.Mode().String(),
			})
		}
		return itemsMsg(items)
	}
}

func open(path string) tea.Cmd{
	return func() tea.Msg{
		err:= exec.Command("xdg-open", path).Start()
		if err!=nil{return err}
		return nil
	}
}

func clearMessageAfter(d time.Duration) tea.Cmd {
    return tea.Tick(d, func(t time.Time) tea.Msg {
        return clearMsg{} // 定义一个空结构体消息
    })
}

type clearMsg struct{}

func (m *module) gotoFile(n int) {
	if n>len(m.entries)-1 || n<0{
		return
	}
	if n>m.cursor{
		m.offset=min(max(len(m.entries)-m.height,0), max(m.offset, n+GAP-m.height))
	} else if n< m.cursor{
		m.offset=max(0, min(m.offset, n-GAP))
	}
	m.cursor=n
}

func openEdit(path string) tea.Cmd {
	c:= exec.Command("vim", path)
	return tea.ExecProcess(c, func(err error) tea.Msg{
		if err!=nil{
			return err
		}
		return  editorMsg{}
	})
}

func (m *module) execCommand() {
	insertCommand:= strings.Fields(m.ti.Value())
	switch insertCommand[0] {
	case "goto" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.gotoFile(n)
	case "down" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.gotoFile(n+m.cursor)
	case "up" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.gotoFile(m.cursor-n)
	default:
		m.message= fmt.Sprintf("Unknow command: %s", insertCommand[0])
	}
	
}

func (m module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.typing{
		switch msg:= msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "ctrl+g":
				m.typing=false
				m.ti.SetValue("")
				return m,nil
			case "enter":
				m.message= ""
				m.isError= false
				m.execCommand()
				m.typing=false
				m.ti.SetValue("")
				return m,clearMessageAfter(3*time.Second)
			default:
				m.ti, cmd= m.ti.Update(msg)
			}
		}
		return m,cmd
	}
	switch msg := msg.(type) {
	case clearMsg:
		m.message= ""
		m.isError= false
	case tea.WindowSizeMsg:
		m.height= msg.Height - 4
	
	case itemsMsg:
		m.entries= msg

	case editorMsg:
		return m,fetchFile(m.path)

	        // Is it a key press?
	case tea.KeyPressMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {
		

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		case "g", "alt+shift+,":
			m.gotoFile(0)

		case "G", "alt+shift+.":
			m.gotoFile(len(m.entries)-1)

		// The "up" and "k" keys move the cursor up
		case "up", "k", "ctrl+p":
			m.gotoFile(m.cursor-1)
			
		// The "down" and "j" keys move the cursor down
		case "down", "j", "ctrl+n":
			m.gotoFile(m.cursor+1)
			
		// for the item that the cursor is pointing at.
		case "space":
			_, ok := m.selected[m.cursor]
                        if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
			m.gotoFile(m.cursor+1)

		//press enter or l to open 
		case "enter", "l":
			selected:= m.entries[m.cursor]
			if selected.mode[0]!='-' {
				m.path= selected.path
				m.cursor=0
				m.offset=0
				return m, fetchFile(m.path)
			} else {
				return m, open(selected.path)
			}

		// press e to edit file in vim
		case "e":
			selected:= m.entries[m.cursor]
			if selected.mode[0]=='-'{
				return m, openEdit(selected.path)
			}

		//press alt+x or : to input command
		case "alt+x", ":":
			m.typing=true
			m.ti.Focus()
		}
	}

        // Return the updated model to the Bubble Tea runtime for processing.
        // Note that we're not returning a command.
        return m, nil
}

func (m module) View() tea.View {
	var  rows []string
	rows = append(rows, headerStyle.Render("📂 "+m.path))
	reserve:= 3
	listHeith:= m.height- reserve
	if listHeith<0 {listHeith=1}

	end:= min(m.offset + m.height, len(m.entries))
	visible:= m.entries[m.offset:end]
	for i, item := range visible{
		absolute:= i+ m.offset
		cursor:= " "
		if m.cursor== absolute{
			cursor=">"
		}
		checked:= " "
		if _, ok:= m.selected[absolute]; ok{
			checked="x"
		}
		lineContent:= fmt.Sprintf("%s%s %s %s", cursor, checked, item.mode, item.name)
		if m.cursor== absolute{
			rows= append(rows, selectedStyle.Render(lineContent))
		} else{
			rows= append(rows, lineContent)
		}
	}
	for len(rows)< m.height-1 {
		rows= append(rows,"")
	}
	footer:= dimStyle.Render("\n type \"q\" to quit")
	if m.typing {
		footer= inputStyle.Render("M-x: ") + m.ti.View()
	} else if m.message!=""{
		if m.isError {
			footer= errorStyle.Render(" ! " + m.message)
		} else{
			footer= infoStyle.Render(" i " + m.message)
		}
	}
	rows= append(rows, footer)
	// Send the UI for rendering
	return tea.View{
		Content: lipgloss.JoinVertical(lipgloss.Left, rows... ),
		AltScreen: true,
	}
}

func main() {
	args:= os.Args
	currentPath,err:=os.Getwd()
	if err!=nil{fmt.Println("failed to get position:", err)}
	if len(args)>1{
		currentPath= args[1]
	}
	f, err:= tea.LogToFile("debug.log", "debug")
	if err!= nil{
		log.Fatalf("err: %e", err)
	}
	defer f.Close()
	p := tea.NewProgram(
		initialModel(currentPath),
	)
        if _, err = p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		log.Fatal(err)
                os.Exit(1)
	}
}
