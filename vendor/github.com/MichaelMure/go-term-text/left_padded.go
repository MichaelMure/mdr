package text

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// LeftPadMaxLine pads a string on the left by a specified amount and pads the
// string on the right to fill the maxLength
func LeftPadMaxLine(text string, length, leftPad int) string {
	var rightPart = text

	scrWidth := runewidth.StringWidth(text)
	// truncate and ellipse if needed
	if scrWidth+leftPad > length {
		rightPart = runewidth.Truncate(text, length-leftPad, "…")
	} else if scrWidth+leftPad < length {
		rightPart = runewidth.FillRight(text, length-leftPad)
	}

	return fmt.Sprintf("%s%s",
		strings.Repeat(" ", leftPad),
		rightPart,
	)
}

// LeftPad left pad each line of the given text
func LeftPad(text string, leftPad int) string {
	var result bytes.Buffer

	pad := strings.Repeat(" ", leftPad)

	lines := strings.Split(text, "\n")

	for i, line := range lines {
		result.WriteString(pad)
		result.WriteString(line)

		// no additional line break at the end
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
