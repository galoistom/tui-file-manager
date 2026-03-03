package main

import (
	"os/exec"
	"regexp"

	tea "charm.land/bubbletea/v2"
)

func (m *module) GotoFile(n int) {
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

func Open(path string) tea.Cmd{
	return func() tea.Msg{
		err:= exec.Command("xdg-open", path).Start()
		if err!=nil{return err}
		return nil
	}
}

func OpenEdit(path string) tea.Cmd {
	c:= exec.Command("vim", path)
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
