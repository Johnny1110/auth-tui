package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pquerna/otp/totp"
)

// AppState defines the current screen/mode of the TUI
type AppState int

const (
	StateList AppState = iota
	StateDeleting
	StateCreatingName
	StateCreatingSecret
	StateRenaming
)

const configFile = "authc-config.json"

// Account define 2FA Account
type Account struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

// model for Bubble Tea State
type model struct {
	accounts []Account
	cursor   int
	now      time.Time
	err      error
	message  string

	// State management
	state AppState

	// Input handling for creating and renaming
	textInput textinput.Model
	tempName  string // Temporarily store name while waiting for secret
}

// UI styles
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	itemStyle   = lipgloss.NewStyle().PaddingLeft(2)
	selected    = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Bold(true)
	codeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	msgStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).PaddingLeft(2).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).PaddingLeft(2).MarginTop(1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).PaddingLeft(2)
)

// tickMsg update per sec.
type tickMsg time.Time

// tick generates a message every second to update the UI
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// load JSON settings
func loadConfig(filename string) ([]Account, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		// If file doesn't exist, return empty list instead of crashing
		if os.IsNotExist(err) {
			initialAccounts := []Account{}
			err = saveConfig(filename, initialAccounts)
			if err != nil {
				return nil, fmt.Errorf("could not create initial config: %w", err)
			}

			return initialAccounts, nil
		}
		return nil, err
	}
	var accounts []Account
	err = json.Unmarshal(file, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// saveConfig writes the accounts list back to the JSON file
func saveConfig(filename string, accounts []Account) error {
	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func initialModel() model {
	accounts, err := loadConfig(configFile)

	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 30

	return model{
		accounts:  accounts,
		now:       time.Now(),
		err:       err,
		state:     StateList,
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

// Update acts as the main router for different application states
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle global tick events regardless of state
	if t, ok := msg.(tickMsg); ok {
		m.now = time.Time(t)
		return m, tick()
	}

	// Route keyboard events based on the current application state
	switch m.state {
	case StateList:
		return m.updateList(msg)
	case StateDeleting:
		return m.updateDeleting(msg)
	case StateCreatingName, StateCreatingSecret, StateRenaming:
		return m.updateInput(msg)
	}

	return m, cmd
}

// --- State Handlers ---

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.message = ""
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.message = ""
			if m.cursor < len(m.accounts)-1 {
				m.cursor++
			}
		case "c", "C":
			m.copyCurrentCode()
		case "d", "D":
			if len(m.accounts) > 0 {
				m.message = ""
				m.state = StateDeleting
			}
		case "n", "N":
			m.message = ""
			m.state = StateCreatingName
			m.textInput.Placeholder = "Enter new account name (e.g., My-GitHub)"
			m.textInput.SetValue("")
			m.textInput.Focus()
		case "r", "R":
			if len(m.accounts) > 0 {
				m.message = ""
				m.state = StateRenaming
				m.textInput.Placeholder = "Enter new name"
				m.textInput.SetValue(m.accounts[m.cursor].Name)
				m.textInput.Focus()
			}
		}
	}
	return m, nil
}

func (m model) updateDeleting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// Remove the account
			accName := m.accounts[m.cursor].Name
			m.accounts = append(m.accounts[:m.cursor], m.accounts[m.cursor+1:]...)

			// Adjust cursor if it's out of bounds after deletion
			if m.cursor >= len(m.accounts) && m.cursor > 0 {
				m.cursor--
			}

			m.saveAndRefresh(fmt.Sprintf("Deleted account: %s", accName))
		case "n", "N", "esc", "c":
			m.cancelAction()
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			m.cancelAction()
			return m, nil
		case tea.KeyEnter:
			value := strings.TrimSpace(m.textInput.Value())
			if value == "" {
				m.message = "Input cannot be empty!"
				return m, nil
			}

			// Handle state transitions based on what we are inputting
			switch m.state {
			case StateCreatingName:
				m.tempName = value
				m.state = StateCreatingSecret
				m.textInput.Placeholder = "Enter Base32 Secret Key"
				m.textInput.SetValue("")
				m.message = ""

			case StateCreatingSecret:
				// Validate secret before saving
				value = strings.ReplaceAll(strings.ToUpper(value), " ", "")
				_, err := totp.GenerateCode(value, m.now)
				if err != nil {
					m.message = "Invalid Secret Key! Please try again."
					return m, nil
				}

				m.accounts = append(m.accounts, Account{Name: m.tempName, Secret: value})
				m.saveAndRefresh(fmt.Sprintf("Added new account: %s", m.tempName))

			case StateRenaming:
				m.accounts[m.cursor].Name = value
				m.saveAndRefresh(fmt.Sprintf("Renamed to: %s", value))
			}
			return m, nil
		}
	}

	// Pass keystrokes to the text input component
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// --- Action Helpers ---

func (m *model) copyCurrentCode() {
	if len(m.accounts) == 0 {
		return
	}
	acc := m.accounts[m.cursor]
	code, err := totp.GenerateCode(acc.Secret, m.now)
	if err == nil {
		clipboard.WriteAll(code)
		m.message = fmt.Sprintf("Copied code [%s] for %s to clipboard!", code, acc.Name)
	} else {
		m.message = "Copy failed!"
	}
}

func (m *model) saveAndRefresh(successMsg string) {
	err := saveConfig(configFile, m.accounts)
	if err != nil {
		m.message = fmt.Sprintf("Failed to save config: %v", err)
	} else {
		m.message = successMsg
	}
	m.state = StateList
	m.textInput.Blur()
}

func (m *model) cancelAction() {
	m.state = StateList
	m.message = "Action canceled."
	m.textInput.Blur()
}

// --- View Rendering ---

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("read config.json failed: %v\n", m.err)
	}

	var b strings.Builder

	// Render Header
	remainingSeconds := 30 - (m.now.Second() % 30)
	header := fmt.Sprintf("==========================  <Authenticator>  [Update countdown: %02ds] ========================== \n", remainingSeconds)
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")

	// Render Main Content based on state
	if m.state == StateList || m.state == StateDeleting {
		b.WriteString(m.viewList())
	} else {
		b.WriteString(m.viewInput())
	}

	// Render Message System
	if m.message != "" {
		if strings.HasPrefix(m.message, "X") {
			b.WriteString(errorStyle.Render(m.message) + "\n")
		} else {
			b.WriteString(msgStyle.Render(m.message) + "\n")
		}
	}

	// Render Help Bar
	helpTxt := "↑/k: Up • ↓/j: Down • c: Copy • n: New • d: Delete • r: Rename • q: Quit"
	if m.state != StateList {
		helpTxt = "enter: Confirm • esc: Cancel • ctrl+c: Quit"
	}
	b.WriteString(helpStyle.Render("\n" + helpTxt))

	return b.String()
}

func (m model) viewList() string {
	if len(m.accounts) == 0 {
		return itemStyle.Render("No Accounts Exist. Press 'n' to create one.\n")
	}

	var b strings.Builder
	for i, acc := range m.accounts {
		code, err := totp.GenerateCode(acc.Secret, m.now)
		if err != nil {
			code = err.Error()
		}

		cursorStr := "  "
		lineStyle := itemStyle
		if m.cursor == i {
			cursorStr = "> "
			lineStyle = selected
		}

		leftPart := fmt.Sprintf("%s%-25s ---> ", cursorStr, acc.Name)
		styledLeft := lineStyle.Render(leftPart)
		row := fmt.Sprintf("%s [%s]", styledLeft, codeStyle.Render(code))

		// Highlight row differently if pending deletion
		if m.state == StateDeleting && m.cursor == i {
			b.WriteString(errorStyle.Render(fmt.Sprintf("%s  <-- [Delete this? (y/n)]", row)) + "\n")
		} else {
			b.WriteString(row + "\n")
		}
	}
	return b.String()
}

func (m model) viewInput() string {
	var prompt string
	switch m.state {
	case StateCreatingName:
		prompt = "Creating new account - Step 1: Name"
	case StateCreatingSecret:
		prompt = fmt.Sprintf("Creating new account [%s] - Step 2: Secret Key", m.tempName)
	case StateRenaming:
		prompt = fmt.Sprintf("Renaming [%s] - Enter new name:", m.accounts[m.cursor].Name)
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n",
		promptStyle.Render(prompt),
		m.textInput.View(),
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("exec err: %v", err)
		os.Exit(1)
	}
}
