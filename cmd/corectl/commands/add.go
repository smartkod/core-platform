package commands

import (
	"fmt"

	"core-platform/cmd/corectl/levels"
)

// Add implements `corectl add module <name>` and `corectl add level <LN>`.
func Add(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("add: expected 'module <name>' or 'level <LN>'")
	}

	switch args[0] {
	case "module":
		if len(args) < 2 {
			return fmt.Errorf("add module: name is required")
		}
		return addModule(args[1])
	case "level":
		if len(args) < 2 {
			return fmt.Errorf("add level: expected L1 or L2")
		}
		return addLevel(args[1])
	default:
		return fmt.Errorf("add: unknown target %q (want module or level)", args[0])
	}
}

func addModule(name string) error {
	root, err := coreRoot()
	if err != nil {
		return err
	}
	if missing := missingModules(root, []levels.Module{{Name: name}}); len(missing) > 0 {
		return fmt.Errorf(
			"add module: %q does not exist in the core yet; "+
				"application modules are scaffolded into ./internal/modules by a later corectl release", name)
	}
	return fmt.Errorf("add module: not implemented yet for %q", name)
}

func addLevel(levelName string) error {
	level, err := levels.Parse(levelName)
	if err != nil {
		return err
	}
	mods, err := levels.ModulesFor(level, nil)
	if err != nil {
		return err
	}

	root, err := coreRoot()
	if err != nil {
		return err
	}
	if missing := missingModules(root, mods); len(missing) > 0 {
		return fatalMissing(missing, string(level))
	}
	return fmt.Errorf("add level: not implemented yet for %s", level)
}
