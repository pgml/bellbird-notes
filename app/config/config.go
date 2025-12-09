package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Notes
	Editor
	BreadCrumb
)

// Map of Section enum values to their string representations
var sections = map[Section]string{
	General:    "General",
	Theme:      "Theme",
	Folders:    "Folders",
	Notes:      "Notes",
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
	AutoOpenNewNote
	IndentLines
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
	AutoOpenNewNote:  "AutoOpenNewNote",
	IndentLines:      "IndentLines",
}

// String returns the string representation of an Option
func (opt Option) String() string {
	return options[opt]
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

func (conf *Config) File() string { return conf.filePath }

// New loads or create a config file with default settings
func New() *Config {
	config := &Config{}

	// dirty hack to get the notes directory created
	// @todo: make not hacky
	app.NotesRootDir()

	// config file
	filePath, err := app.ConfigFile(false)
	if err != nil {
		return config
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
		fmt.Printf("Failed to read config file: %s\n", err)
		os.Exit(2)
	}

	userConf, err := ini.Load(filePath)
	if err != nil {
		debug.LogErr("Failed to read user config file:", err)
		fmt.Printf("Failed to read config file: %s\n", err)
		os.Exit(2)
	}

	config.filePath = filePath
	config.file = conf
	config.userFile = userConf
	config.flushDelay = 400 * time.Millisecond

	// Meta info file
	metaFilePath, err := config.MetaFile()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	metaConf, err := ini.Load(metaFilePath)
	if err != nil {
		fmt.Printf("Failed to read meta infos file: %s\n", err)
		os.Exit(2)
	}

	config.metaFilePath = metaFilePath
	config.metaFile = metaConf

	return config
}

// Reload refreshes the current configuration file in memory
func (c *Config) Reload() {
	conf, err := ini.Load(c.filePath)
	if err != nil {
		debug.LogErr("Failed to read config file:", err)
	}

	c.userFile = conf
}

// Value retrieves the value of a configuration option in a given section.
func (conf *Config) Value(section Section, option Option) (Value, error) {
	if sect := conf.userFile.Section(section.String()); sect != nil {
		if opt := sect.Key(option.String()); opt.String() != "" {
			return Value{opt.String()}, nil
		}
	}

	sect := conf.file.Section(section.String())

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
func (conf *Config) MetaValue(section string, option Option) (string, error) {
	if conf.file == nil {
		return "", errors.New("could not find config file")
	}

	sect := conf.file.Section(section)

	if sect == nil {
		return "", fmt.Errorf("could not find config section: %s", section)
	}

	opt := conf.file.Section(section).Key(option.String())

	if opt == nil {
		return "", fmt.Errorf(
			"could not find config option `%s` in section `%s`",
			option,
			section,
		)
	}

	return conf.metaFile.Section(section).Key(option.String()).String(), nil
}

// SetValue sets a configuration option value in the specified section
// and saves the config file immediately
func (conf *Config) SetValue(section Section, option Option, value string) {
	conf.userFile.
		Section(section.String()).
		Key(option.String()).
		SetValue(value)

	conf.userFile.SaveTo(conf.filePath)
}

// SetMetaValue sets a metadata option value and schedules
// saving changes with debounce
func (conf *Config) SetMetaValue(path string, option Option, value string) {
	sect := conf.metaFile.Section(path)
	opt := sect.Key(option.String())

	if opt.Value() == value {
		return
	}

	opt.SetValue(value)

	conf.debounceFlush()
}

// RenameMetaSection renames a section in the metadata file.
func (conf *Config) RenameMetaSection(oldName string, newName string) error {
	if conf.metaFile == nil {
		return errors.New("could not find config file")
	}

	oldSection, err := conf.metaFile.GetSection(oldName)

	if err != nil {
		return err
	}

	newSection, err := conf.metaFile.NewSection(newName)

	if err != nil {
		return err
	}

	for _, key := range oldSection.Keys() {
		newSection.Key(key.Name()).SetValue(key.Value())
	}

	conf.metaFile.DeleteSection(oldName)
	err = conf.metaFile.SaveTo(conf.metaFilePath)

	if err != nil {
		return err
	}

	return nil
}

// debounceFlush uses a timer and mutex to delay and
// batch saving of metaFile changes
func (conf *Config) debounceFlush() {
	conf.flushMu.Lock()
	defer conf.flushMu.Unlock()

	// Cancel previous timer if it exists
	if conf.flushTimer != nil {
		conf.flushTimer.Stop()
	}

	// Set up a new delayed flush
	conf.flushTimer = time.AfterFunc(conf.flushDelay, func() {
		conf.flushMu.Lock()
		defer conf.flushMu.Unlock()
		conf.metaFile.SaveTo(conf.metaFilePath)
	})
}

// NerdFonts determines whether nerd fonts are enabled either
// via the config file or the cli argument.
// The cli argument always overrides value set in the config
func (conf *Config) NerdFonts() bool {
	if conf.nerdFonts != nil {
		return *conf.nerdFonts
	}

	nf, err := conf.Value(General, NerdFonts)

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

	conf.nerdFonts = &nerdFonts
	return nerdFonts
}

// NotesDir returns a valid path to the directory of the notes
// set in the configuration file.
// If the path starts with a ~ it is replaced with the home directory.
func (conf *Config) NotesDir() (string, error) {
	notesDir, err := conf.Value(General, NotesDirectory)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(notesDir.Value, "~/") {
		homeDir, _ := os.UserHomeDir()
		notesDir.Value = filepath.Join(homeDir, notesDir.Value[2:])
	}

	return notesDir.Value, nil
}

// MetaFile returns the path to the meta info file.
// If the file does not exist it will be created.
// If it's not in the notes directory it attempts to migrate it.
func (conf *Config) MetaFile() (string, error) {
	filePath, err := app.ConfigFile(true)
	if err != nil {
		return "", nil
	}

	notesDir, err := conf.NotesDir()
	if err != nil {
		return "", err
	}

	metaFileName := "." + filepath.Base(filePath)
	newFilePath := filepath.Join(notesDir, metaFileName)

	if _, err := os.Stat(filePath); err == nil {
		if err := conf.migrateMetaFile(filePath, newFilePath); err != nil {
			errMsg := "Error: Could not migrate meta infos file.\n"
			errMsg += err.Error() + "\n\n"
			errMsg += "Please delete one of the following files:\n"
			errMsg += "  - `" + filePath + "`\n"
			errMsg += "  - `" + newFilePath + "`\n"
			return "", errors.New(errMsg)
		}
	} else {
		_, err := utils.CreateFile(newFilePath, false)
		if err != nil {
			_, err := utils.CreateFile(filePath, false)
			if err != nil {
				return "", err
			}
			// if the file couldn't be created in the user directory just
			// return the default
			return filePath, nil
		}
	}

	return newFilePath, nil
}

// CleanMetaFile attempts to remove orphaned sections from the meta files.
// E.g. notes that were deleted
func (conf *Config) CleanMetaFile() {
	go func() {
		for _, section := range conf.metaFile.Sections() {
			// Skip general, non-note or non-directory related info
			if section.Name() == ini.DefaultSection {
				continue
			}

			if _, err := os.Stat(section.Name()); err != nil {
				// fake index set to 0 to avoid AllowNonUniqueSections option
				// since the note was deleted we're sure that this sections
				// isn't needed anymore
				err := conf.metaFile.DeleteSectionWithIndex(section.Name(), 0)

				if err != nil {
					debug.LogDebug(err)
				}
			}
		}

		// write out changes
		conf.metaFile.SaveTo(conf.metaFilePath)
	}()
}

// migrateMetaFile attempts to move the meta info file from the
// config dir to the the notes directory path set in the config file.
// If no path is set nothing happens.
func (conf *Config) migrateMetaFile(oldFile string, newFile string) error {
	if _, err := os.Stat(oldFile); err != nil {
		return fmt.Errorf("%s - %s", err, oldFile)
	}

	if _, err := os.Stat(newFile); err == nil {
		return fmt.Errorf("file already exists: %s", newFile)
	}

	if err := utils.MoveFile(oldFile, newFile); err != nil {
		return err
	}

	return nil
}
