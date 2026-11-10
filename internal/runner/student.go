package runner

import "strings"

func shellArray(name string, files []string) string {
	var b strings.Builder
	b.WriteString(name + "=(")
	for _, file := range files {
		b.WriteString(shellQuote(file) + " ")
	}
	b.WriteString(")\n")
	return b.String()
}
