package importlist

import (
	"fmt"
	"sort"
	"time"

	"github.com/magodo/aztfy/internal/meta"
	"github.com/magodo/aztfy/internal/ui/aztfyclient"
	"github.com/magodo/aztfy/internal/ui/common"
	"github.com/magodo/aztfy/schema"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/magodo/textinput"
)

type Model struct {
	c        meta.Meta
	listkeys listKeyMap

	list list.Model
}

func NewModel(c meta.Meta, l meta.ImportList, idx int) Model {
	// Build candidate words for the textinput
	candidates := make([]string, 0, len(schema.ProviderSchemaInfo.ResourceSchemas))
	for rt := range schema.ProviderSchemaInfo.ResourceSchemas {
		candidates = append(candidates, rt)
	}
	sort.Strings(candidates)

	// Build list items
	var items []list.Item
	for idx, item := range l {
		ti := textinput.NewModel()
		ti.SetCursorMode(textinput.CursorStatic)
		if !item.Skip() {
			ti.SetValue(fmt.Sprintf("%s.%s", item.TFResourceType, item.TFResourceName))
		}
		ti.CandidateWords = candidates
		items = append(items, Item{
			idx:       idx,
			v:         item,
			textinput: ti,
		})
	}

	list := list.NewModel(items, NewImportItemDelegate(), 0, 0)
	list.Title = " " + c.ResourceGroupName() + " "
	list.Styles.Title = common.SubtitleStyle
	list.StatusMessageLifetime = 3 * time.Second
	list.Select(idx)

	bindKeyHelps(&list, newListKeyMap().ToBindings())

	// Reset the quit to deallocate the "ESC" as a quit key.
	list.KeyMap.Quit = key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	)

	return Model{
		c:        c,
		listkeys: newListKeyMap(),

		list: list,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept the apply key (i.e. "w") when user is inputting.
		if m.isUserTyping() {
			break
		}

		switch {
		case key.Matches(msg, m.listkeys.apply):
			// Leave filter applied state before applying the import list
			if m.list.FilterState() == list.FilterApplied {
				m.list.ResetFilter()
			}

			// In case all items are marked as skip, show a warning and do nothing.
			if m.isNothingToImport() {
				return m, m.list.NewStatusMessage(common.ErrorMsgStyle.Render("All resources are skipped, nothing to import"))
			}

			// Ensure all items pass validation
			if !m.userInputsAreValid() {
				return m, m.list.NewStatusMessage(common.ErrorMsgStyle.Render("One or more user input is invalid"))
			}

			return m, aztfyclient.StartImport(m.c, m.importList(true))
		case key.Matches(msg, m.listkeys.error):
			sel := m.list.SelectedItem()
			if sel == nil {
				return m, nil
			}
			selItem := sel.(Item)
			if selItem.v.ImportError == nil {
				return m, nil
			}
			return m, aztfyclient.ShowImportError(selItem.v, selItem.idx, m.importList(false))
		}
	case tea.WindowSizeMsg:
		// The height here minus the height occupied by the title
		m.list.SetSize(msg.Width, msg.Height-3)
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func bindKeyHelps(l *list.Model, bindings []key.Binding) {
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return bindings
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return bindings
	}
}

func (m Model) isUserTyping() bool {
	// In filtering mode
	if m.list.FilterState() == list.Filtering {
		return true
	}

	// Any textinput is in focused mode
	for _, item := range m.list.Items() {
		item := item.(Item)
		if item.textinput.Focused() {
			return true
		}
	}
	return false
}

func (m Model) isNothingToImport() bool {
	for _, item := range m.list.Items() {
		item := item.(Item)
		if !item.v.Skip() {
			return false
		}
	}
	return true
}

func (m Model) userInputsAreValid() bool {
	for _, item := range m.list.Items() {
		item := item.(Item)
		if item.v.ValidateError != nil {
			return false
		}
	}
	return true
}

func (m Model) importList(clearErr bool) meta.ImportList {
	out := make(meta.ImportList, 0, len(m.list.Items()))
	for _, item := range m.list.Items() {
		item := item.(Item)
		if clearErr {
			item.v.ImportError = nil
		}
		out = append(out, item.v)
	}
	return out
}
