package config

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"bellbird-notes/app"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"

	"gopkg.in/ini.v1"
)

type Section int

const (
	General Section = iota
	SideBar
	NotesList
	Editor
	BreadCrumb
)

// Map of Section enum values to their string representations
var sections = map[Section]string{
	General:    "General",
	SideBar:    "Sidebar",
	NotesList:  "NotesList",
	Editor:     "Editor",
	BreadCrumb: "Breadcrumb",
}

// String returns the string representation of a Section
func (s Section) String() string {
	return sections[s]
}

type Option int

const (
	NotesDirectory Option = iota
	LastNotes
	LastOpenNote
	CurrentDirectory
	CurrentComponent
	Visible
	Width
	CursorPosition
	Pinned
	Expanded
	ShowLineNumbers
	NerdFonts
)

// Map of Option enum values to their string names as used in the ini file
var options = map[Option]string{
	NotesDirectory:   "NotesDirectory",
	LastNotes:        "LastNotes",
	LastOpenNote:     "LastOpenNote",
	CurrentDirectory: "CurrentDirectory",
	CurrentComponent: "CurrentComponent",
	Visible:          "Visible",
	Width:            "Width",
	CursorPosition:   "CursorPosition",
	Pinned:           "Pinned",
	Expanded:         "Expanded",
	ShowLineNumbers:  "ShowLineNumbers",
	NerdFonts:        "NerdFonts",
}

// String returns the string representation of an Option
func (o Option) String() string {
	return options[o]
}

// MetaValue represents an entry in the metadata file
type MetaValue struct {
	Section string
	Option  Option
	Value   string
}

// Config holds all config data
type Config struct {
	// path to the main config file
	filePath string

	// path to the meta data config file
	metaFilePath string

	// parsed main config file
	file *ini.File

	// parse meta data file
	metaFile *ini.File

	// timer used to debounce saving meta config changes
	flushTimer *time.Timer

	// mutex to synchronise flush operations
	flushMu sync.Mutex

	// delay before flushing changes to disk
	flushDelay time.Duration

	// cached nerdFonts config value
	nerdFonts *bool
}

// New loads or create a config file with default settings
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
		utils.CreateFile(filePath, false)
	}

	ini.PrettyFormat = false
	ini.PrettyEqual = true

	conf, err := ini.Load(filePath)
	if err != nil {
		debug.LogErr("Failed to read config file:", err)
		return config
	}

	if _, err := os.Stat(metaFilePath); err != nil {
		utils.CreateFile(metaFilePath, false)
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
		flushDelay:   400 * time.Millisecond,
	}
}

// SetDefaults sets default config values if none are present
func (c *Config) SetDefaults() {
	if n, err := c.Value(General, NotesDirectory); err == nil && n == "" {
		notesRootDir, _ := app.NotesRootDir()
		c.SetValue(General, NotesDirectory, notesRootDir)
	}

	if n, err := c.MetaValue("", LastNotes); err == nil && n == "" {
		c.SetMetaValue("", LastNotes, "")
	}

	if n, err := c.MetaValue("", LastOpenNote); err == nil && n == "" {
		c.SetMetaValue("", LastOpenNote, "")
	}
}

// Value retrieves the value of a configuration option in a given section.
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

// MetaValue retrieves a metadata value by a section and option.
func (c *Config) MetaValue(section string, option Option) (string, error) {
	if c.file == nil {
		return "", errors.New("could not find config file")
	}

	sect := c.file.Section(section)

	if sect == nil {
		return "", fmt.Errorf("could not find config section: %s", section)
	}

	opt := c.file.Section(section).Key(option.String())

	if opt == nil {
		return "", fmt.Errorf(
			"could not find config option `%s` in section `%s`",
			option,
			section,
		)
	}

	return c.metaFile.Section(section).Key(option.String()).String(), nil
}

// SetValue sets a configuration option value in the specified section
// and saves the config file immediately
func (c *Config) SetValue(section Section, option Option, value string) {
	c.file.
		Section(section.String()).
		Key(option.String()).
		SetValue(value)

	c.file.SaveTo(c.filePath)
}

// SetMetaValue sets a metadata option value and schedules
// saving changes with debounce
func (c *Config) SetMetaValue(path string, option Option, value string) {
	sect := c.metaFile.Section(path)
	opt := sect.Key(option.String())

	if opt.Value() == value {
		return
	}

	opt.SetValue(value)

	c.debounceFlush()
}

// RenameMetaSection renames a section in the metadata file.
func (c *Config) RenameMetaSection(oldName string, newName string) error {
	if c.metaFile == nil {
		return errors.New("could not find config file")
	}

	oldSection, err := c.metaFile.GetSection(oldName)

	if err != nil {
		return err
	}

	newSection, err := c.metaFile.NewSection(newName)

	if err != nil {
		return err
	}

	for _, key := range oldSection.Keys() {
		newSection.Key(key.Name()).SetValue(key.Value())
	}

	c.metaFile.DeleteSection(oldName)
	err = c.metaFile.SaveTo(c.metaFilePath)

	if err != nil {
		return err
	}

	return nil
}

// debounceFlush uses a timer and mutex to delay and
// batch saving of metaFile changes
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

// NerdFonts determines whether nerd fonts are enabled either
// via the config file or the cli argument.
// The cli argument always overrides value set in the config
func (c *Config) NerdFonts() bool {
	if c.nerdFonts != nil {
		return *c.nerdFonts
	}

	nf, err := c.Value(General, NerdFonts)

	// default is true
	nerdFonts := true

	// if setting is found in config file use it
	if err == nil && nf != "" {
		nerdFonts = nf == "true"
	}

	// overwrite if cli flag is found
	if app.IsFlagPassed("no-nerd-fonts") {
		nerdFonts = false
	}

	c.nerdFonts = &nerdFonts
	return nerdFonts
}
