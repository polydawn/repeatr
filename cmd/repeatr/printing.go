package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/polydawn/refmt/json"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
)

type printer interface {
	printLog(repeatr.Event_Log)
	printOutput(repeatr.Event_Output)
	printResult(repeatr.Event_Result)
}

var (
	_ printer = ansi{}
	_ printer = jsonPrinter{}
)

type ansi struct{ stdout, stderr io.Writer }

var (
	logFlare       = []byte("\033[0;36m-⟩ \033[0m")
	outputFlare    = []byte("\033[0;35m≡⟩ \033[0m")
	runrecordFlare = []byte("\033[0;33m∴⟩ \033[0m")
	colorReset     = []byte("\033[0m")
)

func (p ansi) printLog(evt repeatr.Event_Log) {
	msg := bytes.NewBuffer([]byte(logFlare))
	msg.WriteString(fmt.Sprintf("[\033[1;30m%v\033[0m] ", evt.Time.Local().Format("01-02 15:04:05")))
	msg.Write(logFlare)
	msg.WriteString(fmt.Sprintf("%v: ", evt.Level))
	msg.WriteString(fmt.Sprintf("%v", evt.Msg))
	if len(evt.Detail) > 0 {
		msg.Write([]byte("\033[1;30m ---"))
	}
	for i, detail := range evt.Detail {
		msg.WriteString(fmt.Sprintf(" \033[1;34m%s: \033[1;30m%v", detail[0], detail[1]))
		if i < len(evt.Detail)-1 { // add comma for all values except the last
			msg.WriteByte(',')
		}
	}
	msg.Write(colorReset)
	msg.WriteByte('\n')
	msg.WriteTo(p.stderr)
}

func (p ansi) printOutput(evt repeatr.Event_Output) {
	prefix := bytes.NewBuffer(outputFlare)
	prefix.WriteString(fmt.Sprintf("[\033[1;30m%v\033[0m] ", evt.Time.Local().Format("01-02 15:04:05")))
	prefix.Write(outputFlare)
	prefix.WriteByte('\t')
	leftover, _ := write(p.stderr, []byte(evt.Msg), prefix.Bytes(), append(colorReset, '\n'))
	if len(leftover) > 0 {
		write(p.stderr, append(leftover, '\n'), prefix.Bytes(), append(colorReset, '\n'))
	}
}

func (p ansi) printResult(evt repeatr.Event_Result) {
	if evt.Error != nil {
		p.printLog(repeatr.Event_Log{
			Time:   time.Now(),
			Level:  repeatr.LogError,
			Msg:    "failed to evaluate formula",
			Detail: [][2]string{{"error", evt.Error.Error()}},
		})
		return
	}
	rrMsg := bytes.Buffer{}
	if err := json.NewMarshallerAtlased(&rrMsg, api.RepeatrAtlas).Marshal(evt.RunRecord); err != nil {
		p.printLog(repeatr.Event_Log{
			Time:   time.Now(),
			Level:  repeatr.LogError,
			Msg:    "error serializing runrecord",
			Detail: [][2]string{{"error", err.Error()}},
		})
		return
	}
	msg := bytes.NewBuffer(runrecordFlare)
	msg.WriteString(fmt.Sprintf("[\033[1;30m%v\033[0m] ", time.Now().Local().Format("01-02 15:04:05")))
	msg.WriteString("\033[0;33mrunrecord follows:\033[0m\n")
	msg.WriteTo(p.stderr)
	rrMsg.WriteTo(p.stdout)
}

func write(w io.Writer, msg, prefix, suffix []byte) (leftover []byte, err error) {
	for len(msg) > 0 { // loop until the buffer is exhausted, or another cond breaks out
		adv, tok, err := bufio.ScanLines(msg, false)
		if err != nil {
			return msg, err
		}
		if adv == 0 { // when we no longer have a full chunk, return
			return msg, nil
		}
		w.Write(prefix)
		w.Write(tok)
		w.Write(suffix)
		msg = msg[adv:]
	}
	return []byte{}, nil
}

type jsonPrinter struct{ stdout io.Writer }

func (p jsonPrinter) printLog(evt repeatr.Event_Log) {
	if err := json.NewMarshallerAtlased(p.stdout, repeatr.Atlas).Marshal(repeatr.Event{Log: &evt}); err != nil {
		panic(err)
	}
	p.stdout.Write([]byte{'\n'})
}

func (p jsonPrinter) printOutput(evt repeatr.Event_Output) {
	if err := json.NewMarshallerAtlased(p.stdout, repeatr.Atlas).Marshal(repeatr.Event{Output: &evt}); err != nil {
		panic(err)
	}
	p.stdout.Write([]byte{'\n'})
}

func (p jsonPrinter) printResult(evt repeatr.Event_Result) {
	if err := json.NewMarshallerAtlased(p.stdout, repeatr.Atlas).Marshal(repeatr.Event{Result: &evt}); err != nil {
		panic(err)
	}
	p.stdout.Write([]byte{'\n'})
}
