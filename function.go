package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *module) GotoFile(n int) {
	m.message="";m.isError=false
	if n>len(m.entries)-1 || n<0{
		return
	}
	if n>m.cursor{
		m.offset=min(max(len(m.entries)-m.height,0),
			max(m.offset, n+GAP-m.height))
	} else if n< m.cursor{
		m.offset=max(0, min(m.offset, n-GAP))
	}
	m.cursor=n
}

func (m *module) Open(path string) tea.Cmd{
	return func() tea.Msg{
		err:= exec.Command("xdg-open", path).Start()
		if err!=nil{return err}
		return nil
	}
}

// func GetPath (file fileitm) (string, error){
// 	if file.mode[0]=='L'{
// 		if realpath,err:=filepath.EvalSymlinks(file.path); err!=nil{
// 			return "", Myerror{message:"fialed to eval symlink, is it broken?"}
// 		} else{
// 			return realpath,nil
// 		}
		
// 	}
// 	return file.path,nil
// }

func OpenEdit(path string) tea.Cmd {
	c:= exec.Command(EDITOR, path)
	return tea.ExecProcess(c, func(err error) tea.Msg{
		if err!=nil{
			return err
		}
		return  editorMsg{}
	})
}

func OpenShell(path string) tea.Cmd{
	c:= exec.Command(SHELL)
	c.Dir=path
	return tea.ExecProcess(c, func(err error) tea.Msg{
		if err!=nil{
			return err
		}
		return  editorMsg{}
	})
}

func matchSimple(fileName, pattern string) bool{
	re, err:= regexp.Compile("(?i)"+ pattern)
	if err!=nil{return false}
	return re.MatchString(fileName)
}

func (m *module) Search(pattern string,place int, mod bool) int{
	if !mod{
		for i:= place-1; i>=0; i--{
			if matchSimple(m.entries[i].name, pattern){
				return i
			}
		}
	} else {
		end:=len(m.entries)
		for i:= place+1; i<end; i++{
			if matchSimple(m.entries[i].name, pattern){
				return i
			}
		}
	}
	return -1
}

func (m *module) ExecCommand() {
	insertCommand:= strings.Fields(m.ti.Value())
	switch insertCommand[0] {
	case "goto" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.GotoFile(n)
	case "down" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.GotoFile(n+m.cursor)
	case "up" :
		n,err:= strconv.Atoi(insertCommand[1])
		if err!=nil{m.message="Not numbers";m.isError=true;return}
		m.GotoFile(m.cursor-n)
	case "sh" :
		command:=exec.Command("sh","-c",strings.Join(insertCommand[1:], " "))
		command.Dir= m.path
		if err:=command.Run();err!=nil{
			m.message="fialed to execute";m.isError=true;return
		}
	default:
		m.message= fmt.Sprintf("Unknow command: %s", insertCommand[0])
	}
}

func (m *module) Creatf(mod int) {
	path:=filepath.Join(m.path,m.ti.Value())
	switch mod {
	case 1:
		f,err:= os.OpenFile(path,os.O_CREATE|os.O_EXCL, 0644)
		if err!=nil{
			m.isError=true;m.message=fmt.Sprintf("fialed to create: %v", err)
		} else {
			f.Close()
			m.message="file created successfully"
		}
	case 2:
		if err:=os.MkdirAll(path,0750); err!=nil{
			m.isError=true;m.message=fmt.Sprintf("fialed to create: %v", err)
		}
	case 3:
		if err:=os.Symlink(m.entries[m.cursor].path, path); err!=nil{
			m.isError=true;m.message=fmt.Sprintf("fialed to create: %v", err)
		}
	}
}

