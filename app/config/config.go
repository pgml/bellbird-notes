package config

import (
	"os"

	"bellbird-notes/app"

	"gopkg.in/ini.v1"
)

type Section int

const (
	General Section = iota
	SideBar
	NotesList
	BreadCrumb
)

var sections = map[Section]string{
	General:    "General",
	SideBar:    "Sidebar",
	NotesList:  "NotesList",
	BreadCrumb: "Breadcrumb",
}

func (s Section) String() string {
	return sections[s]
}

type Option int

const (
	DefaultNotesDirectory Option = iota
	UserNotesDirectory
	DefaultFontSize
	CurrentDirectory
	CurrentNote
	OpenNotes
	Visible
	Width
	CaretPosition
	Pinned
)

var options = map[Option]string{
	DefaultNotesDirectory: "DefaultNotesDirectory",
	UserNotesDirectory:    "UserNotesDirectory",
	DefaultFontSize:       "DefaultFontSize",
	CurrentDirectory:      "CurrentDirectory",
	CurrentNote:           "CurrentNote",
	OpenNotes:             "OpenNotes",
	Visible:               "Visible",
	Width:                 "Width",
	CaretPosition:         "CaretPosition",
	Pinned:                "Pinned",
}

func (o Option) String() string {
	return options[o]
}

type Config struct {
	filePath     string
	metaFilePath string
	file         *ini.File
	metaFile     *ini.File
}

func New() *Config {
	config := &Config{}

	filePath, err := app.ConfigFile()
	if err != nil {
		app.LogErr(err)
		return config
	}

	metaFilePath, err := app.ConfigFile()
	if err != nil {
		app.LogErr(err)
		return config
	}

	if _, err := os.Stat(filePath); err != nil {
		createFile(filePath)
	}

	conf, err := ini.Load(filePath)
	if err != nil {
		app.LogErr("Failed to read config file:", err)
		return config
	}

	if _, err := os.Stat(metaFilePath); err != nil {
		createFile(metaFilePath)
	}

	metaConf, err := ini.Load(metaFilePath)
	if err != nil {
		app.LogErr("Failed to read meta infos file:", err)
		return config
	}

	config.filePath = filePath
	config.metaFilePath = metaFilePath
	config.file = conf
	config.metaFile = metaConf
	config.SetDefaults()

	return config
}

func createFile(path string) (bool, error) {
	f, _ := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	defer f.Close()
	return true, nil
}

func (c *Config) SetDefaults() {
	notesRootDir, _ := app.NotesRootDir()
	c.SetValue(General, DefaultNotesDirectory, notesRootDir)
	c.SetValue(General, UserNotesDirectory, notesRootDir)
}

func (c *Config) Value(section Section, option Option) string {
	return c.file.
		Section(section.String()).
		Key(option.String()).
		String()
}

//func (c *Config) MetaValue(section Section, option Option) string {
//	return c.metaFile.
//		Section(section.String()).
//		Key(option.String()).
//		String()
//}

func (c *Config) SetValue(section Section, option Option, value string) {
	c.file.
		Section(section.String()).
		Key(option.String()).
		SetValue(value)

	c.file.SaveTo(c.filePath)
}

//func (c *Config) SetMetaValue(section Section, option Option, value string) {
//	c.file.
//		Section(section.String()).
//		Key(option.String()).
//		SetValue(value)
//
//	c.file.SaveTo(c.metaFilePath)
//}
