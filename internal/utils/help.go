package utils

import (
	"encoding/json"
	"fmt"
)

type HelpContent struct {
	Usage         string
	Description   string
	Arguments     []ArgHelp
	InputJSON     interface{} // Can be a string like "Schema: <SchemaName:L1>" or a map
	OutputJSON    interface{} // Can be a string like "Schema: <SchemaName:L1>" or a map
	ErrorExamples interface{} // Example or schema name
	AuthRequired  bool        // New field
}

type ArgHelp struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Default     string // Optional default value as string
}

func PrintFormattedHelp(hc HelpContent) {
	fmt.Println("Usage:", hc.Usage) // Usage string from generator already includes [REQUIRES AUTH] if needed

	fmt.Println("\nDescription:")
	fmt.Println(" ", hc.Description)
	if hc.AuthRequired {
		fmt.Println("  NOTE: This command requires authentication with the API.")
	}

	if len(hc.Arguments) > 0 {
		fmt.Println("\nArguments:")
		// Calculate max length for alignment
		maxLenName := 0
		maxLenType := 0
		for _, arg := range hc.Arguments {
			if len(arg.Name) > maxLenName {
				maxLenName = len(arg.Name)
			}
			if len(arg.Type) > maxLenType {
				maxLenType = len(arg.Type)
			}
		}

		for _, arg := range hc.Arguments {
			reqStr := ""
			if arg.Required {
				reqStr = "(required)"
			}
			defaultStr := ""
			if arg.Default != "" {
				defaultStr = fmt.Sprintf("(default: %s)", arg.Default)
			}
			// Adjusted formatting for better alignment
			fmt.Printf("  --%-*s %-*s %s %s %s\n",
				maxLenName, arg.Name,
				maxLenType+2, fmt.Sprintf("<%s>", arg.Type), // +2 for <>
				arg.Description,
				reqStr,
				defaultStr)
		}
	}
	fmt.Println("\nUse -h for JSON help, -hh for this human-readable help.")
}

func processSchemaForHelp(schema interface{}, level int, maxLevel int) interface{} {
	if level > maxLevel {
		return fmt.Sprintf("<Nested data: L%d+>", level)
	}
	if s, ok := schema.(string); ok {
		return s
	}
	if m, ok := schema.(map[string]interface{}); ok {
		simplified := make(map[string]interface{})
		for k, v := range m {
			if subMap, okSub := v.(map[string]interface{}); okSub {
				simplified[k] = processSchemaForHelp(subMap, level+1, maxLevel)
			} else if subArray, okArr := v.([]interface{}); okArr && len(subArray) > 0 {
				simplified[k] = []interface{}{processSchemaForHelp(subArray[0], level+1, maxLevel)}
			} else {
				simplified[k] = fmt.Sprintf("<%T>", v)
			}
		}
		return simplified
	}
	return schema
}

func PrintJSONHelp(hc HelpContent) {
	output := make(map[string]interface{})
	output["description"] = hc.Description
	output["usage"] = hc.Usage
	output["authentication_required"] = hc.AuthRequired // Add auth info to JSON help

	argsInfo := make([]map[string]interface{}, len(hc.Arguments))
	for i, arg := range hc.Arguments {
		argsInfo[i] = map[string]interface{}{
			"name":        arg.Name,
			"type":        arg.Type,
			"required":    arg.Required,
			"description": arg.Description,
		}
		if arg.Default != "" {
			argsInfo[i]["default"] = arg.Default
		}
	}
	if len(argsInfo) > 0 {
		output["arguments"] = argsInfo
	} else {
		output["arguments"] = "None"
	}

	output["expected_input_schema"] = processSchemaForHelp(hc.InputJSON, 1, 1)
	output["expected_output_schema"] = processSchemaForHelp(hc.OutputJSON, 1, 1)
	output["error_examples"] = hc.ErrorExamples

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Error generating JSON help:", err)
		return
	}
	fmt.Println(string(jsonData))
}

func CheckHelpFlags(args []string) (showJSONHelp bool, showTextHelp bool, remainingArgs []string) {
	for _, arg := range args {
		if arg == "-h" {
			showJSONHelp = true
		} else if arg == "-hh" {
			showTextHelp = true
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}
	if showTextHelp { // if -hh is present, it overrides -h for the direct command line parsing
		showJSONHelp = false
	}
	return
}
