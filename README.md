# go-procrotator

Restart a server process when certain files change.

Install:

```shell
go install github.com/jakewan/go-procrotator
```

Add a configuration file named `procrotator.toml` or `.procrotator.toml` to the root directory of the target software project.

Here is an example configuration file for a typical Go server application. The hypothetical binary is named `some-go-server`.

```toml
include_file_regexes = ["\\.go$", "\\.tmpl$"]
preamble_commands = ["go build ."]
server_command = "./some-go-server"
```

Execute within the server application directory:

```shell
go-procrotator
```
