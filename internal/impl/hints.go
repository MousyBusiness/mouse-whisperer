package impl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gen "github.com/mousybusiness/mouse_whisperer/gen/mouse_whisperer/overlay/v1"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	log "github.com/sirupsen/logrus"
)

func RegisterHintsHandlers(registry Registry, state *state.State) {
	registry.Register("show_hints", func(ctx context.Context, args map[string]any) (context.Context, error) {
		lookup := state.GetLookup()
		if lookup == nil {
			overlay.SendUIMessage("Hints not loaded")
			return ctx, nil
		}

		// Get category order from config
		categoryOrder := lookup.GetCategoryOrder()

		// Build ordered categories map using ordered commands
		categories := make(map[string][]*commands.CommandEntry)
		for _, category := range categoryOrder {
			cmds := lookup.GetOrderedCommandsForCategory(category)
			if len(cmds) > 0 {
				categories[category] = cmds
			}
		}

		// Reset navigation state
		state.Global.CategoryList = []string{}
		state.Global.CategoryCommands = categories
		state.Global.CurrentCategory = ""
		state.Global.CurrentCommands = nil

		// Build list of available categories with indices
		var items []string
		index := 1
		for _, category := range categoryOrder {
			if cmds, exists := categories[category]; exists && len(cmds) > 0 {
				state.Global.CategoryList = append(state.Global.CategoryList, category)
				count := len(cmds)
				items = append(items, fmt.Sprintf("%d. %s [%d hints]", index, category, count))
				index++
			}
		}
		items = append(items, "", "Say 'pick N' or 'show $category hints'")

		// Send to overlay
		overlay.SendCommandsList("HINTS", items, 20)
		return ctx, nil
	})

	registry.Register("show_category", func(ctx context.Context, args map[string]any) (context.Context, error) {
		ex := xctx.Get(ctx)
		if ex == nil || len(ex.TrailerWords) == 0 {
			return ctx, nil
		}

		// Get the argument (category name or index)
		arg := strings.ToLower(ex.TrailerWords[0])

		// If they said "hints", let it fall through to the other handler
		if arg == "hints" {
			return ctx, nil
		}

		lookup := state.GetLookup()
		if lookup == nil {
			overlay.SendUIMessage("Hints not loaded")
			return ctx, nil
		}

		// Get valid categories from config
		validCategories := lookup.GetCategoryOrder()

		// Check if they're using a category name (e.g., "show global" or "show coding")
		categoryName := ""

		// Check if this matches any valid category
		for _, cat := range validCategories {
			if strings.ToLower(cat) == arg {
				categoryName = cat
				break
			}
		}

		// If we found a category by name, handle it
		if categoryName != "" {
			// Ensure CategoryCommands is populated with ordered commands
			if len(state.Global.CategoryCommands) == 0 {
				categories := make(map[string][]*commands.CommandEntry)
				for _, category := range validCategories {
					cmds := lookup.GetOrderedCommandsForCategory(category)
					if len(cmds) > 0 {
						categories[category] = cmds
					}
				}
				state.Global.CategoryCommands = categories
				state.Global.CategoryList = validCategories
			}

			cmds := state.Global.CategoryCommands[categoryName]
			state.Global.CurrentCategory = categoryName
			state.Global.CurrentCommands = cmds

			var items []string
			for i, cmd := range cmds {
				if len(cmd.Command.Triggers) > 0 {
					items = append(items, fmt.Sprintf("%d. %s", i+1, cmd.Command.Triggers[0]))
				}
			}
			items = append(items, "", "Say 'pick N' for hint details")

			overlay.SendCommandsList(categoryName, items, 20)
			return ctx, nil
		}

		// Parse the index
		index, err := strconv.Atoi(arg)
		if err != nil {
			overlay.SendUIMessage("Say 'show' followed by a number or category name")
			return ctx, nil
		}

		// Check if we're in a category (viewing hints) or at top level (viewing categories)
		if state.Global.CurrentCategory != "" {
			// We're drilling down into a specific hint
			if index < 1 || index > len(state.Global.CurrentCommands) {
				overlay.SendUIMessage(fmt.Sprintf("Hint %d not found. Say 'show hints' to go back", index))
				return ctx, nil
			}

			cmd := state.Global.CurrentCommands[index-1]

			// Send structured command config (simplified)
			overlay.SendCommandConfig(&gen.CommandConfig{
				Name:           string(cmd.Name),
				Description:    cmd.Command.Description,
				Triggers:       cmd.Command.Triggers,
				DismissSeconds: 20,
			})
			return ctx, nil
		}

		// We're selecting a category
		if len(state.Global.CategoryList) == 0 {
			overlay.SendUIMessage("Please say 'show hints' first")
			return ctx, nil
		}

		if index < 1 || index > len(state.Global.CategoryList) {
			overlay.SendUIMessage(fmt.Sprintf("Category %d not found. Say 'show hints' to see all categories", index))
			return ctx, nil
		}

		// Get the category name
		categoryName = state.Global.CategoryList[index-1]
		cmds := state.Global.CategoryCommands[categoryName]

		// Store current state for next drill-down
		state.Global.CurrentCategory = categoryName
		state.Global.CurrentCommands = cmds

		// Format the hints list
		var items []string
		for i, cmd := range cmds {
			if len(cmd.Command.Triggers) > 0 {
				items = append(items, fmt.Sprintf("%d. %s", i+1, cmd.Command.Triggers[0]))
			}
		}
		items = append(items, "", "Say 'pick N' for hint details")

		// Send to overlay
		overlay.SendCommandsList(fmt.Sprintf("%s", strings.ToUpper(categoryName)), items, 20)
		return ctx, nil
	})

	registry.Register("pick", func(ctx context.Context, args map[string]any) (context.Context, error) {
		ex := xctx.Get(ctx)
		if ex == nil || len(ex.TrailerWords) == 0 {
			overlay.SendUIMessage("Say 'pick' followed by a number")
			return ctx, nil
		}

		// Parse the index
		index, err := strconv.Atoi(ex.TrailerWords[0])
		if err != nil {
			overlay.SendUIMessage("Say 'pick' followed by a number")
			return ctx, nil
		}

		// Check if we're viewing audio inputs
		if state.Global.ViewingInputs {
			// Validate index
			if index < 1 || index > len(state.Global.AudioInputList) {
				overlay.SendUIMessage(fmt.Sprintf("Input %d not found", index))
				return ctx, nil
			}

			// Get the device (index is 1-based in UI, but 0-based in array)
			device := state.Global.AudioInputList[index-1]

			// Request device change
			err := state.SelectAudioInput(device.Index)
			if err != nil {
				overlay.SendUIMessage(fmt.Sprintf("Failed to switch input: %v", err))
				log.Errorf("Failed to request device change: %v", err)
				return ctx, err
			}

			// Clear viewing state
			state.Global.ViewingInputs = false
			state.Global.AudioInputList = nil

			// Dismiss the UI and show success message
			overlay.DismissCommands()
			overlay.SendUIMessage(fmt.Sprintf("Switching to: %s", device.Name))
			log.Infof("User selected audio input: %s (index: %d)", device.Name, device.Index)

			return ctx, nil
		}

		// Check if we're in a category (viewing hints) or at top level (viewing categories)
		if state.Global.CurrentCategory != "" {
			// We're picking a specific hint to show details
			if index < 1 || index > len(state.Global.CurrentCommands) {
				overlay.SendUIMessage(fmt.Sprintf("Hint %d not found", index))
				return ctx, nil
			}

			cmd := state.Global.CurrentCommands[index-1]

			// Send structured command config (simplified)
			overlay.SendCommandConfig(&gen.CommandConfig{
				Name:           string(cmd.Name),
				Description:    cmd.Command.Description,
				Triggers:       cmd.Command.Triggers,
				DismissSeconds: 20,
			})
			return ctx, nil
		}

		// We're picking a category
		if len(state.Global.CategoryList) == 0 {
			overlay.SendUIMessage("Please say 'show hints' first")
			return ctx, nil
		}

		if index < 1 || index > len(state.Global.CategoryList) {
			overlay.SendUIMessage(fmt.Sprintf("Category %d not found", index))
			return ctx, nil
		}

		// Get the category name and show its hints (already ordered in CategoryCommands)
		categoryName := state.Global.CategoryList[index-1]
		cmds := state.Global.CategoryCommands[categoryName]

		// Store current state for next drill-down
		state.Global.CurrentCategory = categoryName
		state.Global.CurrentCommands = cmds

		// Format the hints list
		var items []string
		for i, cmd := range cmds {
			if len(cmd.Command.Triggers) > 0 {
				items = append(items, fmt.Sprintf("%d. %s", i+1, cmd.Command.Triggers[0]))
			}
		}
		items = append(items, "", "Say 'pick N' for hint details")

		// Send to overlay
		overlay.SendCommandsList(fmt.Sprintf("%s", strings.ToUpper(categoryName)), items, 20)
		return ctx, nil
	})

	registry.Register("hide_hints", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Dismiss any active hints UI panels and reset navigation state
		state.Global.CurrentCategory = ""
		state.Global.CurrentCommands = nil
		overlay.DismissCommands()
		return ctx, nil
	})

	registry.Register("show_more", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Scroll the active UI panel down to show more content
		overlay.SendUIAction("scroll_down", 200)
		return ctx, nil
	})

	registry.Register("show_less", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Scroll the active UI panel up to show previous content
		overlay.SendUIAction("scroll_up", 200)
		return ctx, nil
	})
}
