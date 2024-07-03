package runtimeconfig

type argServerCommand struct{}

// name implements argDef.
func (a argServerCommand) name() string {
	return "servercommand"
}

// usage implements argDef.
func (a argServerCommand) usage() string {
	return "The command to run the rotated server process"
}
