package session

import (
    "bufio"
    "bytes"
    "errors"
    "fmt"
    "math/rand"
    "os"
    "os/exec"
    "strings"
    "sync"
    "syscall"
    "time"
)

type Info struct {
    ID    string
    Label string
}

var (
    adjectives = []string{"bold", "bright", "calm", "clever", "daring", "eager", "gentle", "lively", "nimble", "radiant", "steady", "swift", "vivid"}
    nouns      = []string{"albatross", "badger", "copper", "dolphin", "falcon", "juniper", "lynx", "maple", "otter", "pine", "raven", "spruce", "swift", "walnut"}
    rng        = rand.New(rand.NewSource(time.Now().UnixNano()))
    rngMu      sync.Mutex
)

func SanitizeToken(input string) string {
    var builder strings.Builder
    for _, r := range input {
        switch {
        case r >= 'a' && r <= 'z':
            builder.WriteRune(r)
        case r >= 'A' && r <= 'Z':
            builder.WriteRune(r + ('a' - 'A'))
        case r >= '0' && r <= '9':
            builder.WriteRune(r)
        case r == '-':
            builder.WriteRune('-')
        case r == '_':
            builder.WriteRune('_')
        case r == '.':
            builder.WriteRune('-')
        }
        if builder.Len() >= 120 {
            break
        }
    }
    return builder.String()
}

func BuildSessionID(hostName, sessionName string) (string, error) {
    hostToken := SanitizeToken(hostName)
    sessionToken := SanitizeToken(sessionName)
    if hostToken == "" || sessionToken == "" {
        return "", errors.New("invalid host or session name")
    }
    id := fmt.Sprintf("schh_%s_%s", hostToken, sessionToken)
    if len(id) >= 240 {
        return "", errors.New("session identifier too long")
    }
    return id, nil
}

func ListSessionsForHost(hostName string) ([]Info, error) {
    sanitized := SanitizeToken(hostName)
    if sanitized == "" {
        return []Info{}, nil
    }

    cmd := exec.Command("screen", "-ls")
    output, err := cmd.CombinedOutput()
    if err != nil {
        var exitErr *exec.ExitError
        if !(errors.As(err, &exitErr) && len(output) > 0) {
            return nil, err
        }
    }
    prefix := fmt.Sprintf("schh_%s_", sanitized)
    sessions, err := parseScreenOutput(output, prefix)
    if err != nil {
        return nil, err
    }
    return sessions, nil
}

func parseScreenOutput(output []byte, prefix string) ([]Info, error) {
    scanner := bufio.NewScanner(bytes.NewReader(output))
    var sessions []Info
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        fields := strings.Fields(line)
        var candidate string
        for _, field := range fields {
            if strings.Contains(field, ".") {
                candidate = field
                break
            }
        }
        if candidate == "" {
            continue
        }
        dotIdx := strings.Index(candidate, ".")
        if dotIdx < 0 || dotIdx+1 >= len(candidate) {
            continue
        }
        name := candidate[dotIdx+1:]
        if !strings.HasPrefix(name, prefix) {
            continue
        }
        label := strings.TrimPrefix(name, prefix)
        if label == "" {
            continue
        }
        sessions = append(sessions, Info{ID: candidate, Label: label})
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return sessions, nil
}

func StartDetachedSession(sessionID, target string) error {
    if sessionID == "" || target == "" {
        return errors.New("missing session identifier or target")
    }
    cmd := exec.Command("screen", "-dmS", sessionID, "ssh", target)
    return cmd.Run()
}

func AttachSession(sessionID string) error {
    if sessionID == "" {
        return errors.New("missing session identifier")
    }
    screenPath, err := exec.LookPath("screen")
    if err != nil {
        return err
    }
    return syscall.Exec(screenPath, []string{"screen", "-r", sessionID}, os.Environ())
}

func GenerateSessionLabel() string {
    rngMu.Lock()
    defer rngMu.Unlock()
    if len(adjectives) == 0 || len(nouns) == 0 {
        return "session"
    }
    adj := adjectives[rng.Intn(len(adjectives))]
    noun := nouns[rng.Intn(len(nouns))]
    number := rng.Intn(1000)
    return fmt.Sprintf("%s-%s-%03d", adj, noun, number)
}
