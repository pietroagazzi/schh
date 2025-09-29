package main

import (
    "errors"
    "fmt"
    "os"
    "strings"

    "schh/internal/config"
    "schh/internal/session"
    "schh/internal/ui"
)

func main() {
    os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
    if len(args) == 0 {
        printUsage()
        return 1
    }

    if args[0] == "host" {
        return runHostCommand(args[1:])
    }

    flagList := false
    flagLast := false
    var hostName string
    var sessionArg string

    if args[0] == "--list" || args[0] == "--last" {
        if len(args) != 2 {
            fmt.Fprintln(os.Stderr, "Invalid arguments.")
            printUsage()
            return 1
        }
        flagList = args[0] == "--list"
        flagLast = args[0] == "--last"
        hostName = args[1]
    } else {
        hostName = args[0]
        for _, arg := range args[1:] {
            switch arg {
            case "--list":
                flagList = true
            case "--last":
                flagLast = true
            default:
                if sessionArg == "" {
                    sessionArg = arg
                } else {
                    fmt.Fprintln(os.Stderr, "Too many positional arguments.")
                    printUsage()
                    return 1
                }
            }
        }
    }

    if flagList && flagLast {
        fmt.Fprintln(os.Stderr, "--list and --last cannot be combined.")
        return 1
    }
    if flagList && sessionArg != "" {
        fmt.Fprintln(os.Stderr, "--list cannot be combined with a session name.")
        return 1
    }
    if flagLast && sessionArg != "" {
        fmt.Fprintln(os.Stderr, "--last cannot be combined with a session name.")
        return 1
    }

    hosts, err := config.LoadHosts()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load hosts: %v\n", err)
        return 1
    }
    host := config.FindHost(hosts, hostName)
    if host == nil {
        fmt.Fprintf(os.Stderr, "Host '%s' is not configured. Use 'schh host add %s [target]'.\n", hostName, hostName)
        return 1
    }

    if flagList {
        return listSessionsForHost(*host)
    }

    if flagLast {
        return attachLastSession(*host)
    }

    if sessionArg != "" {
        return runNamedSession(*host, sessionArg)
    }

    return runInteractive(*host)
}

func printUsage() {
    fmt.Fprintf(os.Stderr, "Usage:\n")
    fmt.Fprintf(os.Stderr, "  schh host add <name> [target]\n")
    fmt.Fprintf(os.Stderr, "  schh host remove <name>\n")
    fmt.Fprintf(os.Stderr, "  schh host list\n")
    fmt.Fprintf(os.Stderr, "  schh <host-name> [session-name] [--list|--last]\n")
    fmt.Fprintf(os.Stderr, "  schh --list <host-name>\n")
    fmt.Fprintf(os.Stderr, "  schh --last <host-name>\n")
}

func runHostCommand(args []string) int {
    if len(args) == 0 {
        printUsage()
        return 1
    }

    switch args[0] {
    case "add":
        if len(args) < 2 {
            fmt.Fprintln(os.Stderr, "Please provide the host name to add.")
            printUsage()
            return 1
        }
        name := args[1]
        target := name
        if len(args) >= 3 {
            target = args[2]
        }
        if containsWhitespace(name) || containsWhitespace(target) {
            fmt.Fprintln(os.Stderr, "Host names and targets cannot contain spaces.")
            return 1
        }
        if err := config.AddHost(name, target); err != nil {
            if errors.Is(err, config.ErrHostExists) {
                fmt.Fprintf(os.Stderr, "Host '%s' already exists.\n", name)
                return 1
            }
            fmt.Fprintf(os.Stderr, "Unable to save host '%s': %v\n", name, err)
            return 1
        }
        fmt.Printf("Host '%s' saved as '%s'.\n", target, name)
        return 0
    case "remove":
        if len(args) < 2 {
            fmt.Fprintln(os.Stderr, "Please provide the host name to remove.")
            printUsage()
            return 1
        }
        name := args[1]
        if err := config.RemoveHost(name); err != nil {
            if errors.Is(err, config.ErrHostNotFound) {
                fmt.Fprintf(os.Stderr, "Host '%s' was not found.\n", name)
                return 1
            }
            fmt.Fprintf(os.Stderr, "Unable to remove host '%s': %v\n", name, err)
            return 1
        }
        cleared, err := config.ClearLastSessionLabel(name)
        if err != nil {
            fmt.Printf("Host '%s' removed (unable to clear recent sessions).\n", name)
            return 0
        }
        if cleared {
            fmt.Printf("Host '%s' removed and recent sessions cleared.\n", name)
        } else {
            fmt.Printf("Host '%s' removed.\n", name)
        }
        return 0
    case "list":
        hosts, err := config.LoadHosts()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Unable to load configured hosts: %v\n", err)
            return 1
        }
        if len(hosts) == 0 {
            fmt.Println("No hosts configured. Use 'schh host add <name> [target]'.")
            return 0
        }
        fmt.Println("Configured hosts:")
        for _, h := range hosts {
            if h.Name == h.Target {
                fmt.Printf("  - %s\n", h.Name)
            } else {
                fmt.Printf("  - %s -> %s\n", h.Name, h.Target)
            }
        }
        return 0
    default:
        fmt.Fprintln(os.Stderr, "Unknown host command.")
        printUsage()
        return 1
    }
}

func listSessionsForHost(host config.Host) int {
    sessions, err := session.ListSessionsForHost(host.Name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to read active sessions: %v\n", err)
        return 1
    }
    lastLabel, err := config.GetLastSessionLabel(host.Name)
    hasLast := err == nil

    fmt.Printf("Active sessions for %s:\n", host.Name)
    if len(sessions) == 0 {
        fmt.Println("  (none)")
        return 0
    }
    for _, s := range sessions {
        marker := ""
        if hasLast && s.Label == lastLabel {
            marker = "  (last used)"
        }
        fmt.Printf("  - %s%s\n", s.Label, marker)
    }
    return 0
}

func attachLastSession(host config.Host) int {
    lastLabel, err := config.GetLastSessionLabel(host.Name)
    if err != nil {
        if errors.Is(err, config.ErrLabelNotFound) {
            fmt.Fprintf(os.Stderr, "No recent session stored for '%s'.\n", host.Name)
            return 1
        }
        fmt.Fprintf(os.Stderr, "Unable to load the recent session: %v\n", err)
        return 1
    }
    sessionID, err := session.BuildSessionID(host.Name, lastLabel)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Stored session name is no longer valid: %v\n", err)
        return 1
    }
    sessions, err := session.ListSessionsForHost(host.Name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to list active sessions: %v\n", err)
        return 1
    }
    exists := false
    for _, s := range sessions {
        if s.ID == sessionID {
            exists = true
            break
        }
    }
    if !exists {
        if err := session.StartDetachedSession(sessionID, host.Target); err != nil {
            fmt.Fprintf(os.Stderr, "Unable to start session '%s': %v\n", lastLabel, err)
            return 1
        }
    }
    if err := config.SetLastSessionLabel(host.Name, lastLabel); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: unable to update recent sessions: %v\n", err)
    }
    if err := session.AttachSession(sessionID); err != nil {
        fmt.Fprintf(os.Stderr, "Unable to attach to session: %v\n", err)
        return 1
    }
    return 0
}

func runNamedSession(host config.Host, sessionArg string) int {
    label := session.SanitizeToken(sessionArg)
    if label == "" {
        fmt.Fprintln(os.Stderr, "Invalid session name.")
        return 1
    }
    sessionID, err := session.BuildSessionID(host.Name, sessionArg)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Invalid session name: %v\n", err)
        return 1
    }
    sessions, err := session.ListSessionsForHost(host.Name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to read active sessions: %v\n", err)
        return 1
    }
    exists := false
    for _, s := range sessions {
        if s.ID == sessionID {
            exists = true
            break
        }
    }
    if !exists {
        if err := session.StartDetachedSession(sessionID, host.Target); err != nil {
            fmt.Fprintf(os.Stderr, "Unable to start session '%s': %v\n", label, err)
            return 1
        }
    }
    if err := config.SetLastSessionLabel(host.Name, label); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: unable to update recent sessions: %v\n", err)
    }
    if err := session.AttachSession(sessionID); err != nil {
        fmt.Fprintf(os.Stderr, "Unable to attach to session: %v\n", err)
        return 1
    }
    return 0
}

func runInteractive(host config.Host) int {
    sessions, err := session.ListSessionsForHost(host.Name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to read active sessions: %v\n", err)
        return 1
    }

    choice, err := ui.ChooseSession(host.Name, sessions, os.Stdin, os.Stdout)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to prompt for sessions: %v\n", err)
        return 1
    }
    switch choice.Action {
    case ui.ActionCancel:
        return 0
    case ui.ActionAttach:
        label := session.SanitizeToken(choice.Label)
        if label == "" {
            label = choice.Label
        }
        if err := config.SetLastSessionLabel(host.Name, label); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: unable to update recent sessions: %v\n", err)
        }
        if err := session.AttachSession(choice.SessionID); err != nil {
            fmt.Fprintf(os.Stderr, "Unable to attach to session: %v\n", err)
            return 1
        }
        return 0
    case ui.ActionCreate:
        label := session.SanitizeToken(choice.Label)
        if label == "" {
            fmt.Fprintln(os.Stderr, "Invalid session name.")
            return 1
        }
        sessionID, err := session.BuildSessionID(host.Name, label)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Invalid session name: %v\n", err)
            return 1
        }
        if err := session.StartDetachedSession(sessionID, host.Target); err != nil {
            fmt.Fprintf(os.Stderr, "Unable to start session '%s': %v\n", label, err)
            return 1
        }
        if err := config.SetLastSessionLabel(host.Name, label); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: unable to update recent sessions: %v\n", err)
        }
        if err := session.AttachSession(sessionID); err != nil {
            fmt.Fprintf(os.Stderr, "Unable to attach to session: %v\n", err)
            return 1
        }
        return 0
    default:
        fmt.Fprintln(os.Stderr, "Unknown selection.")
        return 1
    }
}

func containsWhitespace(text string) bool {
    return strings.ContainsAny(text, " \t")
}
