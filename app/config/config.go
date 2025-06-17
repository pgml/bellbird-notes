package config

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

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
	NotesDirectory Option = iota
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
	NotesDirectory:   "NotesDirectory",
	CurrentDirectory: "CurrentDirectory",
	CurrentNote:      "CurrentNote",
	OpenNotes:        "OpenNotes",
	Visible:          "Visible",
	Width:            "Width",
	CursorPosition:   "CursorPosition",
	Pinned:           "Pinned",
	Expanded:         "Expanded",
}

func (o Option) String() string {
	return options[o]
}

type MetaValue struct {
	Path   string
	Option Option
	Value  string
}

type Config struct {
	filePath     string
	metaFilePath string
	file         *ini.File
	metaFile     *ini.File

	flushTimer *time.Timer
	flushMu    sync.Mutex
	flushDelay time.Duration
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

	ini.PrettyFormat = false
	ini.PrettyEqual = true

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

	return &Config{
		filePath:     filePath,
		metaFilePath: metaFilePath,
		file:         conf,
		metaFile:     metaConf,
		flushDelay:   1 * time.Second,
	}
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

func (c *Config) MetaValue(path string, option Option) (string, error) {
	if c.file == nil {
		return "", errors.New("could not find config file")
	}

	sect := c.file.Section(path)

	if sect == nil {
		return "", fmt.Errorf("could not find config section: %s", path)
	}

	opt := c.file.Section(path).Key(option.String())

	if opt == nil {
		return "", fmt.Errorf(
			"could not find config option `%s` in section `%s`",
			option,
			path,
		)
	}

	return c.metaFile.Section(path).Key(option.String()).String(), nil
}

func (c *Config) SetValue(section Section, option Option, value string) {
	c.file.
		Section(section.String()).
		Key(option.String()).
		SetValue(value)

	c.file.SaveTo(c.filePath)
}

func (c *Config) SetMetaValue(path string, option Option, value string) {
	sect := c.metaFile.Section(path)
	opt := sect.Key(option.String())

	if opt.Value() == value {
		return
	}

	opt.SetValue(value)

	c.debounceFlush()
}

func (c *Config) debounceFlush() {
	c.flushMu.Lock()
	defer c.flushMu.Unlock()

	// Cancel previous timer if it exists
	if c.flushTimer != nil {
		c.flushTimer.Stop()
	}

	// Set up a new delayed flush
	c.flushTimer = time.AfterFunc(c.flushDelay, func() {
		c.flushMu.Lock()
		defer c.flushMu.Unlock()
		c.metaFile.SaveTo(c.metaFilePath)
	})
}
