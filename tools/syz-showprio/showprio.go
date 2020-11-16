// Copyright 2019 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// syz-showprio visualizes the call to call priorities from the prog package.
package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/syzkaller/pkg/db"
	"github.com/google/syzkaller/pkg/mgrconfig"
	"github.com/google/syzkaller/pkg/osutil"
	"github.com/google/syzkaller/prog"
)

var (
	flagOS     = flag.String("os", runtime.GOOS, "target os")
	flagArch   = flag.String("arch", runtime.GOARCH, "target arch")
	flagEnable = flag.String("enable", "", "comma-separated list of enabled syscalls")
	flagCorpus = flag.String("corpus", "", "name of the corpus file")
	flagExport = flag.String("csv", "", "name of CSV output for corpus export")
)

func main() {
	flag.Parse()
	target, err := prog.GetTarget(*flagOS, *flagArch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	if *flagEnable == "" {
		fmt.Fprintf(os.Stderr, "no syscalls enabled")
		os.Exit(1)
	}
	enabled := strings.Split(*flagEnable, ",")
	_, err = mgrconfig.ParseEnabledSyscalls(target, enabled, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse enabled syscalls: %v", err)
		os.Exit(1)
	}
	fmt.Println(*flagCorpus)
	corpus, err := db.ReadCorpus(*flagCorpus, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read corpus: %v", err)
		os.Exit(1)
	}
	if *flagExport != "" {
		exportSyscalls(*flagExport, corpus)
	}
	showPriorities(enabled, target.CalculatePriorities(corpus), target)
}

func showPriorities(calls []string, prios [][]float32, target *prog.Target) {
	printLine(append([]string{"CALLS"}, calls...))
	for _, callRow := range calls {
		line := []string{callRow}
		for _, callCol := range calls {
			val := prios[target.SyscallMap[callRow].ID][target.SyscallMap[callCol].ID]
			line = append(line, fmt.Sprintf("%.2f", val))
		}
		printLine(line)
	}
}

func printLine(values []string) {
	fmt.Printf("|")
	for _, val := range values {
		fmt.Printf("%-20v|", val)
	}
	fmt.Printf("\n")
}

func exportSyscalls(csvname string, corpus []*prog.Prog) {
	w := new(bytes.Buffer)
	writer := csv.NewWriter(w)
	defer writer.Flush()
	var data [][]string
	for pidx, p := range corpus {
		callArgVal, callRetVal := p.SerializeArgRet(false)
		for cidx, call := range p.Calls {
			m := call.Meta
			data = append(data, []string{
				strconv.Itoa(pidx),
				strconv.Itoa(cidx),
				strconv.Itoa(m.ID),
				strconv.FormatUint(m.NR, 10),
				m.Name,
				m.CallName,
				callArgVal[cidx],
				callRetVal[cidx],
			})
			fmt.Println(pidx, p.String(), cidx, m.ID, m.NR, m.Name, m.CallName)
		}
	}
	writer.WriteAll(data)
	if err := osutil.WriteFile("corpus.csv", w.Bytes()); err != nil {
		fmt.Printf("%v", err)
	}
}
