package config

import (
    "bufio"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

type Host struct {
    Name   string
    Target string
}

var (
    ErrHostExists    = errors.New("host already exists")
    ErrHostNotFound  = errors.New("host not found")
    ErrLabelNotFound = errors.New("no saved session label")
)

func ensureConfigDir() (string, error) {
    base, err := os.UserConfigDir()
    if err != nil || base == "" {
        home, homeErr := os.UserHomeDir()
        if homeErr != nil {
            if err != nil {
                return "", err
            }
            return "", homeErr
        }
        base = filepath.Join(home, ".config")
    }
    dir := filepath.Join(base, "schh")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", err
    }
    return dir, nil
}

func hostsFilePath() (string, error) {
    dir, err := ensureConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "hosts"), nil
}

func lastSessionsFilePath() (string, error) {
    dir, err := ensureConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "last_sessions"), nil
}

func LoadHosts() ([]Host, error) {
    path, err := hostsFilePath()
    if err != nil {
        return nil, err
    }

    file, err := os.Open(path)
    if errors.Is(err, os.ErrNotExist) {
        return []Host{}, nil
    }
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var hosts []Host
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) == 0 {
            continue
        }
        name := fields[0]
        target := name
        if len(fields) > 1 {
            target = fields[1]
        }
        hosts = append(hosts, Host{Name: name, Target: target})
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return hosts, nil
}

func FindHost(hosts []Host, name string) *Host {
    for i := range hosts {
        if hosts[i].Name == name {
            return &hosts[i]
        }
    }
    return nil
}

func AddHost(name, target string) error {
    path, err := hostsFilePath()
    if err != nil {
        return err
    }
    hosts, err := LoadHosts()
    if err != nil {
        return err
    }
    if FindHost(hosts, name) != nil {
        return ErrHostExists
    }

    file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()

    if _, err := fmt.Fprintf(file, "%s %s\n", name, target); err != nil {
        return err
    }
    return nil
}

func RemoveHost(name string) error {
    path, err := hostsFilePath()
    if err != nil {
        return err
    }
    hosts, err := LoadHosts()
    if err != nil {
        return err
    }

    index := -1
    for i, h := range hosts {
        if h.Name == name {
            index = i
            break
        }
    }
    if index == -1 {
        return ErrHostNotFound
    }

    hosts = append(hosts[:index], hosts[index+1:]...)

    file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for _, h := range hosts {
        if _, err := fmt.Fprintf(writer, "%s %s\n", h.Name, h.Target); err != nil {
            return err
        }
    }
    if err := writer.Flush(); err != nil {
        return err
    }
    return nil
}

func GetLastSessionLabel(hostName string) (string, error) {
    path, err := lastSessionsFilePath()
    if err != nil {
        return "", err
    }
    file, err := os.Open(path)
    if errors.Is(err, os.ErrNotExist) {
        return "", ErrLabelNotFound
    }
    if err != nil {
        return "", err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        if fields[0] == hostName {
            return fields[1], nil
        }
    }
    if err := scanner.Err(); err != nil {
        return "", err
    }
    return "", ErrLabelNotFound
}

func SetLastSessionLabel(hostName, label string) error {
    entries, err := loadLabelEntries()
    if err != nil {
        return err
    }
    entries[hostName] = label
    return saveLabelEntries(entries)
}

func ClearLastSessionLabel(hostName string) (bool, error) {
    entries, err := loadLabelEntries()
    if err != nil {
        return false, err
    }
    if _, ok := entries[hostName]; !ok {
        return false, nil
    }
    delete(entries, hostName)
    if err := saveLabelEntries(entries); err != nil {
        return false, err
    }
    return true, nil
}

func loadLabelEntries() (map[string]string, error) {
    path, err := lastSessionsFilePath()
    if err != nil {
        return nil, err
    }
    file, err := os.Open(path)
    if errors.Is(err, os.ErrNotExist) {
        return map[string]string{}, nil
    }
    if err != nil {
        return nil, err
    }
    defer file.Close()

    entries := make(map[string]string)
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }
        entries[fields[0]] = fields[1]
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return entries, nil
}

func saveLabelEntries(entries map[string]string) error {
    path, err := lastSessionsFilePath()
    if err != nil {
        return err
    }
    file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    for host, label := range entries {
        if _, err := fmt.Fprintf(writer, "%s %s\n", host, label); err != nil {
            return err
        }
    }
    return writer.Flush()
}
