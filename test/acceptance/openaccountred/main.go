// Command openaccountred executes the child-2 acceptance specs through the
// harness-scoped runner and emits the machine-readable RED evidence contract.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const testPattern = `^TestAcceptanceOpenAccount`

type testEvent struct {
	Action string
	Test   string
	Output string
}

type failingTest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type evidence struct {
	Executed        int           `json:"executed"`
	Passed          int           `json:"passed"`
	Failed          int           `json:"failed"`
	Skipped         int           `json:"skipped"`
	Errors          int           `json:"errors"`
	FailingTests    []failingTest `json:"failingTests"`
	IntentRationale string        `json:"intentRationale"`
}

func main() {
	scoped := exec.Command("ai-conductor", "scoped-run", "./test/acceptance", "-run", testPattern, "-count=1")
	scoped.Stdout = os.Stderr
	scoped.Stderr = os.Stderr
	_ = scoped.Run()

	command := exec.Command("go", "test", "-json", "./test/acceptance", "-run", testPattern, "-count=1")
	output, runErr := command.CombinedOutput()
	_, _ = os.Stderr.Write(output)

	result := evidence{
		FailingTests:    make([]failingTest, 0),
		IntentRationale: "The failing running-server form flows prove that account opening and its page rejection placement are not implemented.",
	}
	reasons := make(map[string]string)
	failedNames := make([]string, 0)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		var event testEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Test != "" && event.Output != "" {
			line := strings.TrimSpace(event.Output)
			if strings.Contains(line, ".go:") || strings.Contains(line, "status =") || strings.Contains(line, "Location =") {
				if _, exists := reasons[event.Test]; !exists {
					reasons[event.Test] = line
				}
			}
		}
		switch event.Action {
		case "pass":
			if event.Test != "" && !strings.Contains(event.Test, "/") {
				result.Passed++
			}
		case "fail":
			if event.Test != "" && !strings.Contains(event.Test, "/") {
				result.Failed++
				failedNames = append(failedNames, event.Test)
			}
		case "skip":
			if event.Test != "" && !strings.Contains(event.Test, "/") {
				result.Skipped++
			}
		}
	}
	result.Executed = result.Passed + result.Failed
	if scanner.Err() != nil || (runErr != nil && result.Failed == 0) {
		result.Errors = 1
	}
	for _, name := range failedNames {
		reason := reasons[name]
		if reason == "" {
			reason = "acceptance assertion failed"
		}
		result.FailingTests = append(result.FailingTests, failingTest{Name: name, Reason: reason})
	}

	payload, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ACCEPTANCE_RED_EVIDENCE: {\"executed\":0,\"passed\":0,\"failed\":0,\"skipped\":0,\"errors\":1,\"failingTests\":[],\"intentRationale\":%q}\n", err.Error())
		return
	}
	fmt.Printf("ACCEPTANCE_RED_EVIDENCE: %s\n", payload)
}
