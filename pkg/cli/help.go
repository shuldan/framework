package cli

import (
	"flag"
	"fmt"
	"strconv"
	"text/template"

	"github.com/shuldan/framework/pkg/contracts"
)

type HelpCommand struct {
	registry contracts.CliRegistry
	command  string
}

func NewHelpCommand(registry contracts.CliRegistry) contracts.CliCommand {
	return &HelpCommand{
		registry: registry,
	}
}

func (h *HelpCommand) Name() string {
	return "help"
}

func (h *HelpCommand) Description() string {
	return "Display help for commands"
}

func (h *HelpCommand) Group() string {
	return contracts.SystemCliGroup
}

func (h *HelpCommand) Configure(flags *flag.FlagSet) {
	flags.StringVar(&h.command, "command", "", "Show help for specific command")
}

func (h *HelpCommand) Validate(contracts.CliContext) error {
	return nil
}

func (h *HelpCommand) Execute(ctx contracts.CliContext) error {
	if h.command != "" {
		if err := h.showCommandHelp(ctx, h.command); err != nil {
			return err
		}
		ctx.Ctx().Stop()
		return nil
	}

	if err := h.showGeneralHelp(ctx); err != nil {
		return err
	}
	ctx.Ctx().Stop()
	return nil
}

func (h *HelpCommand) showGeneralHelp(ctx contracts.CliContext) error {
	groups := h.registry.Groups()

	data := struct {
		Groups map[string][]PrintableCommand
	}{
		Groups: make(map[string][]PrintableCommand),
	}

	for groupName, commands := range groups {
		longest := 0
		for _, cmd := range commands {
			if len(cmd.Name()) > longest {
				longest = len(cmd.Name())
			}
		}

		formatter := "%-" + strconv.Itoa(longest) + "s"
		printableCommands := make([]PrintableCommand, 0, len(commands))

		for _, cmd := range commands {
			printableCommands = append(printableCommands, PrintableCommand{
				PaddedName:  fmt.Sprintf(formatter, cmd.Name()),
				Description: cmd.Description(),
			})
		}

		data.Groups[groupName] = printableCommands
	}

	tmpl := template.New("help")
	helpTemplate := `Usage: command [options] [arguments]

{{ range $group, $commands := .Groups }}{{ $group }}:{{ range $commands }}
  {{.PaddedName}}  {{.Description}}{{ end }}

{{ end }}`

	template.Must(tmpl.Parse(helpTemplate))

	return tmpl.Execute(ctx.Output(), data)
}

func (h *HelpCommand) showCommandHelp(ctx contracts.CliContext, commandName string) error {
	command, exists := h.registry.Get(commandName)
	if !exists {
		return ErrHelpCommandNotFound.WithDetail("command", commandName)
	}

	if command == nil {
		return ErrHelpCommandNotFound.WithDetail("command", commandName).WithDetail("reason", "nil command")
	}

	output := ctx.Output()
	if _, err := fmt.Fprintf(output, "%s - %s\n\n", command.Name(), command.Description()); err != nil {
		return ErrCommandExecution.WithDetail("command", command.Name()).WithCause(err)
	}

	flags := flag.NewFlagSet(command.Name(), flag.ContinueOnError)
	flags.SetOutput(output)
	command.Configure(flags)

	if _, err := fmt.Fprintf(output, "Options:\n"); err != nil {
		return ErrCommandExecution.WithDetail("command", command.Name()).WithCause(err)
	}

	flags.PrintDefaults()

	return nil
}

type PrintableCommand struct {
	PaddedName  string
	Description string
}
