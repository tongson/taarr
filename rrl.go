package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

const rrlReset = "\x1b[0000m"
const rrlRed = "\x1b[0031m"
const rrlGreen = "\x1b[0032m"
const rrlYellow = "\x1b[0033m"
const rrlBlue = "\x1b[0034m"
const rrlMagenta = "\x1b[0035m"
const rrlCyan = "\x1b[0036m"
const rrlDefault = "\x1b[0039m"
const rrlWhite = "\x1b[1;37m"

type rrlColor string

func (c *rrlColor) String() string {
	type _color *rrlColor
	return fmt.Sprintf("%v", _color(c))
}

func rrlPaint(color rrlColor, value string) string {
	return fmt.Sprintf("%v%v%v", color, value, rrlReset)
}

func rrlPaintRow(colors []rrlColor, row []string) []string {
	paintedRow := make([]string, len(row))
	for i, v := range row {
		paintedRow[i] = rrlPaint(colors[i], v)
	}
	return paintedRow
}

func rrlPaintUniformly(color rrlColor, row []string) []string {
	colors := make([]rrlColor, len(row))
	for i, _ := range row {
		colors[i] = color
	}
	return rrlPaintRow(colors, row)
}

func rrlPrint(writer io.Writer, line []string) {
	fmt.Fprintln(writer, strings.Join(line, "\t"))
}

func rrlMain() {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, ' ', 0)
	rrl, err := os.Open(cLOG)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Missing `%s` file in the current directory.\n", cLOG)
		os.Exit(1)
	}
	defer rrl.Close()
	hdrs := []string{"ID", "Target", "Initiated", "NS", "Script", "Log", "Len", "Result"}
	rrlPrint(w, rrlPaintUniformly(rrlDefault, hdrs))
	var maxSz int
	scanner := bufio.NewScanner(rrl)
	rrlInfo, err := rrl.Stat()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Unable to open `%s`.\n", cLOG)
		os.Exit(1)
	}
	maxSz = int(rrlInfo.Size())
	buf := make([]byte, 0, maxSz)
	scanner.Buffer(buf, maxSz)
	for scanner.Scan() {
		log := make(map[string]string)
		err := json.Unmarshal(scanner.Bytes(), &log)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Unable to decode `%s`.\n", cLOG)
			os.Exit(1)
		}
		var colors []rrlColor
		switch {
		case log["msg"] == "failed" || log["msg"] == "sigint":
			colors = []rrlColor{rrlRed, rrlRed, rrlRed, rrlRed, rrlRed, rrlRed, rrlRed, rrlRed}
		case log["msg"] == "repaired":
			colors = []rrlColor{rrlCyan, rrlGreen, rrlMagenta, rrlCyan, rrlGreen, rrlBlue, rrlMagenta, rrlYellow}
		default:
			colors = []rrlColor{rrlCyan, rrlGreen, rrlMagenta, rrlCyan, rrlGreen, rrlBlue, rrlMagenta, rrlWhite}
		}
		if log["duration"] != "" {
			rrlPrint(w, rrlPaintRow(colors, []string{
				log["id"],
				log["target"],
				log["start"],
				log["namespace"],
				log["script"],
				"“" + log["task"] + "”",
				log["duration"],
				log["msg"],
			}))
		}
	}
	rrlPrint(w, rrlPaintUniformly(rrlDefault, hdrs))
	_ = w.Flush()
	_ = rrl.Close()
}
