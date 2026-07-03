package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type runtimePaths struct {
	configDir string
	dataDir   string
}

func resolveRuntimePaths(args []string) runtimePaths {
	configDir := firstNonEmpty(
		argValue(args, "--config-dir", "-config-dir"),
		os.Getenv("PACKETMIND_CONFIG_DIR"),
		defaultExeConfigDir(),
		"./configs",
	)
	configDir = absPath(configDir)

	dataDir := firstNonEmpty(
		argValue(args, "--data-dir", "-data-dir"),
		os.Getenv("PACKETMIND_DATA_DIR"),
		filepath.Join(filepath.Dir(configDir), "data"),
	)

	return runtimePaths{configDir: configDir, dataDir: absPath(dataDir)}
}

func argValue(args []string, names ...string) string {
	for i, arg := range args {
		for _, name := range names {
			if arg == name && i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
			if strings.HasPrefix(arg, name+"=") {
				return strings.TrimSpace(strings.TrimPrefix(arg, name+"="))
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultExeConfigDir() string {
	if runtime.GOOS == "darwin" {
		if base, err := os.UserConfigDir(); err == nil && strings.TrimSpace(base) != "" {
			return filepath.Join(base, "PacketMind", "configs")
		}
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "configs")
}

func absPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func ensureConfigDir(configDir string) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"packetmind.json", "models.json"} {
		target := filepath.Join(configDir, name)
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := copyDefaultConfig(name, target); err != nil {
			return err
		}
	}
	return nil
}

func copyDefaultConfig(name, target string) error {
	for _, sourceDir := range defaultConfigSourceDirs() {
		source := filepath.Join(sourceDir, name)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		return copyFile(source, target)
	}
	return fmt.Errorf("default config %s not found", name)
}

func defaultConfigSourceDirs() []string {
	dirs := make([]string, 0, 3)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs, filepath.Join(exeDir, "configs"))
		dirs = append(dirs, filepath.Clean(filepath.Join(exeDir, "..", "Resources", "configs")))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "configs"))
	}
	return dirs
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
