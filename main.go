package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	itemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("50")).Bold(true)
	fadedTextStyle 		= lipgloss.NewStyle().Faint(true)
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingBottom(1)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingBottom(1)
)

type item struct {
	name   string
	ago    string
	author string
}

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	style := itemStyle

	icon := "☆ "

	if isSelected {
		style = selectedItemStyle
		icon = "★ "
	}

	content := icon + fadedTextStyle.Render(fmt.Sprintf("(%s)", i.ago)) + style.Render(fmt.Sprintf(" %s", i.name))


	fmt.Fprint(w, content)
}

type model struct {
	list     list.Model
	choice   *item
	quitting bool
	err      error
}

func (m model) Init() tea.Cmd {
	return nil
}

type checkoutMsg struct {
	err error
}

func (m model) Checkout() tea.Cmd {
	cmd := exec.Command("git", "checkout", m.choice.name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return checkoutMsg{err: err}
	})

}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	case checkoutMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, tea.Quit
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			item, ok := m.list.SelectedItem().(item)

			if ok {
				m.choice = &item

				return m, m.Checkout()
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n error: %s", m.err)

	}
	return "\n" + m.list.View()
}

func getItems() ([]list.Item, error) {
	gitCmd := "git for-each-ref --sort=-committerdate refs/heads refs/remotes --format='%(color:yellow)%(refname:short)%(color:reset) (%(color:green)%(committerdate:relative)%(color:reset)) %(authorname)' | head -n 150"
	cmd := exec.Command("bash", "-c", gitCmd)

	b, err := cmd.CombinedOutput()

	if err != nil {
		return nil, err
	}

	output := string(b)

	lines := strings.Split(output, "\n")

	items := []list.Item{}

	for _, line := range lines {
		// Remove the color codes
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, "origin/") {
			continue
		}

		name, ago, author, err := parseLine(line)

		if err != nil {
			continue
		}
		items = append(items, item{
			name:   name,
			ago:    ago,
			author: author,
		})
	}

	return items, nil
}

func parseLine(line string) (string, string, string, error) {
	re := regexp.MustCompile(`(.*)\s[(](.*)[)]\s(\w+)`)

	matches := re.FindStringSubmatch(line)

	if len(matches) != 4 {
		return "", "", "", fmt.Errorf("could not parse line: %s", line)
	}

	return matches[1], matches[2], matches[3], nil
}

func main() {
	const defaultWidth = 20

	items, _ := getItems()

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
