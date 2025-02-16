# Caia CLI - Chat with Claude 3.5 Sonnet

Caia CLI is a  command-line interface that allows you to interact with Claude 3.5 Sonnet to manage your codebase and Excel files. It provides seamless integration for creating, editing, and reading files with AI assistance. It is a work in progress and not ready for use yet. Please contribute to the project or wait for the next release.

## Features

- **Code Operations**
  - Create new files in multiple programming languages
  - Edit existing code files
  - Read file contents
  - Add features and fix bugs
  - Refactor code
  - Add tests and documentation

- **Excel Operations**
  - Create new Excel files with structured data
  - Read existing Excel files
  - Update spreadsheet content
  - Add or modify sheets
  - Handle multiple data types (strings, numbers, booleans)

- **Smart File Management**
  - Automatic workspace indexing
  - File type detection
  - Directory creation
  - Confirmation prompts for write operations
  - Automatic read operations

## Setup

1. Get your API key from [Anthropic Console](https://console.anthropic.com/)

2. Set up your API key in one of two ways:

   a. Using a `.env` file (recommended):
   ```bash
   # Copy the example .env file
   cp .env.example .env
   
   # Edit the .env file and add your API key
   nano .env
   ```

   b. Using environment variables:
   ```bash
   export ANTHROPIC_API_KEY='your-api-key'
   ```

   The CLI will check for the API key in the following order:
   1. `.env` file in the current directory
   2. Environment variables
   3. `.env.local` file in the current directory

## Installation

1. Make sure you have Go 1.23.4 or later installed
2. Clone this repository:
   ```bash
   git clone https://github.com/yourusername/caia-ai-cli.git
   cd caia-ai-cli
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Set up your Anthropic API key:
   ```bash
   export ANTHROPIC_API_KEY='your-api-key'
   ```

## Usage

1. Start the CLI:
   ```bash
   go run main.go
   ```

2. Available commands:
   - `/exit` - Exit the program
   - `/clear` - Clear conversation history
   - `/help` - Show help message
   - `/index` - Reindex workspace files

3. Example operations:
   ```
   > Create a Python script that generates random numbers
   > Show me what's in main.go
   > Create an Excel file with sample sales data
   > Add error handling to user_auth.js
   ```

## Examples

1. Creating a new Python file:
   ```
   > Create a Python script that calculates fibonacci numbers
   ```

2. Reading an Excel file:
   ```
   > Show me what's in sales_data.xlsx
   ```

3. Editing existing code:
   ```
   > Add input validation to login.js
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Dependencies

- [anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) - Anthropic Claude API SDK
- [excelize](https://github.com/xuri/excelize) - Excel file manipulation

## Acknowledgments

- Thanks to Anthropic for providing the Claude API
- Thanks to all contributors who have helped with the project # caia-ai-cli
