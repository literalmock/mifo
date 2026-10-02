package components

import "strings"

var bigUnicode = map[rune][3]string{
	'0': {"█▀█", "█ █", "▀▀▀"},
	'1': {"▀█ ", " █ ", "▀▀▀"},
	'2': {"▀▀█", "█▀▀", "▀▀▀"},
	'3': {"▀▀█", " ▀█", "▀▀▀"},
	'4': {"█ █", "▀▀█", "  ▀"},
	'5': {"█▀▀", "▀▀█", "▀▀▀"},
	'6': {"█▀▀", "█▀█", "▀▀▀"},
	'7': {"▀▀█", "  █", "  ▀"},
	'8': {"█▀█", "█▀█", "▀▀▀"},
	'9': {"█▀█", "▀▀█", "▀▀▀"},
	'%': {"▀ █", " █ ", "█ ▄"},
	'.': {" ", " ", "▀"},
	' ': {" ", " ", " "},
}

var bigASCII = map[rune][3]string{
	'0': {"###", "# #", "###"},
	'1': {"## ", " # ", "###"},
	'2': {"###", "## ", "###"},
	'3': {"###", " ##", "###"},
	'4': {"# #", "###", "  #"},
	'5': {"###", " ##", "###"},
	'6': {"#  ", "###", "###"},
	'7': {"###", "  #", "  #"},
	'8': {"###", "###", "###"},
	'9': {"###", "###", "  #"},
	'%': {"# #", " # ", "# #"},
	'.': {" ", " ", "#"},
	' ': {" ", " ", " "},
}

// BigText renders s (digits, '%', '.') in a three-row block font.
func BigText(s string, unicode bool) [3]string {
	font := bigASCII
	if unicode {
		font = bigUnicode
	}
	var rows [3]strings.Builder
	for i, r := range s {
		g, ok := font[r]
		if !ok {
			continue
		}
		for j := 0; j < 3; j++ {
			if i > 0 {
				rows[j].WriteByte(' ')
			}
			rows[j].WriteString(g[j])
		}
	}
	return [3]string{rows[0].String(), rows[1].String(), rows[2].String()}
}
