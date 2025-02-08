# Gogenesis Code Generator

Gogenesis is a command-line tool that converts a CIP-0057 Plutus blueprint schema (`plutus.json`) into code types for multiple target languages. Currently, only TypeScript is supported.

The typescript generator is for use with [Lucid Evolution](https://github.com/Anastasia-Labs/lucid-evolution)

**Note**: Currently only tested for Plutus V2.

## Demo

## Installation

Download from: https://github.com/mgpai22/gogenesis/releases/

## Directory Structure

- **cmd/gogenesis/main.go**: The main entry point which parses CLI flags, loads the Plutus JSON, and invokes the appropriate code generator.
- **internal/parser/**: Contains logic for parsing the Plutus JSON schema.
- **internal/generator/**: Hosts the common generator logic and shared helper functions.
  - **internal/generator/typescript/**: Implements the TypeScript code generator.
  - **internal/generator/golang/**: Implements the Go code generator.

## Build Instructions

1. **Clone the repository:**

   ```bash
   git clone https://github.com/mgpai22/gogenesis.git
   cd gogenesis
   ```

2. **Build the tool:**

   ```bash
   make
   ```

   This will produce an executable named `gogenesis` (or `gogenesis.exe` on Windows) in the project root directory.

## Usage

Run the generated binary from the command line. The required and optional flags are:

- **-json**: Path to the Plutus JSON schema file _(required)_.
- **-out**: Output directory for the generated files (default is `./generated`).
- **-lang**: Target language. Options are `typescript` and `golang` (default is `typescript`).

**Note**: At the moment, `golang` only generates empty types.

### Example

To generate TypeScript types:

```bash
./gogenesis -json path/to/plutus.json -out ./path/to/output -lang typescript
```

## Contributing

Contributions to extend and improve the generator (or to add more target languages) are welcome. Please open issues or pull requests on GitHub.
