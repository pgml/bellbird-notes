package config

import (
	"errors"
	"fmt"
	"os"

	"bellbird-notes/app"
	"bellbird-notes/app/debug"

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
	CursorPosition
	Pinned
	Expanded
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
	CursorPosition:        "CursorPosition",
	Pinned:                "Pinned",
	Expanded:              "Expanded",
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

	filePath, err := app.ConfigFile(false)
	if err != nil {
		debug.LogErr(err)
		return config
	}

	metaFilePath, err := app.ConfigFile(true)
	if err != nil {
		debug.LogErr(err)
		return config
	}

	if _, err := os.Stat(filePath); err != nil {
		createFile(filePath)
	}

	conf, err := ini.Load(filePath)
	if err != nil {
		debug.LogErr("Failed to read config file:", err)
		return config
	}

	if _, err := os.Stat(metaFilePath); err != nil {
		createFile(metaFilePath)
	}

	metaConf, err := ini.Load(metaFilePath)
	if err != nil {
		debug.LogErr("Failed to read meta infos file:", err)
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

func (c *Config) Value(section Section, option Option) (string, error) {
	if c.file == nil {
		return "", errors.New("could not find config file")
	}

	sect := c.file.Section(section.String())

	if sect == nil {
		return "", fmt.Errorf("could not find config section: %s", section)
	}

	opt := c.file.Section(section.String()).Key(option.String())

	if opt == nil {
		return "", fmt.Errorf(
			"could not find config option `%s` in section `%s`",
			option,
			section,
		)
	}

	return c.file.
		Section(section.String()).
		Key(option.String()).
		String(), nil
}

func (c *Config) MetaValue(path string, option Option) string {
	return c.metaFile.
		Section(path).
		Key(option.String()).
		String()
}

func (c *Config) SetValue(section Section, option Option, value string) {
	c.file.
		Section(section.String()).
		Key(option.String()).
		SetValue(value)

	c.file.SaveTo(c.filePath)
}

func (c *Config) SetMetaValue(path string, option Option, value string) {
	c.metaFile.
		Section(path).
		Key(option.String()).
		SetValue(value)

	c.metaFile.SaveTo(c.metaFilePath)
}
