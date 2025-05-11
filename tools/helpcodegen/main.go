// tools/helpcodegen/main.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// --- Structs to model OpenAPI data ---
type OpenAPISpec struct {
	Paths    map[string]PathItem   `yaml:"paths"`
	Security []map[string][]string `yaml:"security"`
}

type PathItem map[string]OperationDetail

type OperationDetail struct {
	Tags        []string              `yaml:"tags"`
	Summary     string                `yaml:"summary"`
	OperationID string                `yaml:"operationId"`
	Parameters  []Parameter           `yaml:"parameters"`
	RequestBody *RequestBody          `yaml:"requestBody"`
	Responses   map[string]Response   `yaml:"responses"`
	Security    []map[string][]string `yaml:"security"`
}

type Parameter struct {
	Name        string          `yaml:"name"`
	In          string          `yaml:"in"`
	Description string          `yaml:"description"`
	Required    bool            `yaml:"required"`
	Schema      ParameterSchema `yaml:"schema"`
}

type ParameterSchema struct {
	Type    string      `yaml:"type"`
	Format  string      `yaml:"format"`
	Default interface{} `yaml:"default"`
	Enum    []string    `yaml:"enum"`
}

type RequestBody struct {
	Description string               `yaml:"description"`
	Required    bool                 `yaml:"required"`
	Content     map[string]MediaType `yaml:"content"`
}

type MediaType struct {
	Schema RefSchema `yaml:"schema"`
}

type RefSchema struct {
	Ref   string     `yaml:"$ref"`
	Type  string     `yaml:"type"`
	Items *RefSchema `yaml:"items"`
}

type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// --- Structs for Go template ---
type HelpGenData struct {
	PackageName  string
	HelpVarName  string
	Usage        string
	Description  string
	Arguments    []ArgHelpGo
	InputJSON    string
	OutputJSON   string
	ErrorExample string
	AuthRequired bool
}

type ArgHelpGo struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Default     string
}

// --- Helper Functions ---
func toCamelCase(snakeStr string, firstCap bool) string {
	components := strings.Split(snakeStr, "_")
	var result strings.Builder
	for i, comp := range components {
		if i == 0 && !firstCap {
			result.WriteString(comp)
		} else {
			if len(comp) > 0 {
				result.WriteString(strings.ToUpper(string(comp[0])))
				result.WriteString(comp[1:])
			}
		}
	}
	return result.String()
}

func getSchemaRefName(schema *RefSchema) string {
	if schema == nil {
		return "interface{}"
	}
	if schema.Ref != "" {
		parts := strings.Split(schema.Ref, "/")
		return parts[len(parts)-1]
	}
	if schema.Type == "array" && schema.Items != nil {
		return "[]" + getSchemaRefName(schema.Items)
	}
	if schema.Type == "" {
		return "object"
	}
	return schema.Type
}

// Template for generating ONLY the var definition
const helpVarTemplate = `
var {{.HelpVarName}} = utils.HelpContent{
	Usage:         {{printf "%q" .Usage}},
	Description:   {{printf "%q" .Description}},
	{{- if .Arguments }}
	Arguments: []utils.ArgHelp{
		{{- range .Arguments}}
		{
			Name:        {{printf "%q" .Name}},
			Type:        {{printf "%q" .Type}},
			Required:    {{.Required}},
			Description: {{printf "%q" .Description}},
			{{- if .Default }}
			Default:     {{printf "%q" .Default}},
			{{- end }}
		},
		{{- end }}
	},
	{{- else}}
	Arguments:   nil,
	{{- end }}
	InputJSON:     {{printf "%q" .InputJSON}},
	OutputJSON:    {{printf "%q" .OutputJSON}},
	ErrorExamples: {{.ErrorExample}},
	AuthRequired:  {{.AuthRequired}},
}
`

func cleanGeneratedFiles(outdirRoot string, subprogramDirs []string) {
	log.Println("Cleaning previously generated files...")
	for _, spDirName := range subprogramDirs {
		dirToClean := filepath.Join(outdirRoot, spDirName)
		files, err := os.ReadDir(dirToClean)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Directory doesn't exist, nothing to clean
			}
			log.Printf("Warning: Could not read directory %s for cleaning: %v", dirToClean, err)
			continue
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), "_generated_help.go") || strings.HasSuffix(f.Name(), "_unformatted.go_error") {
				filePath := filepath.Join(dirToClean, f.Name())
				err := os.Remove(filePath)
				if err != nil {
					log.Printf("Warning: Could not remove file %s: %v", filePath, err)
				} else {
					fmt.Printf("Removed %s\n", filePath)
				}
			}
		}
	}
}

func main() {
	specFile := flag.String("spec", "", "Path to the OpenAPI YAML specification file.")
	outdirRoot := flag.String("outdir_root", "", "Root directory for cmd subprogram packages (e.g., ./cmd).")
	clean := flag.Bool("clean", false, "Clean previously generated files before generating new ones.")
	flag.Parse()

	if *specFile == "" || *outdirRoot == "" {
		log.Fatal("Both --spec and --outdir_root flags are required.")
	}

	yamlFile, err := os.ReadFile(*specFile)
	if err != nil {
		log.Fatalf("Error reading YAML spec file: %v", err)
	}

	var spec OpenAPISpec
	err = yaml.Unmarshal(yamlFile, &spec)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	// Collect all potential subprogram directory names first for cleaning
	potentialSubprogramDirs := make(map[string]bool)
	subprogramOps := make(map[string]map[string]struct {
		Details OperationDetail
		Path    string
		Method  string
	})

	for pathStr, pathItem := range spec.Paths {
		for methodStr, opDetail := range pathItem {
			if opDetail.OperationID == "" {
				// fmt.Printf("Warning: Operation at %s %s is missing operationId. Skipping.\n", strings.ToUpper(methodStr), pathStr)
				continue
			}
			if len(opDetail.Tags) == 0 {
				// fmt.Printf("Warning: OperationId '%s' is missing tags. Skipping.\n", opDetail.OperationID)
				continue
			}
			subprogramName := opDetail.Tags[0]
			potentialSubprogramDirs[subprogramName] = true // Store for cleaning pass

			if _, ok := subprogramOps[subprogramName]; !ok {
				subprogramOps[subprogramName] = make(map[string]struct {
					Details OperationDetail
					Path    string
					Method  string
				})
			}
			subprogramOps[subprogramName][opDetail.OperationID] = struct {
				Details OperationDetail
				Path    string
				Method  string
			}{Details: opDetail, Path: pathStr, Method: methodStr}
		}
	}

	if *clean {
		var dirNames []string
		for dirName := range potentialSubprogramDirs {
			dirNames = append(dirNames, dirName)
		}
		cleanGeneratedFiles(*outdirRoot, dirNames)
	}

	tmpl, err := template.New("helpVar").Parse(helpVarTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	for spName, opsMap := range subprogramOps {
		var allHelpContents bytes.Buffer
		fmt.Fprintf(&allHelpContents, "// Code generated by tools/helpcodegen/main.go; DO NOT EDIT.\n")
		fmt.Fprintf(&allHelpContents, "package %s\n\n", spName)
		fmt.Fprintf(&allHelpContents, "import \"mangaupdatescli/internal/utils\"\n\n")

		var opIDs []string
		for opID := range opsMap {
			opIDs = append(opIDs, opID)
		}
		sort.Strings(opIDs)

		for _, opID := range opIDs {
			opData := opsMap[opID]
			cliCommandName := opID
			authRequired := len(opData.Details.Security) > 0 || (len(opData.Details.Security) == 0 && len(spec.Security) > 0)

			data := HelpGenData{
				PackageName:  spName,
				HelpVarName:  "help" + toCamelCase(opID, true) + "Content",
				Description:  opData.Details.Summary,
				AuthRequired: authRequired,
			}

			usageParts := []string{"mangaupdatescli", spName, cliCommandName}
			if opData.Details.Parameters != nil {
				for _, param := range opData.Details.Parameters {
					paramNameFlag := strings.ReplaceAll(param.Name, "_", "-")
					paramTypeStr := param.Schema.Type
					if param.Schema.Format != "" {
						paramTypeStr += "(" + param.Schema.Format + ")"
					}
					usageParamStr := fmt.Sprintf("--%s <%s>", paramNameFlag, paramTypeStr)
					if !param.Required {
						usageParamStr = "[" + usageParamStr + "]"
					}
					usageParts = append(usageParts, usageParamStr)
					argDefault := ""
					if param.Schema.Default != nil {
						argDefault = fmt.Sprintf("%v", param.Schema.Default)
					}
					data.Arguments = append(data.Arguments, ArgHelpGo{
						Name:        param.Name,
						Type:        paramTypeStr,
						Required:    param.Required,
						Description: param.Description,
						Default:     argDefault,
					})
				}
			}
			data.Usage = strings.Join(usageParts, " ")
			if data.AuthRequired {
				data.Usage += " [REQUIRES AUTH]"
			}

			if opData.Details.RequestBody != nil && opData.Details.RequestBody.Content != nil {
				if rbJSON, ok := opData.Details.RequestBody.Content["application/json"]; ok {
					data.InputJSON = fmt.Sprintf("Request Body Schema: <%s:L1>", getSchemaRefName(&rbJSON.Schema))
					if opData.Details.RequestBody.Required {
						data.InputJSON += " (required)"
					}
				} else if _, okMulti := opData.Details.RequestBody.Content["multipart/form-data"]; okMulti {
					data.InputJSON = "Request Body: Multipart Form Data"
					if opData.Details.RequestBody.Required {
						data.InputJSON += " (required)"
					}
				} else {
					data.InputJSON = "Request Body: Present"
					if opData.Details.RequestBody.Required {
						data.InputJSON += " (required)"
					}
				}
			} else if len(data.Arguments) > 0 {
				data.InputJSON = "Path/Query Parameters"
			} else {
				data.InputJSON = "None"
			}

			var successStatusCode string
			successStatusCodes := []string{"200", "201", "202", "204"}
			for _, code := range successStatusCodes {
				if _, ok := opData.Details.Responses[code]; ok {
					successStatusCode = code
					break
				}
			}
			if successStatusCode != "" {
				respSuccess := opData.Details.Responses[successStatusCode]
				if respSuccess.Content != nil {
					if contentJSON, ok := respSuccess.Content["application/json"]; ok {
						data.OutputJSON = fmt.Sprintf("Schema (on %s): <%s:L1>", successStatusCode, getSchemaRefName(&contentJSON.Schema))
					} else if _, okXml := respSuccess.Content["application/xml"]; okXml {
						data.OutputJSON = fmt.Sprintf("XML Output (on %s)", successStatusCode)
					} else if len(respSuccess.Content) > 0 {
						for contentType := range respSuccess.Content {
							data.OutputJSON = fmt.Sprintf("Output (on %s): %s", successStatusCode, contentType)
							break
						}
					} else if successStatusCode == "204" {
						data.OutputJSON = fmt.Sprintf("No Content (on %s)", successStatusCode)
					} else {
						data.OutputJSON = fmt.Sprintf("Success Response (on %s, content type unspecified or empty)", successStatusCode)
					}
				} else {
					data.OutputJSON = fmt.Sprintf("Success Response (on %s, no content defined)", successStatusCode)
				}
			} else {
				data.OutputJSON = "Schema: <ApiResponseV1:L1> (Default success, or specific success code)"
			}

			errorMap := make(map[string]string)
			for code, respInfo := range opData.Details.Responses {
				if strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") {
					errDesc := respInfo.Description
					var schemaName string
					if respInfo.Content != nil {
						if contentJSON, ok := respInfo.Content["application/json"]; ok {
							schemaName = getSchemaRefName(&contentJSON.Schema)
						}
					}
					if schemaName != "" && schemaName != "ApiResponseV1" {
						errorMap[code] = fmt.Sprintf("%s (Schema: <%s:L1>)", errDesc, schemaName)
					} else {
						errorMap[code] = errDesc
					}
				}
			}
			if len(errorMap) > 0 {
				var b bytes.Buffer
				b.WriteString("map[string]string{")
				keys := make([]string, 0, len(errorMap))
				for k := range errorMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for i, k := range keys {
					fmt.Fprintf(&b, "%q: %q", k, errorMap[k])
					if i < len(keys)-1 {
						b.WriteString(", ")
					}
				}
				b.WriteString("}")
				data.ErrorExample = b.String()
			} else {
				data.ErrorExample = `map[string]string{"Generic": "Standard API errors."}`
			}

			var singleVarBuffer bytes.Buffer
			if err := tmpl.Execute(&singleVarBuffer, data); err != nil {
				log.Fatalf("Error executing template for %s (%s): %v", opID, spName, err)
			}
			allHelpContents.Write(singleVarBuffer.Bytes())
			allHelpContents.WriteString("\n")
		}

		if len(opIDs) > 0 {
			outputFilePath := filepath.Join(*outdirRoot, spName, fmt.Sprintf("%s_generated_help.go", spName))
			os.MkdirAll(filepath.Dir(outputFilePath), 0755)
			formattedBytes, err := format.Source(allHelpContents.Bytes())
			if err != nil {
				log.Printf("Error formatting Go source for %s (writing unformatted to %s_unformatted.go_error): %v", spName, outputFilePath, err)
				os.WriteFile(outputFilePath+"_unformatted.go_error", allHelpContents.Bytes(), 0644)
			} else {
				if err := os.WriteFile(outputFilePath, formattedBytes, 0644); err != nil {
					log.Fatalf("Error writing Go file %s: %v", outputFilePath, err)
				}
			}
			fmt.Printf("Generated help for %s at %s\n", spName, outputFilePath)
		} else {
			// This case should ideally not be hit if we are generating for all tagged operations
			// unless a tag exists with no operations or operations without operationIds.
			fmt.Printf("No operations to generate help for in subprogram %s. Skipping file generation.\n", spName)
		}
	}
	fmt.Println("Help code generation complete.")
}
