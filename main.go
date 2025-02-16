package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/xuri/excelize/v2"

	"caia-ai-cli/pkg/config"
)

type FileInfo struct {
	Path       string         `json:"path"`
	Name       string         `json:"name"`
	Size       int64          `json:"size"`
	ModTime    time.Time      `json:"mod_time"`
	IsDir      bool           `json:"is_dir"`
	Language   string         `json:"language,omitempty"`
	SheetNames []string       `json:"sheet_names,omitempty"`
	RowCount   map[string]int `json:"row_count,omitempty"`
}

var workspaceFiles []FileInfo

func getFileLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "Go"
	case ".js", ".jsx":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	case ".cpp", ".cc", ".cxx":
		return "C++"
	case ".c":
		return "C"
	case ".rs":
		return "Rust"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".swift":
		return "Swift"
	case ".kt":
		return "Kotlin"
	case ".cs":
		return "C#"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".md":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".xlsx", ".xls":
		return "Excel"
	default:
		return ""
	}
}

func indexWorkspace() error {
	workspaceFiles = []FileInfo{} // Reset the index
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		fileInfo := FileInfo{
			Path:    path,
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		if !info.IsDir() {
			fileInfo.Language = getFileLanguage(info.Name())

			// Handle Excel files
			if fileInfo.Language == "Excel" {
				if f, err := excelize.OpenFile(path); err == nil {
					defer f.Close()
					fileInfo.SheetNames = f.GetSheetList()
					fileInfo.RowCount = make(map[string]int)

					for _, sheet := range fileInfo.SheetNames {
						if rows, err := f.GetRows(sheet); err == nil {
							fileInfo.RowCount[sheet] = len(rows)
						}
					}
				}
			}
		}

		workspaceFiles = append(workspaceFiles, fileInfo)
		return nil
	})
	return err
}

const (
	welcomeMessage = `
Welcome to Caia CLI - Chat with Claude 3.5 Sonnet
================================================
Commands:
  /exit   - Exit the program
  /clear  - Clear conversation history
  /help   - Show this help message
  /index  - Reindex workspace files

You can ask Claude to help you with:

Code Operations:
- Create new files in any language
- Edit existing code files
- Fix bugs or add features
- Refactor code
- Add tests and documentation

Excel Operations:
- Create new Excel files with data
- Read existing Excel files
- Update spreadsheet content
- Add or modify sheets

Examples:
- "Create a Java class for handling user authentication"
- "Make a JavaScript file for form validation"
- "Create an Excel file with sample sales data"
- "Add error handling to main.go"
- "Show me what's in the budget spreadsheet"

Each operation will ask for your confirmation before making any changes.
Start typing to chat with Claude!
`
	systemPrompt = `You are an AI assistant that helps users work with their codebase and Excel files. You have access to information about all files in the current workspace.

Current workspace files:
%s

IMPORTANT RULES FOR ALL RESPONSES:
1. Keep responses focused and well-structured
2. Support multiple operations when requested
3. Never use triple quotes or special characters in JSON
4. Always use proper JSON escaping with single backslash (\\n for newlines)
5. Each operation must be a complete, valid JSON object
6. DO NOT create bug fixes or improvements to the codebase unless explicitly asked
7. DO NOT remove any existing code, features or files unless explicitly asked
8. Carefully review the codebase before making any changes
9. Try to adhere to the existing code style and structure
10. Ensure all code is properly formatted and indented, and use proper file extensions
11. Do not cause any harm to the codebase or the system
12. Do not create new bugs or issues
13. If you are unsure about the changes, ask the user for clarification
14. If you are unsure about the codebase, ask the user for clarification
15. If you are unsure about the user's request, ask the user for clarification



For code files (non-Excel):
{
    "operation": "create",
    "filename": "example.py",
    "content": "def hello():\\n    print('Hello')\\n"
}

For Excel files:
{
    "operation": "create",
    "filename": "data.xlsx",
    "actions": [
        {
            "type": "create_sheet",
            "sheet": "Sheet1"
        },
        {
            "type": "add_row",
            "sheet": "Sheet1",
            "row": ["Header1", "Header2"]
        }
    ]
}

Guidelines:
- Each operation must be a separate, complete JSON object
- Use proper file extensions
- Include necessary imports
- Add basic comments
- Keep all content concise

Example response for multiple files:
"I'll create two simple files.

{
    "operation": "create",
    "filename": "hello.py",
    "content": "def hello():\\n    print('Hello Python!')\\n"
}
{
    "operation": "create",
    "filename": "greet.js",
    "content": "function greet() {\\n    console.log('Hello JavaScript!');\\n}\\n"
}"

Remember: Each operation must be a complete, valid JSON object with proper escaping.`
)

type Action struct {
	Operation string        `json:"operation"`
	Filename  string        `json:"filename"`
	Content   string        `json:"content,omitempty"`
	Actions   []ExcelAction `json:"actions,omitempty"`
}

type ExcelAction struct {
	Type  string   `json:"type"`
	Sheet string   `json:"sheet"`
	Cell  string   `json:"cell,omitempty"`
	Value string   `json:"value,omitempty"`
	Row   []string `json:"row,omitempty"`
}

func promptForConfirmation(action Action) bool {
	// Skip confirmation for read operations
	if action.Operation == "read" {
		return true
	}

	var prompt string
	switch action.Operation {
	case "create":
		if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") ||
			strings.HasSuffix(strings.ToLower(action.Filename), ".xls") {
			prompt = fmt.Sprintf("\nDo you want to create Excel file '%s' with %d sheet operations? (y/n): ",
				action.Filename, len(action.Actions))
		} else {
			// For code files, show a preview
			contentPreview := action.Content
			if len(contentPreview) > 200 {
				contentPreview = contentPreview[:200] + "..."
			}
			prompt = fmt.Sprintf("\nDo you want to create '%s' with the following content?\n\nPreview:\n%s\n\n(y/n): ",
				action.Filename, contentPreview)
		}
	case "edit":
		if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") ||
			strings.HasSuffix(strings.ToLower(action.Filename), ".xls") {
			prompt = fmt.Sprintf("\nDo you want to edit Excel file '%s' with %d operations? (y/n): ",
				action.Filename, len(action.Actions))
		} else {
			contentPreview := action.Content
			if len(contentPreview) > 200 {
				contentPreview = contentPreview[:200] + "..."
			}
			prompt = fmt.Sprintf("\nDo you want to edit '%s' with the following content?\n\nPreview:\n%s\n\n(y/n): ",
				action.Filename, contentPreview)
		}
	default:
		prompt = fmt.Sprintf("\nDo you want to perform '%s' operation on '%s'? (y/n): ",
			action.Operation, action.Filename)
	}

	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func handleOperation(action Action) error {
	// For read operations, skip the "Operation cancelled" message
	if !promptForConfirmation(action) {
		if action.Operation != "read" {
			fmt.Println("Operation cancelled by user.")
		}
		return nil
	}

	switch action.Operation {
	case "create":
		if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") ||
			strings.HasSuffix(strings.ToLower(action.Filename), ".xls") {
			return handleExcelOperation(action)
		}
		// For non-Excel files, create with content
		dir := filepath.Dir(action.Filename)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("error creating directory: %v", err)
			}
		}
		// Replace escaped newlines with actual newlines
		content := strings.ReplaceAll(action.Content, "\\n", "\n")
		return os.WriteFile(action.Filename, []byte(content), 0644)
	case "edit":
		if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") ||
			strings.HasSuffix(strings.ToLower(action.Filename), ".xls") {
			return handleExcelOperation(action)
		}
		// Replace escaped newlines with actual newlines
		content := strings.ReplaceAll(action.Content, "\\n", "\n")
		return os.WriteFile(action.Filename, []byte(content), 0644)
	case "read":
		if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") ||
			strings.HasSuffix(strings.ToLower(action.Filename), ".xls") {
			return handleExcelOperation(action)
		}
		// Read non-Excel files
		content, err := os.ReadFile(action.Filename)
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}
		fmt.Printf("\nContents of %s:\n\n%s\n", action.Filename, string(content))
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", action.Operation)
	}
}

func handleExcelOperation(action Action) error {
	switch action.Operation {
	case "read":
		f, err := excelize.OpenFile(action.Filename)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer f.Close()

		for _, a := range action.Actions {
			if a.Type != "read_sheet" {
				continue
			}

			rows, err := f.GetRows(a.Sheet)
			if err != nil {
				return fmt.Errorf("error reading sheet %s: %v", a.Sheet, err)
			}

			fmt.Printf("\nReading sheet '%s' from %s:\n\n", a.Sheet, action.Filename)
			for i, row := range rows {
				fmt.Printf("Row %d: %v\n", i+1, row)
			}
		}
		return nil

	case "create":
		f := excelize.NewFile()
		defer f.Close()

		for _, a := range action.Actions {
			switch a.Type {
			case "create_sheet":
				if a.Sheet != "Sheet1" { // Sheet1 exists by default
					_, err := f.NewSheet(a.Sheet)
					if err != nil {
						return fmt.Errorf("error creating sheet %s: %v", a.Sheet, err)
					}
				}
			case "set_cell":
				err := f.SetCellValue(a.Sheet, a.Cell, a.Value)
				if err != nil {
					return fmt.Errorf("error setting cell %s in sheet %s: %v", a.Cell, a.Sheet, err)
				}
			case "add_row":
				if len(a.Row) > 0 {
					// Convert each value in the row to the appropriate type
					var rowInterface []interface{}
					for _, val := range a.Row {
						// Try to convert string numbers to float64
						if f, err := strconv.ParseFloat(val, 64); err == nil {
							rowInterface = append(rowInterface, f)
							continue
						}
						// Try to convert string "true"/"false" to bool
						if b, err := strconv.ParseBool(val); err == nil {
							rowInterface = append(rowInterface, b)
							continue
						}
						// Keep as string if no conversion possible
						rowInterface = append(rowInterface, val)
					}

					rows, err := f.GetRows(a.Sheet)
					if err != nil {
						return fmt.Errorf("error getting rows from sheet %s: %v", a.Sheet, err)
					}
					err = f.SetSheetRow(a.Sheet, fmt.Sprintf("A%d", len(rows)+1), &rowInterface)
					if err != nil {
						return fmt.Errorf("error adding row to sheet %s: %v", a.Sheet, err)
					}
				}
			}
		}

		if err := f.SaveAs(action.Filename); err != nil {
			return fmt.Errorf("error saving file: %v", err)
		}

		fmt.Printf("\nExcel file created: %s\n", action.Filename)
		return nil

	case "edit":
		f, err := excelize.OpenFile(action.Filename)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer f.Close()

		for _, a := range action.Actions {
			switch a.Type {
			case "create_sheet":
				_, err := f.NewSheet(a.Sheet)
				if err != nil {
					return fmt.Errorf("error creating sheet %s: %v", a.Sheet, err)
				}
			case "set_cell":
				err := f.SetCellValue(a.Sheet, a.Cell, a.Value)
				if err != nil {
					return fmt.Errorf("error setting cell %s in sheet %s: %v", a.Cell, a.Sheet, err)
				}
			case "add_row":
				if len(a.Row) > 0 {
					// Convert each value in the row to the appropriate type
					var rowInterface []interface{}
					for _, val := range a.Row {
						// Try to convert string numbers to float64
						if f, err := strconv.ParseFloat(val, 64); err == nil {
							rowInterface = append(rowInterface, f)
							continue
						}
						// Try to convert string "true"/"false" to bool
						if b, err := strconv.ParseBool(val); err == nil {
							rowInterface = append(rowInterface, b)
							continue
						}
						// Keep as string if no conversion possible
						rowInterface = append(rowInterface, val)
					}

					rows, err := f.GetRows(a.Sheet)
					if err != nil {
						return fmt.Errorf("error getting rows from sheet %s: %v", a.Sheet, err)
					}
					err = f.SetSheetRow(a.Sheet, fmt.Sprintf("A%d", len(rows)+1), &rowInterface)
					if err != nil {
						return fmt.Errorf("error adding row to sheet %s: %v", a.Sheet, err)
					}
				}
			}
		}

		if err := f.Save(); err != nil {
			return fmt.Errorf("error saving changes: %v", err)
		}

		fmt.Printf("\nChanges saved to: %s\n", action.Filename)
		return nil

	default:
		return fmt.Errorf("unknown operation: %s", action.Operation)
	}
}

func main() {
	// Get API key using our config package
	apiKey, err := config.GetAnthropicAPIKey()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Initialize the client
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	// Initial workspace indexing
	if err := indexWorkspace(); err != nil {
		fmt.Printf("Warning: Error indexing workspace files: %v\n", err)
	}

	// Print welcome message
	fmt.Println(welcomeMessage)

	// Initialize conversation history
	messages := []anthropic.MessageParam{}

	// Create a scanner for user input
	scanner := bufio.NewScanner(os.Stdin)

	// Main chat loop
	for {
		// Print prompt and get user input
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		// Handle commands
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/exit":
				fmt.Println("Goodbye!")
				return
			case "/clear":
				messages = []anthropic.MessageParam{}
				fmt.Println("Conversation history cleared.")
				continue
			case "/help":
				fmt.Println(welcomeMessage)
				continue
			case "/index":
				if err := indexWorkspace(); err != nil {
					fmt.Printf("Error indexing workspace files: %v\n", err)
				} else {
					fmt.Println("Workspace files indexed successfully.")
				}
				continue
			default:
				fmt.Printf("Unknown command: %s\n", input)
				continue
			}
		}

		// Add user message to history
		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(input)))

		// Create workspace information for system prompt
		var workspaceInfo strings.Builder
		for _, file := range workspaceFiles {
			if file.IsDir {
				workspaceInfo.WriteString(fmt.Sprintf("\n- Directory: %s\n", file.Path))
			} else {
				if file.Language == "Excel" {
					workspaceInfo.WriteString(fmt.Sprintf("\n- Excel file: %s (Modified: %s)\n",
						file.Path,
						file.ModTime.Format("2006-01-02 15:04:05")))
					if len(file.SheetNames) > 0 {
						workspaceInfo.WriteString("  Sheets:\n")
						for _, sheet := range file.SheetNames {
							rows := file.RowCount[sheet]
							workspaceInfo.WriteString(fmt.Sprintf("    - %s (%d rows)\n", sheet, rows))
						}
					}
				} else {
					workspaceInfo.WriteString(fmt.Sprintf("\n- File: %s (Type: %s, Modified: %s)\n",
						file.Path,
						file.Language,
						file.ModTime.Format("2006-01-02 15:04:05")))
				}
			}
		}

		// Create streaming request with dynamic system prompt
		stream := client.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
			MaxTokens: anthropic.F(int64(1024)),
			Messages:  anthropic.F(messages),
			System: anthropic.F([]anthropic.TextBlockParam{
				anthropic.NewTextBlock(fmt.Sprintf(systemPrompt, workspaceInfo.String())),
			}),
		})

		// Print assistant's response and accumulate it
		fmt.Print("\nClaude: ")
		var fullResponse strings.Builder
		message := anthropic.Message{}

		for stream.Next() {
			event := stream.Current()
			message.Accumulate(event)

			switch delta := event.Delta.(type) {
			case anthropic.ContentBlockDeltaEventDelta:
				if delta.Text != "" {
					fmt.Print(delta.Text)
					fullResponse.WriteString(delta.Text)
				}
			}
		}

		if err := stream.Err(); err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}

		// Add assistant's response to conversation history
		messages = append(messages, message.ToParam())

		// Try to parse response as action
		response := fullResponse.String()
		if strings.Contains(response, `"operation"`) {
			// Find all JSON objects in the response
			var actions []Action
			var currentPos int
			responseLen := len(response)

			for currentPos < responseLen {
				// Find opening brace of JSON object
				startPos := strings.Index(response[currentPos:], "{")
				if startPos == -1 {
					break
				}
				startPos += currentPos

				// Find matching closing brace
				braceCount := 1
				inQuote := false
				inEscape := false
				endPos := startPos + 1

				for endPos < responseLen && braceCount > 0 {
					char := response[endPos]

					if inEscape {
						inEscape = false
					} else {
						switch char {
						case '\\':
							inEscape = true
						case '"':
							inQuote = !inQuote
						case '{':
							if !inQuote {
								braceCount++
							}
						case '}':
							if !inQuote {
								braceCount--
							}
						}
					}
					endPos++
				}

				if braceCount == 0 {
					// Extract and parse the JSON object
					jsonStr := response[startPos:endPos]
					var action Action
					if err := json.Unmarshal([]byte(jsonStr), &action); err == nil {
						// Validate the action
						if action.Operation != "" && action.Filename != "" {
							if (action.Operation == "create" || action.Operation == "edit") &&
								(action.Content != "" || len(action.Actions) > 0) {
								actions = append(actions, action)
								fmt.Printf("\nFound valid action for file: %s\n", action.Filename)
							} else if action.Operation == "read" {
								actions = append(actions, action)
								fmt.Printf("\nFound valid read action for file: %s\n", action.Filename)
							}
						}
					} else {
						fmt.Printf("\nError parsing JSON: %v\n", err)
					}
					currentPos = endPos
				} else {
					currentPos = startPos + 1
				}
			}

			// Execute all found actions
			if len(actions) > 0 {
				fmt.Printf("\nFound %d file operations to execute.\n", len(actions))

				for _, action := range actions {
					// Print operation summary
					switch action.Operation {
					case "create":
						if strings.HasSuffix(strings.ToLower(action.Filename), ".xlsx") {
							fmt.Printf("\nPreparing to create Excel file: %s with %d sheet operations\n",
								action.Filename, len(action.Actions))
						} else {
							fmt.Printf("\nPreparing to create file: %s\n", action.Filename)
						}
					case "edit":
						fmt.Printf("\nPreparing to edit file: %s\n", action.Filename)
					case "read":
						fmt.Printf("\nPreparing to read file: %s\n", action.Filename)
					}

					// Handle the operation with confirmation
					if err := handleOperation(action); err != nil {
						fmt.Printf("Error performing operation: %v\n", err)
					} else {
						fmt.Printf("Successfully handled operation for %s\n", action.Filename)
					}
				}

				// Reindex after operations
				if err := indexWorkspace(); err != nil {
					fmt.Printf("Warning: Error reindexing workspace files: %v\n", err)
				}
			} else {
				fmt.Printf("\nNo valid file operations found in the response.\n")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}
}
