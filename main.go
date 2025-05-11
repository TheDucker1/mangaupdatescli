package main

//go:generate go run tools/helpcodegen/main.go --spec openapi.yaml --outdir_root cmd --clean

import (
	"fmt"
	"mangaupdatescli/cmd/authors"
	"mangaupdatescli/cmd/categories"
	"mangaupdatescli/cmd/genre"
	"mangaupdatescli/cmd/groups"
	"mangaupdatescli/cmd/misc"
	"mangaupdatescli/cmd/publishers"
	"mangaupdatescli/cmd/releases"
	"mangaupdatescli/cmd/series"
	"os"
)

func printTopLevelHelp() {
	fmt.Println("MangaUpdates API CLI Tool")
	fmt.Println("Usage: mangaupdatescli <subprogram> <command> [arguments...]")
	fmt.Println("\nAvailable Subprograms:")
	fmt.Println("  authors")
	fmt.Println("  categories")
	fmt.Println("  genre")
	fmt.Println("  groups")
	fmt.Println("  misc")
	fmt.Println("  publishers")
	fmt.Println("  releases")
	fmt.Println("  series")
	fmt.Println("\nUse 'mangaupdatescli <subprogram> -h' or '-hh' for command list and descriptions of a subprogram.")
	fmt.Println("Use 'mangaupdatescli <subprogram> <command> -h' for JSON help on a specific command.")
	fmt.Println("Use 'mangaupdatescli <subprogram> <command> -hh' for human-readable help on a specific command.")
}

func main() {
	if len(os.Args) < 2 {
		printTopLevelHelp()
		os.Exit(1)
	}

	subprogram := os.Args[1]

	// Handle top-level help explicitly
	if subprogram == "help" || subprogram == "--help" || subprogram == "-help" {
		printTopLevelHelp()
		return
	}

	var command string
	var actualArgs []string

	if len(os.Args) > 2 {
		// Check if the second argument to the subprogram is a help flag
		if os.Args[2] == "-h" || os.Args[2] == "-hh" || os.Args[2] == "help" {
			command = "help"      // Treat as a request for subprogram-level help
			if len(os.Args) > 3 { // if there are more args, they are for the specific command's help
				command = os.Args[3]
				if len(os.Args) > 4 {
					actualArgs = os.Args[4:]
				}
				if os.Args[2] == "-h" { // Prepend -h for the command handler
					actualArgs = append([]string{"-h"}, actualArgs...)
				} else if os.Args[2] == "-hh" { // Prepend -hh
					actualArgs = append([]string{"-hh"}, actualArgs...)
				}
			}
		} else {
			command = os.Args[2]
			if len(os.Args) > 3 {
				actualArgs = os.Args[3:]
			}
		}
	} else {
		// If only 'mangaupdatescli misc' is called, default to showing subprogram help
		command = "help"
	}

	implicitJsonHelp := len(os.Args) <= 2 || os.Args[2] == "-h"

	switch subprogram {
	case "authors":
		if command == "help" && len(actualArgs) == 0 {
			authors.PrintAuthorsSubprogramHelp(implicitJsonHelp)
			return
		}
		authors.HandleCommand(command, actualArgs)
	case "categories":
		if command == "help" && len(actualArgs) == 0 {
			categories.PrintCategoriesSubprogramHelp(implicitJsonHelp)
			return
		}
		categories.HandleCommand(command, actualArgs)
	case "genre":
		if command == "help" && len(actualArgs) == 0 {
			genre.PrintGenreSubprogramHelp(implicitJsonHelp)
			return
		}
		genre.HandleCommand(command, actualArgs)
	case "groups":
		if command == "help" && len(actualArgs) == 0 {
			groups.PrintGroupsSubprogramHelp(implicitJsonHelp)
			return
		}
		groups.HandleCommand(command, actualArgs)
	case "misc":
		if command == "help" && len(actualArgs) == 0 { // e.g. ./mangaupdatescli misc -h
			misc.PrintMiscSubprogramHelp(implicitJsonHelp) // Pass true if JSON help requested
			return
		}
		misc.HandleCommand(command, actualArgs)
	case "publishers":
		if command == "help" && len(actualArgs) == 0 { // e.g. ./mangaupdatescli misc -h
			publishers.PrintPublishersSubprogramHelp(implicitJsonHelp) // Pass true if JSON help requested
			return
		}
		publishers.HandleCommand(command, actualArgs)
	case "releases":
		if command == "help" && len(actualArgs) == 0 { // e.g. ./mangaupdatescli misc -h
			releases.PrintReleasesSubprogramHelp(implicitJsonHelp) // Pass true if JSON help requested
			return
		}
		releases.HandleCommand(command, actualArgs)
	case "series":
		if command == "help" && len(actualArgs) == 0 { // e.g. ./mangaupdatescli misc -h
			series.PrintSeriesSubprogramHelp(implicitJsonHelp) // Pass true if JSON help requested
			return
		}
		series.HandleCommand(command, actualArgs)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subprogram: %s\n", subprogram)
		printTopLevelHelp()
		os.Exit(1)
	}
}
