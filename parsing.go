package main

import (
	"errors"
	"fmt"
	"strings"
)

func parsing(args []string, components *FlagsComponents) error {
	flags := []string{"-O", "-B", "-P", "--rate-limit", "--mirror", "-R", "--reject", "-X", "--exclude", "--convert-links", "-i"}

	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "http") || strings.HasPrefix(args[i], "ftp") {
			components.Link = args[i]
		} else if strings.HasPrefix(args[i], "-O") && i <= len(args)-2 {
			checker, err := CatchOutputFile(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-P") && i <= len(args)-2 {
			checker, err := CatchPath(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		}else if strings.HasPrefix(args[i], "-i") && i <= len(args)-2 {
			checker, err := CatchInput(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		}else if strings.HasPrefix(args[i], "--rate-limit") && i <= len(args)-2 {
			checker, err := CatchRate(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "--mirror") {
			if !CheckValidFlag(args[i], flags) {
				return errors.New("invalid flag --mirror")
			}
			components.isMirror = true
		} else if strings.HasPrefix(args[i], "-R") || strings.HasPrefix(args[i], "--reject") && i <= len(args)-2 {
			checker, err := CatchTheRejectedSuffix(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-X") || strings.HasPrefix(args[i], "--exclude") && i <= len(args)-2 {
			checker, err := CatchTheRExcludedFolders(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "-B") {
			if !CheckValidFlag(args[i], flags) {
				return errors.New("invalid flag --B")
			}
			components.Background = true
		} else if strings.HasPrefix(args[i], "--convert-links") {
			if !CheckValidFlag(args[i], flags) {
				return errors.New("invalid flag --convert-links")
			}
			components.Convert = true
		}
		i += 1
	}
	if components.Link == "" {
		return errors.New("you don't provide the program with link to download from it")
	}
	fmt.Println(*components)
	return nil
}
