package buildinfo

import "log"

func Print(version string, date string, commit string) {
	if version == "" {
		version = "N/A"
	}
	log.Printf("Build version: %s", version)

	if date == "" {
		date = "N/A"
	}
	log.Printf("Build date: %s", date)

	if commit == "" {
		commit = "N/A"
	}
	log.Printf("Build commit: %s", commit)
}
