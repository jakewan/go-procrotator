package runtimeconfig

type argPreambleCommand struct{}

// name implements argDef.
func (a argPreambleCommand) name() string {
	return "preamblecommand"
}

// usage implements argDef.
func (a argPreambleCommand) usage() string {
	return `A command to execute before the server command.

May be specified multiple times.`
}
