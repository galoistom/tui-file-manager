package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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
	return FetchFile(m.path)
}

func FetchFile(path string) tea.Cmd {
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

func (m *module) handleCreate (msg tea.Msg) (tea.Model, tea.Cmd) {
	m.message=""
	if temp==-1{
		switch msg:= msg.(type){
		case tea.KeyMsg:temp=HandleCreateMap(msg.String())
		}
	} else {
		return m.handleTyping(msg,func(){m.Creatf(temp)})
	}
	return m,nil
}

func (m *module) handleDelete (msg tea.Msg) (tea.Model, tea.Cmd){
	m.message=""
	m.currentMode=modeNormal
	switch msg:=msg.(type){
	case tea.KeyMsg:
		switch msg.String(){
		case "c", "enter":
			os.RemoveAll(m.entries[m.cursor].path)
			m.GotoFile(0)
			return m, FetchFile(m.path)
		case "s":
			for i := range m.selected{
				os.RemoveAll(m.entries[i].path)
			}
			m.selected=make(map[int]struct{})
			m.GotoFile(0)
			return m, FetchFile(m.path)
		}
	}
	return m,nil
}


func (m *module) handleTyping (msg tea.Msg, action func()) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg:= msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+g":
			m.currentMode=modeNormal;m.ti.SetValue("")
			m.message="";m.isError=false
			return m,nil
		case "enter":
			m.message= "";m.isError= false
			action()
			m.currentMode=modeNormal;m.ti.SetValue("")
			return m,FetchFile(m.path)
		default:m.ti, cmd= m.ti.Update(msg)
		}
	}
	return m,cmd
}

func (m *module) handleSearching(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg:= msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+g":
			m.currentMode=modeNormal;m.ti.SetValue("")
			m.GotoFile(temp)
			return m, nil
		case "enter":
			m.currentMode=modeNormal
			temp=m.cursor
			m.ti.SetValue("")
		case "ctrl+s":
		        place:= m.Search(m.ti.Value(), m.cursor,true)
      		        if place==-1{m.GotoFile(temp);return m,cmd}
		        m.GotoFile(place)	
		case "ctrl+r":
		        place:= m.Search(m.ti.Value(), m.cursor,false)
			if place==-1{m.GotoFile(temp);return m,cmd}
			m.GotoFile(place)
		default:
			m.ti, cmd= m.ti.Update(msg)
			place:= m.Search(m.ti.Value(), temp,m.searching)
			if place==-1{m.GotoFile(temp);return m,cmd}
			m.GotoFile(place)
		}
	}
	return m,cmd
}

func (m module) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.currentMode{
	case modeCommand: return m.handleTyping(msg,m.ExecCommand)

	case modeSearch: return m.handleSearching(msg)

	case modeCreate: return m.handleCreate(msg)

	case modeDelete: return m.handleDelete(msg)

	case modeRename:
		m.RenameUpdate()
		m.currentMode=modeNormal
		return m,FetchFile(m.path)
	}
	switch msg := msg.(type) {
	case error: fmt.Println(msg)

	case tea.WindowSizeMsg: m.height= msg.Height - 4
	
	case itemsMsg: m.entries= msg

	case editorMsg: return m,FetchFile(m.path)

	case tea.KeyPressMsg:
		m.isError=false
                m.message=""
		// Cool, what was the actual key pressed?
		switch msg.String() {
		
		// These keys should exit the program.
		case "ctrl+c", "q": return m, tea.Quit

		case "g", "alt+shift+,": m.GotoFile(0)

		case "G", "alt+shift+.": m.GotoFile(len(m.entries)-1)

		// The "up" and "k" keys move the cursor up
		case "up", "k", "ctrl+p": m.GotoFile(m.cursor-1)
			
		// The "down" and "j" keys move the cursor down
		case "down", "j", "ctrl+n": m.GotoFile(m.cursor+1)
			
		// for the item that the cursor is pointing at.
		case "space":
			_, ok := m.selected[m.cursor]
                        if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
			m.GotoFile(m.cursor+1)

		//press enter or l to open 
		case "enter", "l":
			selected:= m.entries[m.cursor]
			switch selected.mode[0]{
			case 'd':
				m.path= selected.path
				m.cursor=0
				m.offset=0
				return m, FetchFile(m.path)
			case 'L':
				realpath,err:=filepath.EvalSymlinks(selected.path)
				if  err!=nil{
					m.message="broken link";m.isError=true
					return m, nil
				}
				info,err:=os.Stat(realpath)
				if err!=nil{
					m.message="targer not exists";m.isError=true
					return m, nil
				}
				if info.IsDir(){
					m.path=realpath
					m.cursor=0
					return m,FetchFile(m.path)
				} else {
					return m,m.Open(realpath)
				}

			default:
				return m, m.Open(selected.path)
			}

		// press e to edit file in vim
		case "e":
			selected:= m.entries[m.cursor]
			switch selected.mode[0]{
			case '-':return m, tea.Batch(OpenShell(m.path,EDITOR+" "+selected.path),FetchFile(m.path))
			case 'L':
				realpath,err:=filepath.EvalSymlinks(selected.path)
				if  err!=nil{
					m.message="broken link";m.isError=true
					return m, nil
				}
				info,err:=os.Stat(realpath)
				if err!=nil{
					m.message="targer not exists";m.isError=true
					return m, nil
				}
				if !info.IsDir(){
					return m, tea.Batch(OpenShell(m.path,EDITOR+" "+realpath),FetchFile(m.path))
				}
			}

		//rename file
		case "R":
			f,err:= os.CreateTemp("", "tui-file-manager-*")
			if err!=nil{
				m.message="fialed to creat temp file when renaming: "+err.Error();m.isError=true
				return m,nil
			}
			var index strings.Builder
			for i,ent:= range m.entries{
				fmt.Fprintf(&index,"%d %s %s\n", i, ent.mode, ent.name)
			}
			if _,err= f.WriteString(index.String()); err!=nil{
				m.message="fialed to write into temp file when renaming: "+ err.Error();m.isError=true
				return m, nil
			}
			m.tempFile=f.Name()
			f.Close()
			m.currentMode=modeRename
			return m,OpenShell(m.path, EDITOR+" "+m.tempFile)

		//delete files
		case "x":
			m.currentMode=modeDelete
			m.message="'c'urent/ 's'elect"

		//copy and paste
		case "y","alt+w":
			yank= []string{}
			for i := range m.selected{
				yank=append(yank,m.entries[i].name)
			}
			m.selected=make(map[int]struct{})
		case "p","ctrl+y":
			for _,i:= range yank{
				cmd:=exec.Command("cp", "-r", i, m.path)
				if err:=cmd.Run();err!=nil{
					return m, func() tea.Msg{return Myerror{
						err:err,
						message: "failed to pase to dictionary: ",
					}}}
			}
			return m, FetchFile(m.path)

		//create new file
		case "n":
			m.message="'f'ile/ 'd'ictionary / 's'ymlink"
			m.currentMode=modeCreate
			temp=-1

		//press alt+x or : to input command
		case "alt+x", ":":
			m.currentMode=modeCommand
			m.ti.Focus()

		//searching
		case "ctrl+s":
			m.currentMode=modeSearch
			m.searching=true
			m.ti.Focus()
			temp=m.cursor
		case "ctrl+r":
			m.currentMode=modeSearch			
			m.searching=false
			m.ti.Focus()
			temp=m.cursor

		//open shell
		case "t": return m, tea.Batch(OpenShell(m.path, SHELL), FetchFile(m.path))
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
	switch m.currentMode{
	case modeCommand:
		footer= inputStyle.Render("\n M-x: ") + m.ti.View()
	case modeSearch:
		if m.searching{
			footer= inputStyle.Render("\n C-s: ") + m.ti.View()
		} else {
			footer= inputStyle.Render("\n C-r: ") + m.ti.View()
		}
	case modeCreate:
		footer= inputStyle.Render("\n name: ") + m.ti.View()
	}
	if m.message!=""{
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
