package pages

import "os"

func selfUID() int { return os.Getuid() }
