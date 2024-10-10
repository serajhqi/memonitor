package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"

	"github.com/mum4k/termdash/widgets/linechart"
)

var arr []float64
var pid *int

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run program.go <PID> <IntervalMill>")
		return
	}

	pidArg := os.Args[1]
	intervalArg := os.Args[2]
	id, _ := strconv.Atoi(pidArg)
	intervalMilli, _ := strconv.Atoi(intervalArg)
	pid = &id

	for i := 0; i < 100; i++ {
		arr = append(arr, 0)
	}
	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// ----------------------------------------------
	var interval = time.Duration(intervalMilli) * time.Millisecond
	lc, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
	)
	if err != nil {
		panic(err)
	}
	go playLineChart(ctx, lc, interval)
	// ----------------------------------------------

	c, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("PRESS Q TO QUIT"),
		container.BorderTitle("Mem"),
		container.Border(linestyle.Light),
		container.PlaceWidget(lc),
	)
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}

	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quitter)); err != nil {
		panic(err)
	}
}

func GetMemoryUsage(pid int) ([]float64, error) {
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(statusFile)
	if err != nil {
		return arr, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// VmRSS value is in kB, convert to int
				memKb, err := strconv.Atoi(fields[1])
				if len(arr) > 100 {

					arr = append(arr[1:], float64(memKb)/1024)
				} else {
					arr = append(arr, float64(memKb)/1024)
				}

				if err != nil {
					return arr, err
				}
				return arr, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return arr, err
	}

	return arr, fmt.Errorf("could not find memory usage info")
}

func playLineChart(ctx context.Context, lc *linechart.LineChart, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			inputs, _ := GetMemoryUsage(*pid)

			if err := lc.Series("first", inputs,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
				linechart.SeriesXLabels(map[int]string{
					0: "zero",
				}),
			); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}
