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

const resolution = 100

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run program.go <PID> <IntervalMill>")
		return
	}

	pidArg := os.Args[1]
	intervalArg := os.Args[2]
	id, _ := strconv.Atoi(pidArg)
	intervalArgInt, _ := strconv.Atoi(intervalArg)
	pid = &id
	interval := time.Duration(intervalArgInt) * time.Millisecond

	currentTime := time.Now()
	for i := resolution; i > 0; i-- {
		pair := Pair{
			value: 0,
			time:  currentTime.Add(time.Duration(intervalArgInt*i) * time.Millisecond).Format("15:04:05"),
		}
		arr = append(arr, pair)
	}

	t, err := tcell.New()
	if err != nil {
		panic(err)
	}
	defer t.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// ----------------------------------------------
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

type Pair struct {
	value float64
	time  string
}

var arr []Pair
var pid *int

func GetMemoryUsage(pid int) ([]Pair, error) {
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
				pair := Pair{value: float64(memKb) / 1024, time: time.Now().Format("04:05")}
				if len(arr) > resolution {
					arr = append(arr[1:], pair)
				} else {
					arr = append(arr, pair)
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
			memRecs, _ := GetMemoryUsage(*pid)
			var inputs []float64
			var xItems map[int]string = make(map[int]string)
			for i, v := range memRecs {
				inputs = append(inputs, v.value)
				xItems[i] = v.time
			}

			if err := lc.Series("first", inputs,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
				linechart.SeriesXLabels(xItems),
			); err != nil {
				panic(err)
			}

		case <-ctx.Done():
			return
		}
	}
}
