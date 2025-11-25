package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Project struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Tag         string    `json:"tag"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	// Color palette
	primaryColor   = lipgloss.Color("#7C3AED")
	accentColor    = lipgloss.Color("#EC4899")
	successColor   = lipgloss.Color("#10B981")
	mutedColor     = lipgloss.Color("#6B7280")
	brightColor    = lipgloss.Color("#F59E0B")
	bgColor        = lipgloss.Color("#1F2937")
	highlightColor = lipgloss.Color("#3B82F6")
	warningColor   = lipgloss.Color("#F59E0B")

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1).
			MarginTop(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Container styles
	leftPanelStyle = lipgloss.NewStyle().
			Width(45).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	rightPanelStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)

	// List item styles
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(brightColor).
				Bold(true).
				PaddingLeft(1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			PaddingLeft(3)

	tagStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	pathStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Help and status styles
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			MarginLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginLeft(2)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Italic(true).
			MarginTop(0)

	// Form styles
	formTitleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true).
			MarginBottom(0).
			MarginTop(1)

	focusedButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor).
				Padding(0, 3).
				Bold(true).
				MarginTop(1)

	blurredButtonStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 3).
				MarginTop(1)

	// Detail view styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB"))

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1).
			MarginBottom(1)
)

type editorFinishedMsg struct{ err error }

type viewMode int

const (
	viewList viewMode = iota
	viewAdd
)

type model struct {
	projects         []Project
	filteredIdxs     []int
	cursor           int
	viewport         viewport.Model
	textInput        textinput.Model
	filterMode       bool
	leftWidth        int
	ready            bool
	projectsFile     string
	statusMessage    string
	isError          bool
	mode             viewMode
	addInputs        []textinput.Model
	addFocusIndex    int
	pathValidation   string   // Validation message for path field
	autocompleteOpts []string // Autocomplete options
}

func initialModel() model {
	home, _ := os.UserHomeDir()
	projectsFile := filepath.Join(home, ".config", "projects", "projects.json")
	os.MkdirAll(filepath.Dir(projectsFile), 0o755)

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primaryColor)
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	ti.CharLimit = 156
	ti.Width = 38

	vp := viewport.New(20, 10)
	vp.SetContent("")

	inputs := make([]textinput.Model, 4)
	inputStyles := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	promptStyle := lipgloss.NewStyle().Foreground(primaryColor)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "My Awesome Project"
	inputs[0].PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	inputs[0].PromptStyle = promptStyle
	inputs[0].TextStyle = inputStyles
	inputs[0].CharLimit = 100
	inputs[0].Width = 60

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "/home/user/projects/awesome-project"
	inputs[1].PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	inputs[1].PromptStyle = promptStyle
	inputs[1].TextStyle = inputStyles
	inputs[1].CharLimit = 500
	inputs[1].Width = 60

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "go, rust, python"
	inputs[2].PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	inputs[2].PromptStyle = promptStyle
	inputs[2].TextStyle = inputStyles
	inputs[2].CharLimit = 50
	inputs[2].Width = 60

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "A brief description of your project"
	inputs[3].PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	inputs[3].PromptStyle = promptStyle
	inputs[3].TextStyle = inputStyles
	inputs[3].CharLimit = 500
	inputs[3].Width = 60

	m := model{
		projectsFile: projectsFile,
		leftWidth:    45,
		viewport:     vp,
		textInput:    ti,
		mode:         viewList,
		addInputs:    inputs,
	}

	if err := m.loadProjects(); err != nil {
		m.statusMessage = fmt.Sprintf("Error loading projects: %v", err)
		m.isError = true
	}
	m.applyFilter("")

	return m
}

func (m *model) loadProjects() error {
	data, err := os.ReadFile(m.projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			m.projects = []Project{}
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &m.projects); err != nil {
		return err
	}

	sort.Slice(m.projects, func(i, j int) bool {
		return m.projects[i].UpdatedAt.After(m.projects[j].UpdatedAt)
	})

	return nil
}

func (m *model) saveProjects() error {
	data, err := json.MarshalIndent(m.projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.projectsFile, data, 0o644)
}

func (m *model) addProject(p Project) error {
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.projects = append([]Project{p}, m.projects...)
	return m.saveProjects()
}

func (m *model) deleteProject(idx int) error {
	if idx < 0 || idx >= len(m.projects) {
		return fmt.Errorf("invalid index")
	}
	m.projects = append(m.projects[:idx], m.projects[idx+1:]...)
	return m.saveProjects()
}

func (m *model) applyFilter(q string) {
	q = strings.TrimSpace(strings.ToLower(q))
	m.filteredIdxs = m.filteredIdxs[:0]

	for i, p := range m.projects {
		if q == "" ||
			strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Tag), q) ||
			strings.Contains(strings.ToLower(p.Description), q) ||
			strings.Contains(strings.ToLower(p.Path), q) {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
	}

	if len(m.filteredIdxs) == 0 {
		m.cursor = 0
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("No projects match your filter"))
		return
	}
	if m.cursor >= len(m.filteredIdxs) {
		m.cursor = 0
	}
	m.loadSelectedToViewport()
}

func (m *model) loadSelectedToViewport() {
	if len(m.filteredIdxs) == 0 {
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("No projects available"))
		return
	}
	idx := m.filteredIdxs[m.cursor]
	p := m.projects[idx]

	var content strings.Builder

	content.WriteString(detailLabelStyle.Render("Project Name") + "\n")
	content.WriteString(detailValueStyle.Render(p.Name) + "\n")
	content.WriteString(dividerStyle.Render("─────────────────────────────────") + "\n")

	if p.Tag != "" {
		content.WriteString(detailLabelStyle.Render("Tag") + "\n")
		content.WriteString(tagStyle.Render("# "+p.Tag) + "\n")
		content.WriteString(dividerStyle.Render("─────────────────────────────────") + "\n")
	}

	content.WriteString(detailLabelStyle.Render("Path") + "\n")
	content.WriteString(pathStyle.Render(p.Path) + "\n")
	content.WriteString(dividerStyle.Render("─────────────────────────────────") + "\n")

	if p.Description != "" {
		content.WriteString(detailLabelStyle.Render("Description") + "\n")
		content.WriteString(detailValueStyle.Render(p.Description) + "\n")
		content.WriteString(dividerStyle.Render("─────────────────────────────────") + "\n")
	}

	content.WriteString(detailLabelStyle.Render("Timeline") + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(
		fmt.Sprintf("Created:  %s\nModified: %s",
			p.CreatedAt.Format("Jan 02, 2006 15:04"),
			p.UpdatedAt.Format("Jan 02, 2006 15:04"))))

	m.viewport.SetContent(content.String())
}

// validatePath checks if the path exists and returns an error message if not
func validatePath(path string) string {
	if path == "" {
		return ""
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "⚠ Path does not exist"
		}
		return fmt.Sprintf("⚠ Error: %v", err)
	}

	// Optionally check if it's a directory
	if !info.IsDir() {
		return "⚠ Path is not a directory"
	}

	return "" // Valid path
}

// expandPath expands ~ to home directory and makes path absolute
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// autocomplete performs path autocompletion similar to bash tab completion
func autocomplete(currentPath string) (string, []string) {
	if currentPath == "" {
		currentPath = "."
	}

	// Expand ~ to home directory
	if strings.HasPrefix(currentPath, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			currentPath = filepath.Join(home, currentPath[1:])
		}
	}

	// Split into directory and prefix
	dir := filepath.Dir(currentPath)
	prefix := filepath.Base(currentPath)

	// If the path ends with /, we're looking for completions in that directory
	if strings.HasSuffix(currentPath, string(filepath.Separator)) {
		dir = currentPath
		prefix = ""
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return currentPath, nil
	}

	// Find matching entries
	var matches []string
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless prefix starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		// Check if entry matches prefix
		if strings.HasPrefix(name, prefix) {
			fullPath := filepath.Join(dir, name)
			if entry.IsDir() {
				fullPath += string(filepath.Separator)
			}
			matches = append(matches, fullPath)
		}
	}

	// Sort matches
	sort.Strings(matches)

	// If exactly one match, return it
	if len(matches) == 1 {
		return matches[0], nil
	}

	// If multiple matches, find common prefix
	if len(matches) > 1 {
		common := longestCommonPrefix(matches)
		if common != currentPath && common != "" {
			return common, matches
		}
		return currentPath, matches
	}

	return currentPath, nil
}

// longestCommonPrefix finds the longest common prefix among strings
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
			m.isError = true
		} else {
			m.statusMessage = "✓ Returned from editor"
			m.isError = false
		}
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		if m.mode == viewAdd {
			// Handle tab completion for path field
			if k == "tab" && m.addFocusIndex == 1 {
				currentPath := m.addInputs[1].Value()
				completed, matches := autocomplete(currentPath)

				if completed != currentPath {
					m.addInputs[1].SetValue(completed)
					m.addInputs[1].SetCursor(len(completed))

					// Clear autocomplete options if we got a single match
					if len(matches) == 0 {
						m.autocompleteOpts = nil
						// Validate the completed path
						m.pathValidation = validatePath(completed)
					} else {
						m.autocompleteOpts = matches
					}
				} else if len(matches) > 0 {
					m.autocompleteOpts = matches
				}

				return m, nil
			}

			switch k {
			case "esc":
				m.mode = viewList
				m.statusMessage = "Cancelled"
				m.isError = false
				m.pathValidation = ""
				m.autocompleteOpts = nil
				for i := range m.addInputs {
					m.addInputs[i].SetValue("")
				}
				m.addFocusIndex = 0
				return m, nil
			case "shift+tab", "up":
				m.addFocusIndex--
				if m.addFocusIndex < 0 {
					m.addFocusIndex = len(m.addInputs)
				}
				for i := 0; i < len(m.addInputs); i++ {
					if i == m.addFocusIndex {
						m.addInputs[i].Focus()
					} else {
						m.addInputs[i].Blur()
					}
				}
				m.autocompleteOpts = nil
				return m, nil
			case "down":
				// Only navigate down if not showing autocomplete options
				if len(m.autocompleteOpts) == 0 {
					m.addFocusIndex++
					if m.addFocusIndex > len(m.addInputs) {
						m.addFocusIndex = 0
					}
					for i := 0; i < len(m.addInputs); i++ {
						if i == m.addFocusIndex {
							m.addInputs[i].Focus()
						} else {
							m.addInputs[i].Blur()
						}
					}
				}
				return m, nil
			case "enter":
				if m.addFocusIndex == len(m.addInputs) {
					name := strings.TrimSpace(m.addInputs[0].Value())
					path := strings.TrimSpace(m.addInputs[1].Value())
					tag := strings.TrimSpace(m.addInputs[2].Value())
					desc := strings.TrimSpace(m.addInputs[3].Value())

					if name == "" || path == "" {
						m.statusMessage = "Name and Path are required!"
						m.isError = true
						return m, nil
					}

					// Validate path before adding
					validation := validatePath(path)
					if validation != "" {
						m.statusMessage = "Cannot add project: " + strings.TrimPrefix(validation, "⚠ ")
						m.isError = true
						m.pathValidation = validation
						return m, nil
					}

					// Expand and use absolute path
					path = expandPath(path)

					project := Project{
						Name:        name,
						Path:        path,
						Tag:         tag,
						Description: desc,
					}

					if err := m.addProject(project); err != nil {
						m.statusMessage = fmt.Sprintf("Error: %v", err)
						m.isError = true
					} else {
						m.statusMessage = fmt.Sprintf("✓ Added '%s'", name)
						m.isError = false
						m.applyFilter("")
						m.mode = viewList
						m.pathValidation = ""
						m.autocompleteOpts = nil
						for i := range m.addInputs {
							m.addInputs[i].SetValue("")
						}
						m.addFocusIndex = 0
					}
					return m, nil
				} else {
					// Move to next field on enter
					m.addFocusIndex++
					if m.addFocusIndex > len(m.addInputs) {
						m.addFocusIndex = len(m.addInputs)
					}
					for i := 0; i < len(m.addInputs); i++ {
						if i == m.addFocusIndex {
							m.addInputs[i].Focus()
						} else {
							m.addInputs[i].Blur()
						}
					}
					m.autocompleteOpts = nil
					return m, nil
				}
			}

			if m.addFocusIndex < len(m.addInputs) {
				var cmd tea.Cmd
				oldValue := m.addInputs[m.addFocusIndex].Value()
				m.addInputs[m.addFocusIndex], cmd = m.addInputs[m.addFocusIndex].Update(msg)

				// Validate path in real-time if it's the path field
				if m.addFocusIndex == 1 {
					newValue := m.addInputs[1].Value()
					if newValue != oldValue {
						m.pathValidation = validatePath(newValue)
						m.autocompleteOpts = nil // Clear autocomplete when typing
					}
				}

				return m, cmd
			}
			return m, nil
		}

		if m.filterMode {
			switch k {
			case "enter":
				m.filterMode = false
				m.applyFilter(m.textInput.Value())
				if m.textInput.Value() != "" {
					m.statusMessage = fmt.Sprintf("✓ Filtered by '%s'", m.textInput.Value())
					m.isError = false
				}
				return m, nil
			case "esc":
				m.filterMode = false
				m.textInput.SetValue("")
				m.applyFilter("")
				m.statusMessage = "Filter cleared"
				m.isError = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		switch k {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a":
			m.mode = viewAdd
			m.addFocusIndex = 0
			m.addInputs[0].Focus()
			for i := 1; i < len(m.addInputs); i++ {
				m.addInputs[i].Blur()
			}
			m.statusMessage = ""
			m.pathValidation = ""
			m.autocompleteOpts = nil
			return m, nil
		case "/":
			m.filterMode = true
			m.textInput.Focus()
			return m, nil
		case "j", "down":
			if len(m.filteredIdxs) > 0 {
				m.cursor = (m.cursor + 1) % len(m.filteredIdxs)
				m.loadSelectedToViewport()
				m.statusMessage = ""
			}
			return m, nil
		case "k", "up":
			if len(m.filteredIdxs) > 0 {
				m.cursor = (m.cursor - 1 + len(m.filteredIdxs)) % len(m.filteredIdxs)
				m.loadSelectedToViewport()
				m.statusMessage = ""
			}
			return m, nil
		case "d":
			if len(m.filteredIdxs) == 0 {
				m.statusMessage = "No project to delete"
				m.isError = true
				return m, nil
			}
			idx := m.filteredIdxs[m.cursor]
			name := m.projects[idx].Name
			if err := m.deleteProject(idx); err != nil {
				m.statusMessage = fmt.Sprintf("Error: %v", err)
				m.isError = true
			} else {
				m.statusMessage = fmt.Sprintf("✓ Deleted '%s'", name)
				m.isError = false
				m.applyFilter(m.textInput.Value())
			}
			return m, nil
		case "o", "enter":
			if len(m.filteredIdxs) == 0 {
				m.statusMessage = "No project to open"
				m.isError = true
				return m, nil
			}
			idx := m.filteredIdxs[m.cursor]
			path := m.projects[idx].Path
			m.projects[idx].UpdatedAt = time.Now()
			m.saveProjects()
			m.statusMessage = fmt.Sprintf("Opening '%s'...", m.projects[idx].Name)
			m.isError = false
			return m, openProjectCmd(path)
		case "r":
			if err := m.loadProjects(); err != nil {
				m.statusMessage = fmt.Sprintf("Error: %v", err)
				m.isError = true
			} else {
				m.applyFilter(m.textInput.Value())
				m.statusMessage = "✓ Reloaded"
				m.isError = false
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.leftWidth = 45
			rightW := msg.Width - m.leftWidth - 8
			if rightW < 30 {
				rightW = 30
			}
			rightH := msg.Height - 6
			if rightH < 10 {
				rightH = 10
			}
			m.viewport.Height = rightH
			m.viewport.Width = rightW
			m.ready = true
			m.applyFilter(m.textInput.Value())
		} else {
			rightW := msg.Width - m.leftWidth - 8
			if rightW < 30 {
				rightW = 30
			}
			rightH := msg.Height - 6
			if rightH < 10 {
				rightH = 10
			}
			m.viewport.Width = rightW
			m.viewport.Height = rightH
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginTop(2).
			MarginLeft(2).
			Render("⚡ Loading Project Phonebook...")
	}

	if m.mode == viewAdd {
		var b strings.Builder

		header := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Background(lipgloss.Color("#1F2937")).
			Padding(1, 2).
			MarginBottom(2).
			Width(70).
			Render("✨ Add New Project")

		b.WriteString(header + "\n\n")

		labels := []string{"Project Name", "Project Path", "Tags", "Description"}
		for i, input := range m.addInputs {
			b.WriteString(labelStyle.Render(labels[i]) + "\n")
			b.WriteString(input.View() + "\n")

			// Show path validation message for path field
			if i == 1 && m.pathValidation != "" {
				b.WriteString(warningStyle.Render(m.pathValidation) + "\n")
			}

			// Show autocomplete options for path field
			if i == 1 && len(m.autocompleteOpts) > 0 && m.addFocusIndex == 1 {
				b.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Italic(true).
					Render(fmt.Sprintf("  %d matches - press tab again to cycle", len(m.autocompleteOpts))) + "\n")

				// Show first few matches
				maxShow := 5
				if len(m.autocompleteOpts) < maxShow {
					maxShow = len(m.autocompleteOpts)
				}
				for j := 0; j < maxShow; j++ {
					b.WriteString(lipgloss.NewStyle().
						Foreground(mutedColor).
						Render(fmt.Sprintf("    • %s", m.autocompleteOpts[j])) + "\n")
				}
				if len(m.autocompleteOpts) > maxShow {
					b.WriteString(lipgloss.NewStyle().
						Foreground(mutedColor).
						Render(fmt.Sprintf("    ... and %d more", len(m.autocompleteOpts)-maxShow)) + "\n")
				}
			}
		}

		submitBtn := "[ Submit ]"
		if m.addFocusIndex == len(m.addInputs) {
			submitBtn = focusedButtonStyle.Render("Submit")
		} else {
			submitBtn = blurredButtonStyle.Render("Submit")
		}

		b.WriteString("\n" + submitBtn + "\n\n")

		helpText := helpStyle.Render("tab: autocomplete/navigate • shift+tab: back • enter: submit • esc: cancel")
		b.WriteString(helpText)

		if m.statusMessage != "" {
			var statusText string
			if m.isError {
				statusText = errorStyle.Render("✗ " + m.statusMessage)
			} else {
				statusText = statusStyle.Render(m.statusMessage)
			}
			b.WriteString("\n" + statusText)
		}

		return lipgloss.NewStyle().
			Padding(2).
			Render(b.String())
	}

	var b strings.Builder

	// Header
	header := titleStyle.Render("📚 Project Phonebook")
	subtitle := subtitleStyle.Render(fmt.Sprintf("%d projects", len(m.projects)))
	b.WriteString(header + " " + subtitle + "\n\n")

	// Filter input
	if m.filterMode {
		filterBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Width(40).
			Render(m.textInput.View())
		b.WriteString(filterBox + "\n\n")
	}

	// Project list
	for i, idx := range m.filteredIdxs {
		p := m.projects[idx]

		var line string
		if i == m.cursor {
			line = selectedItemStyle.Render("▶ " + p.Name)
		} else {
			line = normalItemStyle.Render(p.Name)
		}
		b.WriteString(line + "\n")

		// Tag and path
		var metadata strings.Builder
		if p.Tag != "" {
			metadata.WriteString(tagStyle.Render(" #" + p.Tag))
		}
		metadata.WriteString("\n")
		metadata.WriteString(pathStyle.Render("   " + truncate(p.Path, 38)))

		b.WriteString(metadata.String() + "\n\n")

		if i >= 30 {
			b.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Render("   ... and more") + "\n")
			break
		}
	}

	left := leftPanelStyle.
		Width(m.leftWidth).
		Height(m.viewport.Height).
		Render(b.String())

	right := rightPanelStyle.
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Render(m.viewport.View())

	combined := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// Help bar
	helpKeys := []string{
		helpKey("j/k", "move"),
		helpKey("o/↵", "open"),
		helpKey("a", "add"),
		helpKey("d", "delete"),
		helpKey("/", "filter"),
		helpKey("r", "reload"),
		helpKey("q", "quit"),
	}
	help := helpStyle.Render(strings.Join(helpKeys, "  •  "))

	// Status message
	status := ""
	if m.statusMessage != "" {
		if m.isError {
			status = errorStyle.Render("✗ " + m.statusMessage)
		} else {
			status = statusStyle.Render(m.statusMessage)
		}
	}

	return combined + "\n" + help + status
}

func helpKey(key, desc string) string {
	return lipgloss.NewStyle().Foreground(primaryColor).Render(key) +
		lipgloss.NewStyle().Foreground(mutedColor).Render(" "+desc)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func openProjectCmd(path string) tea.Cmd {
	return func() tea.Msg {
		// Write debug info to a log file
		logFile, _ := os.OpenFile("/tmp/phonebook-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if logFile != nil {
			defer logFile.Close()
			fmt.Fprintf(logFile, "\n=== Opening project ===\n")
			fmt.Fprintf(logFile, "Time: %s\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(logFile, "Path: %s\n", path)
		}

		info, err := os.Stat(path)
		if err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "Stat error: %v\n", err)
			}
			return editorFinishedMsg{err: fmt.Errorf("path error: %v", err)}
		}

		if logFile != nil {
			fmt.Fprintf(logFile, "IsDir: %v\n", info.IsDir())
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim"
		}

		if logFile != nil {
			fmt.Fprintf(logFile, "Editor: %s\n", editor)
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "Abs path error: %v\n", err)
			}
			return editorFinishedMsg{err: err}
		}

		if logFile != nil {
			fmt.Fprintf(logFile, "Abs path: %s\n", absPath)
			fmt.Fprintf(logFile, "About to exec: %s %s\n", editor, absPath)
		}

		c := exec.Command(editor, absPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if logFile != nil {
			fmt.Fprintf(logFile, "Calling tea.ExecProcess...\n")
		}

		return tea.ExecProcess(c, func(err error) tea.Msg {
			// Log the callback
			logFile2, _ := os.OpenFile("/tmp/phonebook-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if logFile2 != nil {
				defer logFile2.Close()
				fmt.Fprintf(logFile2, "ExecProcess callback called\n")
				fmt.Fprintf(logFile2, "Error: %v\n", err)
			}
			return editorFinishedMsg{err: err}
		})
	}
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
