package commands

import (
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/elliotchance/orderedmap/v3"
	log "github.com/sirupsen/logrus"
)

type TrailerText string
type IncorrectTriggerPhrase string
type TriggerPhrase string

func (t TriggerPhrase) HasPrefix(p string) bool {
	if len(t) == 0 {
		return false
	}
	return strings.HasPrefix(string(t), p)
}

type CategoryName string
type CommandName string
type TriggerMap = *orderedmap.OrderedMap[TriggerPhrase, *CommandEntry]
type EntryMap = *orderedmap.OrderedMap[CommandName, *CommandEntry]
type CategoryEntryMap = *orderedmap.OrderedMap[CategoryName, EntryMap]

// CommandLookup provides efficient command lookup by trigger phrase
type CommandLookup struct {
	Categorized CategoryEntryMap
	Triggers    TriggerMap
}

// CommandEntry pairs a command with its category and name
type CommandEntry struct {
	Category CategoryName
	Name     CommandName
	Command  *Command
}

// NewCommandLookup builds an efficient lookup structure from the config
func NewCommandLookup(raw map[string]map[string]Command, order []string) *CommandLookup {
	lookup := &CommandLookup{
		Triggers:    orderedmap.NewOrderedMap[TriggerPhrase, *CommandEntry](),
		Categorized: orderedmap.NewOrderedMap[CategoryName, EntryMap](),
	}

	// Enforce the stipulated ordering of categories (in config settings block)
	configCategories := slices.Collect(maps.Keys(raw))
	for _, k := range order {
		if !slices.Contains(configCategories, k) {
			log.Warn("Invalid category in category ordering: " + k)
			continue
		}
		lookup.Categorized.Set(CategoryName(k), orderedmap.NewOrderedMap[CommandName, *CommandEntry]())
	}

	for _, c := range configCategories {
		catName := CategoryName(c)

		// Add category to lookup that wasn't in order preference
		if _, hasCategory := lookup.Categorized.Get(catName); !hasCategory {
			lookup.Categorized.Set(catName, orderedmap.NewOrderedMap[CommandName, *CommandEntry]())
		}

		commandNames := slices.Collect(maps.Keys(raw[c]))
		// Sort commands in alphabetical order
		sort.Strings(commandNames)

		for _, cmdName := range commandNames {
			cmd := raw[c][cmdName]
			cmdCopy := cmd // Create a copy to avoid pointer issues
			entry := &CommandEntry{
				Category: catName,
				Name:     CommandName(cmdName),
				Command:  &cmdCopy,
			}

			for _, trigger := range cmd.Triggers {
				lookup.Triggers.Set(TriggerPhrase(trigger), entry)
			}

			m, ok := lookup.Categorized.Get(catName)
			if !ok {
				panic("Internal error: category not found")
			}
			m.Set(CommandName(cmdName), entry)
		}
	}

	return lookup
}

// GetCategoryOrder returns the ordered list of categories from config
func (cl *CommandLookup) GetCategoryOrder() []string {
	var result []string
	for cat := range cl.Categorized.AllFromFront() {
		result = append(result, string(cat))
	}
	return result
}

// GetOrderedCommandsForCategory returns commands for a category in their config order
func (cl *CommandLookup) GetOrderedCommandsForCategory(category string) []*CommandEntry {
	v, ok := cl.Categorized.Get(CategoryName(category))
	if !ok {
		return nil
	}

	var result []*CommandEntry
	for _, entry := range v.AllFromFront() {
		result = append(result, entry)
	}

	return result
}
