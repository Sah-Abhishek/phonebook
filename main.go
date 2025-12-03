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
	"unicode"

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

type fuzzyMatch struct {
	index int
	score int
}

var (
	// Color palette - Enhanced
	primaryColor   = lipgloss.Color("#A78BFA") // Lighter purple
	accentColor    = lipgloss.Color("#F472B6") // Pink
	successColor   = lipgloss.Color("#34D399") // Green
	mutedColor     = lipgloss.Color("#9CA3AF") // Gray
	brightColor    = lipgloss.Color("#FBBF24") // Amber
	bgColor        = lipgloss.Color("#111827") // Darker bg
	highlightColor = lipgloss.Color("#60A5FA") // Blue
	warningColor   = lipgloss.Color("#FB923C") // Orange
	textColor      = lipgloss.Color("#F3F4F6") // Light text

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1).
			MarginTop(1).
			Padding(0, 1)

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
				PaddingLeft(1).
				Background(lipgloss.Color("#1F2937"))

	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(3)

	tagStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Bold(true)

	pathStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	matchHighlightStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Underline(true)

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
				MarginTop(1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor)

	blurredButtonStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 3).
				MarginTop(1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor)

	// Detail view styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Underline(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(textColor).
				MarginLeft(1)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1).
			MarginBottom(1)

	// Filter box style
	filterBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			Width(40).
			Background(lipgloss.Color("#1F2937"))

	// Counter badge style
	counterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(highlightColor).
			Padding(0, 1).
			Bold(true).
			MarginLeft(1)
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
	pathValidation   string
	autocompleteOpts []string
	filterQuery      string // Store current filter query
}

func initialModel() model {
	home, _ := os.UserHomeDir()
	projectsFile := filepath.Join(home, ".config", "projects", "projects.json")
	os.MkdirAll(filepath.Dir(projectsFile), 0o755)

	ti := textinput.New()
	ti.Placeholder = "Press / to search projects..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(textColor)
	ti.CharLimit = 156
	ti.Width = 38

	vp := viewport.New(20, 10)
	vp.SetContent("")

	inputs := make([]textinput.Model, 4)
	inputStyles := lipgloss.NewStyle().Foreground(textColor)
	promptStyle := lipgloss.NewStyle().Foreground(primaryColor).Bold(true)

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

// fuzzyScore calculates a fuzzy match score for a query against a target string
func fuzzyScore(query, target string) int {
	if query == "" {
		return 0
	}

	query = strings.ToLower(query)
	target = strings.ToLower(target)

	// Exact match gets highest score
	if strings.Contains(target, query) {
		return 1000 + (100 - len(target))
	}

	// Fuzzy matching
	score := 0
	queryIdx := 0
	consecutiveMatches := 0

	for targetIdx := 0; targetIdx < len(target) && queryIdx < len(query); targetIdx++ {
		if target[targetIdx] == query[queryIdx] {
			score += 10 + consecutiveMatches*5 // Bonus for consecutive matches
			consecutiveMatches++
			queryIdx++

			// Bonus if match is at word boundary
			if targetIdx == 0 || !unicode.IsLetter(rune(target[targetIdx-1])) {
				score += 20
			}
		} else {
			consecutiveMatches = 0
		}
	}

	// Penalty for unmatched query characters
	if queryIdx < len(query) {
		return 0 // Not all query characters matched
	}

	return score
}

func (m *model) applyFilter(q string) {
	m.filterQuery = q
	q = strings.TrimSpace(q)

	if q == "" {
		// No filter, show all projects sorted by UpdatedAt
		m.filteredIdxs = m.filteredIdxs[:0]
		for i := range m.projects {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
	} else {
		// Fuzzy match and score
		var matches []fuzzyMatch
		for i, p := range m.projects {
			// Calculate score from all searchable fields
			nameScore := fuzzyScore(q, p.Name)
			tagScore := fuzzyScore(q, p.Tag)
			descScore := fuzzyScore(q, p.Description)
			pathScore := fuzzyScore(q, p.Path) / 2 // Lower weight for path

			maxScore := nameScore
			if tagScore > maxScore {
				maxScore = tagScore
			}
			if descScore > maxScore {
				maxScore = descScore
			}
			if pathScore > maxScore {
				maxScore = pathScore
			}

			if maxScore > 0 {
				matches = append(matches, fuzzyMatch{index: i, score: maxScore})
			}
		}

		// Sort by score descending
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})

		// Extract indices
		m.filteredIdxs = m.filteredIdxs[:0]
		for _, match := range matches {
			m.filteredIdxs = append(m.filteredIdxs, match.index)
		}
	}

	if len(m.filteredIdxs) == 0 {
		m.cursor = 0
		m.viewport.SetContent(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Align(lipgloss.Center).
			Render("✨ No projects match your search\n\nTry a different query or press 'a' to add a new project"))
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
			Align(lipgloss.Center).
			Render("No projects available\n\nPress 'a' to add your first project"))
		return
	}
	idx := m.filteredIdxs[m.cursor]
	p := m.projects[idx]

	var content strings.Builder

	// Project name with icon
	content.WriteString(detailLabelStyle.Render("Project Name") + "\n")
	content.WriteString(detailValueStyle.Render(p.Name) + "\n")
	content.WriteString(dividerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") + "\n")

	if p.Tag != "" {
		content.WriteString(detailLabelStyle.Render("  Tag") + "\n")
		content.WriteString(tagStyle.Render("# "+p.Tag) + "\n")
		content.WriteString(dividerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") + "\n")
	}

	content.WriteString(detailLabelStyle.Render(" Path") + "\n")
	content.WriteString(pathStyle.Render(p.Path) + "\n")
	content.WriteString(dividerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") + "\n")

	if p.Description != "" {
		content.WriteString(detailLabelStyle.Render(" Description") + "\n")
		content.WriteString(detailValueStyle.Render(p.Description) + "\n")
		content.WriteString(dividerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") + "\n")
	}

	content.WriteString(detailLabelStyle.Render(" Timeline") + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render(
		fmt.Sprintf("Created:  %s\nModified: %s",
			p.CreatedAt.Format("Jan 02, 2006 15:04"),
			p.UpdatedAt.Format("Jan 02, 2006 15:04"))))

	m.viewport.SetContent(content.String())
}

// highlightMatches highlights the matched characters in a string
func highlightMatches(query, target string) string {
	if query == "" {
		return target
	}

	query = strings.ToLower(query)
	targetLower := strings.ToLower(target)

	// Check for exact substring match first
	if idx := strings.Index(targetLower, query); idx != -1 {
		before := target[:idx]
		match := target[idx : idx+len(query)]
		after := target[idx+len(query):]
		return before + matchHighlightStyle.Render(match) + after
	}

	return target
}

// validatePath checks if the path exists and returns an error message if not
func validatePath(path string) string {
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "⚠ Path does not exist"
		}
		return fmt.Sprintf("⚠ Error: %v", err)
	}

	if !info.IsDir() {
		return " Path is not a directory"
	}

	return ""
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

	if strings.HasPrefix(currentPath, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			currentPath = filepath.Join(home, currentPath[1:])
		}
	}

	dir := filepath.Dir(currentPath)
	prefix := filepath.Base(currentPath)

	if strings.HasSuffix(currentPath, string(filepath.Separator)) {
		dir = currentPath
		prefix = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return currentPath, nil
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()

		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		if strings.HasPrefix(name, prefix) {
			fullPath := filepath.Join(dir, name)
			if entry.IsDir() {
				fullPath += string(filepath.Separator)
			}
			matches = append(matches, fullPath)
		}
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		return matches[0], nil
	}

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
			if k == "tab" && m.addFocusIndex == 1 {
				currentPath := m.addInputs[1].Value()
				completed, matches := autocomplete(currentPath)

				if completed != currentPath {
					m.addInputs[1].SetValue(completed)
					m.addInputs[1].SetCursor(len(completed))

					if len(matches) == 0 {
						m.autocompleteOpts = nil
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

					validation := validatePath(path)
					if validation != "" {
						m.statusMessage = "Cannot add project: " + strings.TrimPrefix(validation, "⚠ ")
						m.isError = true
						m.pathValidation = validation
						return m, nil
					}

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

				if m.addFocusIndex == 1 {
					newValue := m.addInputs[1].Value()
					if newValue != oldValue {
						m.pathValidation = validatePath(newValue)
						m.autocompleteOpts = nil
					}
				}

				return m, cmd
			}
			return m, nil
		}

		if m.filterMode {
			switch k {
			case "esc":
				m.filterMode = false
				m.textInput.Blur()
				m.textInput.SetValue("")
				m.applyFilter("")
				m.statusMessage = ""
				return m, nil
			case "enter":
				// Open the selected project with Enter
				if len(m.filteredIdxs) == 0 {
					m.statusMessage = "No project to open"
					m.isError = true
					return m, nil
				}
				m.filterMode = false
				m.textInput.Blur()
				idx := m.filteredIdxs[m.cursor]
				path := m.projects[idx].Path
				m.projects[idx].UpdatedAt = time.Now()
				m.saveProjects()
				m.statusMessage = fmt.Sprintf("Opening '%s'...", m.projects[idx].Name)
				m.isError = false
				return m, openProjectCmd(path)
			case "down", "ctrl+n":
				// Navigate down while filtering
				if len(m.filteredIdxs) > 0 {
					m.cursor = (m.cursor + 1) % len(m.filteredIdxs)
					m.loadSelectedToViewport()
				}
				return m, nil
			case "up", "ctrl+p":
				// Navigate up while filtering
				if len(m.filteredIdxs) > 0 {
					m.cursor = (m.cursor - 1 + len(m.filteredIdxs)) % len(m.filteredIdxs)
					m.loadSelectedToViewport()
				}
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				oldValue := m.textInput.Value()
				m.textInput, cmd = m.textInput.Update(msg)

				// Real-time filtering
				newValue := m.textInput.Value()
				if newValue != oldValue {
					m.applyFilter(newValue)
				}

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
			m.statusMessage = ""
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
			Render(" Loading Project Phonebook...")
	}

	if m.mode == viewAdd {
		var b strings.Builder

		header := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Background(bgColor).
			Padding(1, 2).
			MarginBottom(2).
			Width(70).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Render("✨ Add New Project")

		b.WriteString(header + "\n\n")

		labels := []string{" Project Name", " Project Path", "  Tags", " Description"}
		for i, input := range m.addInputs {
			b.WriteString(labelStyle.Render(labels[i]) + "\n")
			b.WriteString(input.View() + "\n")

			if i == 1 && m.pathValidation != "" {
				b.WriteString(warningStyle.Render(m.pathValidation) + "\n")
			}

			if i == 1 && len(m.autocompleteOpts) > 0 && m.addFocusIndex == 1 {
				b.WriteString(lipgloss.NewStyle().
					Foreground(mutedColor).
					Italic(true).
					Render(fmt.Sprintf("   %d matches - press tab again to cycle", len(m.autocompleteOpts))) + "\n")

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
			submitBtn = focusedButtonStyle.Render("  Submit  ")
		} else {
			submitBtn = blurredButtonStyle.Render("  Submit  ")
		}

		b.WriteString("\n" + submitBtn + "\n\n")

		helpText := helpStyle.Render("tab: autocomplete/next • shift+tab: previous • enter: submit • esc: cancel")
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

	var leftContent strings.Builder

	// Header with counter (2 lines)
	header := titleStyle.Render(" Project Phonebook")
	count := counterStyle.Render(fmt.Sprintf("%d", len(m.projects)))
	if m.filterQuery != "" {
		filteredCount := counterStyle.Render(fmt.Sprintf("%d/%d", len(m.filteredIdxs), len(m.projects)))
		leftContent.WriteString(header + " " + filteredCount + "\n\n")
	} else {
		leftContent.WriteString(header + " " + count + "\n\n")
	}

	// Filter input - always show it (2 lines with spacing)
	filterBox := filterBoxStyle.Render(m.textInput.View())
	leftContent.WriteString(filterBox + "\n\n")

	// Calculate how many items can fit
	// Fixed header elements take up specific lines
	// - Header: 2 lines
	// - Filter box: 2 lines
	// - Border padding: ~4 lines
	// Each project item: 4 lines
	fixedHeaderLines := 8
	linesPerItem := 4

	availableLines := m.viewport.Height - fixedHeaderLines
	if availableLines < 0 {
		availableLines = 4
	}
	maxDisplay := availableLines / linesPerItem
	if maxDisplay < 1 {
		maxDisplay = 1
	}

	// Calculate scroll window
	startIdx := 0
	endIdx := len(m.filteredIdxs)

	if len(m.filteredIdxs) > maxDisplay {
		// Center the cursor in the visible window
		startIdx = m.cursor - maxDisplay/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + maxDisplay
		if endIdx > len(m.filteredIdxs) {
			endIdx = len(m.filteredIdxs)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Render only the items that fit
	if len(m.filteredIdxs) == 0 {
		leftContent.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Render("✨ No projects match\n\nPress 'a' to add one"))
	} else {
		for i := startIdx; i < endIdx; i++ {
			idx := m.filteredIdxs[i]
			p := m.projects[idx]

			var line string
			displayName := highlightMatches(m.filterQuery, p.Name)

			if i == m.cursor {
				line = selectedItemStyle.Render("▶ " + displayName)
			} else {
				line = normalItemStyle.Render(displayName)
			}
			leftContent.WriteString(line + "\n")

			// Tag and path
			var metadata strings.Builder
			if p.Tag != "" {
				highlightedTag := highlightMatches(m.filterQuery, p.Tag)
				metadata.WriteString(tagStyle.Render(" #" + highlightedTag))
			}
			metadata.WriteString("\n")
			metadata.WriteString(pathStyle.Render("   " + truncate(p.Path, 38)))

			leftContent.WriteString(metadata.String() + "\n\n")
		}

		// Show scroll indicator if needed
		if len(m.filteredIdxs) > maxDisplay {
			scrollInfo := fmt.Sprintf("   [%d-%d of %d]", startIdx+1, endIdx, len(m.filteredIdxs))
			leftContent.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				Render(scrollInfo))
		}
	}

	left := leftPanelStyle.
		Width(m.leftWidth).
		Height(m.viewport.Height).
		Render(leftContent.String())

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
		helpKey("/", "search"),
		helpKey("esc", "clear search"),
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
	return lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(key) +
		lipgloss.NewStyle().Foreground(mutedColor).Render(" "+desc)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func openProjectCmd(path string) tea.Cmd {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return func() tea.Msg {
			return editorFinishedMsg{err: fmt.Errorf("path does not exist: %s", path)}
		}
	}

	c := exec.Command("nvim", ".")
	c.Dir = path

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
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
