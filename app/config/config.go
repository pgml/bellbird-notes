package config

import (
	_ "embed"
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

//go:embed default.conf
var defaultConf []byte

type Section int

const (
	General Section = iota
	Theme
	Folders
	NotesList
	Editor
	BreadCrumb
)

// Map of Section enum values to their string representations
var sections = map[Section]string{
	General:    "General",
	Theme:      "Theme",
	Folders:    "Folders",
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
	LastDirectory
	CurrentComponent
	Visible
	Width
	CursorPosition
	Pinned
	Expanded
	LineNumbers
	NerdFonts
	Border
	SearchIgnoreCase
	OpenNewNote
)

// Map of Option enum values to their string names as used in the ini file
var options = map[Option]string{
	NotesDirectory:   "NotesDirectory",
	LastNotes:        "LastNotes",
	LastOpenNote:     "LastOpenNote",
	LastDirectory:    "LastDirectory",
	CurrentComponent: "CurrentComponent",
	Visible:          "Visible",
	Width:            "Width",
	CursorPosition:   "CursorPosition",
	Pinned:           "Pinned",
	Expanded:         "Expanded",
	LineNumbers:      "LineNumbers",
	NerdFonts:        "NerdFonts",
	Border:           "Border",
	SearchIgnoreCase: "SearchIgnoreCase",
	OpenNewNote:      "OpenNewNote",
}

// String returns the string representation of an Option
func (o Option) String() string {
	return options[o]
}

// Value represents an entry in the metadata file
type Value struct {
	//Section string
	//Option  Option
	Value string
}

func (v Value) GetBool() bool {
	return v.Value == "true"
}

// Config holds all config data
type Config struct {
	// path to the main config file
	filePath string

	// path to the meta data config file
	metaFilePath string

	// parsed default config file
	file *ini.File

	// parsed user config file
	userFile *ini.File

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

func (c *Config) File() string { return c.filePath }

// New loads or create a config file with default settings
func New() *Config {
	config := &Config{}

	filePath, err := app.ConfigFile(false)
	if err != nil {
		return config
	}

	metaFilePath, err := app.ConfigFile(true)
	if err != nil {
		debug.LogErr(err)
		return nil
	}

	if _, err := os.Stat(filePath); err != nil {
		_, err := utils.CreateFile(filePath, false)
		if err != nil {
			debug.LogErr(err)
		}
	}

	ini.PrettyFormat = false
	ini.PrettyEqual = true

	conf, err := ini.Load(defaultConf)
	if err != nil {
		debug.LogErr("Failed to read config file:", err)
		return nil
	}

	userConf, err := ini.Load(filePath)
	if err != nil {
		debug.LogErr("Failed to read user config file:", err)
		return nil
	}

	if _, err := os.Stat(metaFilePath); err != nil {
		utils.CreateFile(metaFilePath, false)
	}

	metaConf, err := ini.Load(metaFilePath)
	if err != nil {
		debug.LogErr("Failed to read meta infos file:", err)
		return nil
	}

	return &Config{
		filePath:     filePath,
		metaFilePath: metaFilePath,
		file:         conf,
		userFile:     userConf,
		metaFile:     metaConf,
		flushDelay:   400 * time.Millisecond,
	}
}

func (c *Config) Reload() {
	conf, err := ini.Load(c.filePath)
	if err != nil {
		debug.LogErr("Failed to read config file:", err)
	}

	c.userFile = conf
}

// Value retrieves the value of a configuration option in a given section.
func (c *Config) Value(section Section, option Option) (Value, error) {
	if sect := c.userFile.Section(section.String()); sect != nil {
		if opt := sect.Key(option.String()); opt.String() != "" {
			return Value{opt.String()}, nil
		}
	}

	sect := c.file.Section(section.String())

	if sect == nil {
		return Value{}, fmt.Errorf("No section: %s", section.String())
	}

	if opt := sect.Key(option.String()); opt.String() != "" {
		return Value{opt.String()}, nil
	} else {
		return Value{}, fmt.Errorf(
			"couldn't find config option `%s` in section `%s`",
			option.String(),
			section.String(),
		)
	}
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
	if err == nil && nf.Value != "" {
		nerdFonts = nf.GetBool()
	}

	// overwrite if cli flag is found
	if app.IsFlagPassed("no-nerd-fonts") {
		nerdFonts = false
	}

	c.nerdFonts = &nerdFonts
	return nerdFonts
}

// CleanMetaFile attempts to remove orphaned sections from the meta files.
// E.g. notes that were deleted
func (c *Config) CleanMetaFile() {
	go func() {
		for _, section := range c.metaFile.Sections() {
			// Skip general, non-note or non-directory related info
			if section.Name() == ini.DefaultSection {
				continue
			}

			if _, err := os.Stat(section.Name()); err != nil {
				// fake index set to 0 to avoid AllowNonUniqueSections option
				// since the note was deleted we're sure that this sections
				// isn't needed anymore
				err := c.metaFile.DeleteSectionWithIndex(section.Name(), 0)

				if err != nil {
					debug.LogDebug(err)
				}
			}
		}

		// write out changes
		c.metaFile.SaveTo(c.metaFilePath)
	}()
}
