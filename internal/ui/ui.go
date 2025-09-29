package ui

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "strconv"
    "strings"

    "schh/internal/session"
)

type Action int

const (
    ActionCancel Action = iota
    ActionAttach
    ActionCreate
)

type Choice struct {
    Action    Action
    SessionID string
    Label     string
}

var ErrCanceled = errors.New("canceled by user")

func ChooseSession(hostName string, sessions []session.Info, in io.Reader, out io.Writer) (Choice, error) {
    if in == nil || out == nil {
        return Choice{Action: ActionCancel}, errors.New("input and output streams are required")
    }
    reader := bufio.NewReader(in)

    if len(sessions) == 0 {
        suggestion := session.GenerateSessionLabel()
        label, err := promptForLabel(hostName, suggestion, reader, out)
        if err != nil {
            if errors.Is(err, ErrCanceled) {
                return Choice{Action: ActionCancel}, nil
            }
            return Choice{Action: ActionCancel}, err
        }
        return Choice{Action: ActionCreate, Label: label}, nil
    }

    for {
        fmt.Fprintf(out, "\nActive sessions for %s:\n", hostName)
        for idx, info := range sessions {
            fmt.Fprintf(out, "  %d) %s\n", idx+1, info.Label)
        }
        fmt.Fprintf(out, "  %d) Start a new session\n", len(sessions)+1)
        fmt.Fprintln(out, "Type the number to select an option, or 'q' to cancel.")
        fmt.Fprint(out, "> ")

        line, err := reader.ReadString('\n')
        if err != nil {
            return Choice{Action: ActionCancel}, err
        }
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            continue
        }
        if strings.EqualFold(trimmed, "q") {
            return Choice{Action: ActionCancel}, nil
        }
        number, err := strconv.Atoi(trimmed)
        if err != nil {
            fmt.Fprintln(out, "Please enter a valid number.")
            continue
        }
        if number >= 1 && number <= len(sessions) {
            selected := sessions[number-1]
            return Choice{Action: ActionAttach, SessionID: selected.ID, Label: selected.Label}, nil
        }
        if number == len(sessions)+1 {
            suggestion := session.GenerateSessionLabel()
            label, err := promptForLabel(hostName, suggestion, reader, out)
            if err != nil {
                if errors.Is(err, ErrCanceled) {
                    continue
                }
                return Choice{Action: ActionCancel}, err
            }
            return Choice{Action: ActionCreate, Label: label}, nil
        }
        fmt.Fprintln(out, "Selection out of range. Try again.")
    }
}

func promptForLabel(hostName, suggestion string, reader *bufio.Reader, out io.Writer) (string, error) {
    fmt.Fprintf(out, "\nStarting a new session for %s.\n", hostName)
    if suggestion != "" {
        fmt.Fprintf(out, "Suggested name: %s\n", suggestion)
        fmt.Fprintln(out, "Press Enter to accept the suggestion, type a custom name, or 'q' to cancel.")
    } else {
        fmt.Fprintln(out, "Type a session name, or 'q' to cancel.")
    }
    for {
        fmt.Fprint(out, "> ")
        line, err := reader.ReadString('\n')
        if err != nil {
            return "", err
        }
        trimmed := strings.TrimSpace(line)
        if trimmed == "" {
            if suggestion != "" {
                return suggestion, nil
            }
            return "", ErrCanceled
        }
        if strings.EqualFold(trimmed, "q") {
            return "", ErrCanceled
        }
        return trimmed, nil
    }
}
