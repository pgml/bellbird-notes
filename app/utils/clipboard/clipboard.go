package clipboard

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"

	"golang.design/x/clipboard"
)

var (
	useWayland = false
	useX11     = false
	useNative  = false
	ready      = false
)

func Init() error {
	switch runtime.GOOS {
	case "windows", "darwin":
		useNative = true

	case "linux":
		display := os.Getenv("DISPLAY")
		wayland := os.Getenv("WAYLAND_DISPLAY")

		switch {
		case wayland != "":
			useWayland = true

		case display != "":
			useNative = true
			useX11 = true

		default:
			return errors.New("no clipboard backend detected (no DISPLAY or WAYLAND_DISPLAY)")
		}
	default:
		return errors.New("unsupported OS for clipboard")
	}

	if useNative {
		if err := clipboard.Init(); err != nil {
			return err
		}
	}

	ready = true
	return nil
}

func Write(text string) error {
	if !ready {
		return errors.New("clipboard not initialized")
	}

	if useNative {
		clipboard.Write(clipboard.FmtText, []byte(text))
		return nil
	} else if useWayland {
		cmd := exec.Command("wl-copy", "--type", "text/plain", "--foreground")
		stdin, err := cmd.StdinPipe()

		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, text)
		}()

		go func() {
			cmd.Wait()
		}()

		return nil
	}

	return errors.New("no clipboard backend available for Write")
}

func Read() (string, error) {
	if !ready {
		return "", errors.New("clipboard not initialized")
	}

	if useNative {
		data := clipboard.Read(clipboard.FmtText)
		return string(data), nil
	} else if useWayland {
		out, err := exec.Command("wl-paste", "--no-newline").Output()

		if err != nil {
			return "", err
		}

		return string(bytes.TrimSpace(out)), nil
	}

	return "", errors.New("no clipboard backend available for Read")
}
