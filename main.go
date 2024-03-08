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
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("205"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("205"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
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

	str := fmt.Sprintf(" %s (%s) %s", i.name, i.ago, i.author)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(" >" + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	choice   *item
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			item, ok := m.list.SelectedItem().(item)

			if ok {
				m.choice = &item

				cmd := exec.Command("git", "checkout", m.choice.name)
				cmd.Stdout = os.Stdout
				cmd.Run()
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
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

	items, err := getItems()

	if err != nil {
		os.Exit(1)
	}
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "✨ Recent branches ✨"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
