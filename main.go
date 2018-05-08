package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func main() {
	aw := new(AppWindow)

	saveDirs, err := discoverSaveDirs()
	if err != nil {
		fmt.Printf("ERROR: could not discover save directories: %s\n", err)
		return
	}

	aw.saveDirs = saveDirs
	aw.saveDir = aw.saveDirs[0]

	saves, err := discoverSaveFiles(aw.saveDirs[0])
	if err != nil {
		fmt.Printf("ERROR: could not discover saves: %s\n", err)
		return
	}
	aw.saves = saves

	go monitor(aw)

	MainWindow{
		AssignTo: &aw.MainWindow,
		Title:    "DDDA Save Archiver",
		MinSize:  Size{600, 400},
		Layout:   VBox{},
		Children: []Widget{
			ComboBox{
				AssignTo: &aw.saveDirCB,
				Editable: false,
				Model:    aw.saveDirs,
				OnCurrentIndexChanged: func() {
					var err error
					aw.saveDir = aw.saveDirs[aw.saveDirCB.CurrentIndex()]
					aw.saves, err = discoverSaveFiles(aw.saveDir)
					if err != nil {
						walk.MsgBox(aw.MainWindow, "Error", err.Error(), walk.MsgBoxIconError)
					}
					aw.savesLB.SetModel(aw.saves)
				},
			},
			ListBox{
				AssignTo:           &aw.savesLB,
				AlwaysConsumeSpace: true,
				DataMember:         "Title",
				Model:              aw.saves,
				OnCurrentIndexChanged: func() {
					aw.selectedSave = aw.saves[aw.savesLB.CurrentIndex()]
				},
			},
			PushButton{
				Text: "Restore",
				OnClicked: func() {
					restoreSave(aw.saveDir, aw.selectedSave)
				},
			},
		},
	}.Run()
}

type AppWindow struct {
	*walk.MainWindow

	saveDirs     []string
	saveDirCB    *walk.ComboBox
	saveDir      string
	savesLB      *walk.ListBox
	saves        []SaveFile
	selectedSave SaveFile
}

type SaveFile struct {
	Name  string
	Title string
}

func newSaveFile(name string, timestamp time.Time) SaveFile {
	return SaveFile{
		Name:  name,
		Title: timestamp.Format(time.Stamp),
	}
}

func discoverSaveDirs() (saveDirs []string, err error) {
	steamPath, err := walk.RegistryKeyString(walk.CurrentUserKey(), "Software\\Valve\\Steam", "SteamPath")
	if err != nil {
		return
	}
	userDataDir, err := os.Open(filepath.Join(steamPath, "userdata"))
	if err != nil {
		return
	}
	names, err := userDataDir.Readdirnames(0)
	if err != nil {
		return
	}

	for _, name := range names {
		dir := filepath.Join(userDataDir.Name(), name, "367500", "remote")
		if _, statErr := os.Stat(filepath.Join(dir, "DDDA.sav")); statErr == nil {
			saveDirs = append(saveDirs, dir)
		}
	}

	return
}

const backupExtension = ".sav.bak"

func discoverSaveFiles(p string) (saves []SaveFile, err error) {
	names, err := filepath.Glob(filepath.Join(p, "*"+backupExtension))
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		_, base := filepath.Split(name)
		timestamp, err := extractTimestamp(base)
		if err != nil {
			return nil, err
		}
		saves = append(saves, newSaveFile(base, timestamp))
	}

	return
}

func extractTimestamp(s string) (time.Time, error) {
	s = strings.TrimSuffix(s, backupExtension)
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid backup name: %s", s)
	}
	s = parts[1]
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid backup timestamp: %s", s)
	}
	return time.Unix(n, 0), nil
}

const primarySaveFileName = "DDDA.sav"

func monitor(aw *AppWindow) {
	p := filepath.Join(aw.saveDir, primarySaveFileName)
	dir := aw.saveDir
	base := "DDDA"
	ext := ".sav"

	fmt.Printf("Monitoring %s...\n", p)
	t := time.Now()
	for {
		time.Sleep(5 * time.Second)
		i, err := os.Stat(p)
		if err != nil {
			fmt.Printf("ERROR: could not check file time: %s\n", err)
			return
		}
		if i.ModTime().Sub(t) > 0 {
			t = i.ModTime()
			name := fmt.Sprintf("%s-%d%s.bak", base, t.Unix(), ext)
			bp := filepath.Join(dir, name)
			fmt.Printf("File changed, backing up to %s\n", name)
			err := copyFile(p, bp)
			if err != nil {
				fmt.Printf("ERROR: could not backup file: %s\n", err)
				return
			}
			aw.saves = append(aw.saves, newSaveFile(name, t))
			aw.savesLB.SetModel(aw.saves)
		}
	}
}

func copyFile(srcpath, destpath string) error {
	data, err := ioutil.ReadFile(srcpath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(destpath, data, 0666)
}

func restoreSave(saveDir string, save SaveFile) {
	primary := filepath.Join(saveDir, primarySaveFileName)
	err := os.Rename(primary, primary+".orig")
	if err != nil {
		walk.MsgBox(nil, "Error", err.Error(), walk.MsgBoxIconError)
		return
	}

	savePath := filepath.Join(saveDir, save.Name)
	stat, err := os.Stat(savePath)
	if err != nil {
		walk.MsgBox(nil, "Error", err.Error(), walk.MsgBoxIconError)
		return
	}

	err = copyFile(savePath, primary)
	if err != nil {
		walk.MsgBox(nil, "Error", err.Error(), walk.MsgBoxIconError)
		return
	}

	err = os.Chtimes(primary, stat.ModTime(), stat.ModTime())
	if err != nil {
		walk.MsgBox(nil, "Warning", err.Error(), walk.MsgBoxIconWarning)
	}

	walk.MsgBox(nil, "Save Restored", fmt.Sprintf("Backup '%s' restored", save.Title), walk.MsgBoxIconInformation)
}
