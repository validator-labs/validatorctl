[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/validator-labs/validatorctl/issues)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Test](https://github.com/validator-labs/validatorctl/actions/workflows/test.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/validator-labs/validatorctl)](https://goreportcard.com/report/github.com/validator-labs/validatorctl)
[![codecov](https://codecov.io/gh/validator-labs/validatorctl/graph/badge.svg?token=GVZ4LZ5SOY)](https://codecov.io/gh/validator-labs/validatorctl)
[![Go Reference](https://pkg.go.dev/badge/github.com/validator-labs/validatorctl.svg)](https://pkg.go.dev/github.com/validator-labs/validatorctl)

# validatorctl
A CLI tool for the validator ecosystem.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Setup](#setup)
  - [Binary Installation](#binary-installation)
  - [Building from Source](#building-from-source)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Prerequisites

The `validatorctl` relies on a few binaries that you'll need to ensure you have installed:
- Docker v24.0.6+
- Helm v3.14.0+
- Kind v0.20.0+
- Kubectl v1.24.10+

## Setup

### Binary Installation

You can download the `validatorctl` binary you require directly from the [releases page](https://github.com/validator-labs/validatorctl/releases) or via curl.
For instance, the v0.0.1 darwin binary can be installed and run like this:

```sh
curl -L -O https://github.com/validator-labs/validatorctl/releases/download/v0.0.1/validator-darwin-arm64
chmod +x validator-darwin-arm64
sudo mv validator-darwin-arm64 /usr/local/bin/validator
validator help
```

### Building from Source

To build `validatorctl` from source, you'll need to ensure you're running `go1.22.4`.
You can then build `validatorctl` and run it with the following commands:

```sh
make build-cli
./bin/validator help
```

## Usage
`validatorctl` provides several commands for managing validator plugins. Below are some common commands:
- Install validator plugins with the `validator install` command.
- Describe validation results with the `validator describe` command.
- Re-configure validator plugins after they've been installed with the `validator upgrade` command.
- Uninstall the validator and all plugins with the `validator uninstall` command.

For more information about any supported `validatorctl` command, run `validator help`
```sh
❯ validator help
Welcome to the Validator CLI.
Install validator & configure validator plugins.
Use 'validator help <sub-command>' to explore all of the functionality the Validator CLI has to offer.

Usage:
  validator [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  describe    Describe all validation results in a Kubernetes cluster
  help        Help about any command
  install     Install validator & configure validator plugin(s)
  uninstall   Uninstall validator & all validator plugin(s)
  upgrade     Upgrade validator & re-configure validator plugin(s)
  version     Prints the Validator CLI version

Flags:
  -c, --config string      Validator CLI config file location
  -h, --help               help for validator
  -l, --log-level string   Log level. One of: [panic fatal error warn info debug trace] (default "info")
  -w, --workspace string   Workspace location for staging runtime configurations and logs (default "$HOME/.validator")

Use "validator [command] --help" for more information about a command.
```

## Contributing
Contributions are always welcome; take a look at our [contributions guide](https://github.com/validator-labs/.github/blob/main/.github/CONTRIBUTING.md) and [code of conduct](https://github.com/validator-labs/.github/blob/main/.github/CODE_OF_CONDUCT.md) to get started.

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
