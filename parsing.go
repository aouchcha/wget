package main

import (
	"errors"
	"fmt"
	"strings"
)

func parsing(args []string, components *FlagsComponents) error {
	flags := []string{"-O", "-B", "-P", "--limit-rate", "--mirror", "-R", "--reject", "-X", "--exclude", "--convert-links", "-i"}

	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "-O") && i <= len(args)-2 {
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
		} else if strings.HasPrefix(args[i], "-i") && i <= len(args)-2 {
			checker, err := CatchInput(args[i:i+2], components, flags)
			if err != nil {
				return err
			}
			if checker {
				i += 2
				continue
			}
		} else if strings.HasPrefix(args[i], "--limit-rate") && i <= len(args)-2 {
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
		} else if strings.HasPrefix(args[i], "-") {
			if !CheckValidFlag(args[i], flags) {
				return fmt.Errorf("invalid flag %s", args[i])
			}
		}else {
			if strings.HasPrefix(args[i], "http") {
				components.Links = append(components.Links, args[i])
			}else {
				components.Links = append(components.Links, fmt.Sprintf("http://%s/",args[i]))
			}
		}
		i += 1
	}
	// if len(components.Links) == 0 {
	// 	return errors.New("you don't provide the program with link to download from it")
	// }
	// fmt.Println(*components)
	return nil
}
