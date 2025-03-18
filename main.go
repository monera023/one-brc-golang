package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
)

type stats struct {
	min, max, sum float64
	count         int32
}

type part struct {
	offset, size int64
}

func parse(row string) (string, float64, bool) {
	for p, r := range row {
		if r == ';' {
			station, data := row[:p], row[p+1:]
			temperature, err := strconv.ParseFloat(data, 64)
			if err != nil {
				return "", 0, false
			}
			return station, temperature, true
		}
	}
	return "", 0, false
}

func ProcessPart(filePath string, fileOffset, fileSize int64, resultsCh chan map[string]stats) {
	// Read file..
	f, err := os.Open(filePath)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	_, err = f.Seek(fileOffset, io.SeekStart)

	if err != nil {
		panic(err)
	}

	file := io.LimitedReader{R: f, N: fileSize}

	stationStats := make(map[string]stats)

	scanner := bufio.NewScanner(&file)

	for scanner.Scan() {
		line := scanner.Text()
		// station, tempStr, hasSemi := strings.Cut(line, ";")
		station, temp, hasSemi := parse(line)
		if !hasSemi {
			continue
		}

		// temp, err := strconv.ParseFloat(tempStr, 64)
		// if err != nil {
		// 	panic(err)
		// }

		s, ok := stationStats[station]
		if !ok {
			s.min = temp
			s.max = temp
			s.sum = temp
			s.count = 1
		} else {
			s.min = min(s.min, temp)
			s.max = max(s.max, temp)
			s.sum += temp
			s.count++
		}
		stationStats[station] = s
	}

	resultsCh <- stationStats

}

func main() {
	// Read file..
	f, err := os.Open("measurements.txt")

	if err != nil {
		panic(err)
	}

	defer f.Close()

	f1, err := os.Create("cpuProfile.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f1.Close()
	pprof.StartCPUProfile(f1)
	defer pprof.StopCPUProfile()

	st, err := f.Stat()

	if err != nil {
		fmt.Println("Got error in stat..", err)
	}

	const maxLineLength = 100
	numParts := 10
	parts := make([]part, 0, numParts)
	sz := st.Size()
	fmt.Println("Size of file =", sz)

	splitSize := sz / int64(numParts)
	fmt.Println("size of each split=", splitSize)

	offset := int64(0)

	for offset < sz {
		// Find offset before split end -- move 100 before
		seekOffset := max(offset+splitSize-maxLineLength, 0)

		// Move file to 100 end of split
		f.Seek(int64(seekOffset), io.SeekStart)

		buf := make([]byte, maxLineLength)
		bytesRead, _ := io.ReadFull(f, buf)
		chunk := buf[:bytesRead]

		newLine := bytes.LastIndexAny(chunk, "\n")

		remaining := len(chunk) - newLine - 1
		nextOffset := seekOffset + int64(len(chunk)) - int64(remaining)
		parts = append(parts, part{offset: offset, size: nextOffset - offset})
		offset = nextOffset
	}

	fmt.Println("Finally...", parts)
	resultsCh := make(chan map[string]stats)

	// Processing chunks in different go routine
	for _, part := range parts {
		go ProcessPart("measurements.txt", part.offset, part.size, resultsCh)
	}

	totals := make(map[string]stats)

	for i := 0; i < len(parts); i++ {
		result := <-resultsCh

		for station, stat := range result {
			ts, ok := totals[station]

			if !ok {
				totals[station] = stats{
					min:   stat.min,
					max:   stat.max,
					sum:   stat.sum,
					count: stat.count,
				}
			} else {
				ts.min = min(ts.min, stat.min)
				ts.max = min(ts.max, stat.max)
				ts.sum += stat.sum
				ts.count += stat.count
				totals[station] = ts
			}
		}
	}

	stations := make([]string, 0, len(totals))
	for station := range totals {
		stations = append(stations, station)
	}
	sort.Strings(stations)

	fmt.Print("{")
	for i, station := range stations {
		if i > 0 {
			fmt.Print(", ")
		}
		s := totals[station]
		mean := s.sum / float64(s.count)
		fmt.Printf("%s=%.1f/%.1f/%.1f", station, s.min, mean, s.max)
	}
	fmt.Print("}\n")

}
