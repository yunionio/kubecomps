package results

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"yunion.io/x/pkg/errors"
)

// AnsiblePlaybookJSONResults
type AnsiblePlaybookJSONResults struct {
	CustomStats       interface{}                                 `json:"custom_stats"`
	GlobalCustomStats interface{}                                 `json:"global_custom_stats"`
	Plays             []AnsiblePlaybookJSONResultsPlay            `json:"plays"`
	Stats             map[string]*AnsiblePlaybookJSONResultsStats `json:"stats"`
}

func (r *AnsiblePlaybookJSONResults) String() string {

	str := ""

	for _, play := range r.Plays {
		for _, task := range play.Tasks {
			name := task.Task.Name
			for host, result := range task.Hosts {
				str = fmt.Sprintf("%s[%s] (%s)	%s\n", str, host, name, result.Msg)
			}
		}
	}

	for host, stats := range r.Stats {
		str = fmt.Sprintf("%s\nHost: %s\n%s\n", str, host, stats.String())
	}

	return str
}

// CheckStats return error when is found a failure or unreachable host
func (r *AnsiblePlaybookJSONResults) CheckStats() error {
	errorMsg := ""
	for host, stats := range r.Stats {
		if stats.Failures > 0 {
			errorMsg = fmt.Sprintf("Host %s finished with %d failures", host, stats.Failures)
		}

		if stats.Unreachable > 0 {
			errorMsg = fmt.Sprintf("Host %s finished with %d unrecheable hosts", host, stats.Unreachable)
		}

		if len(errorMsg) > 0 {
			return errors.Errorf("(results::JSONStdoutCallbackResults): %v", errorMsg)
		}
	}

	return nil
}

// AnsiblePlaybookJSONResultsPlay
type AnsiblePlaybookJSONResultsPlay struct {
	Play  *AnsiblePlaybookJSONResultsPlaysPlay `json:"play"`
	Tasks []AnsiblePlaybookJSONResultsPlayTask `json:"tasks"`
}

// AnsiblePlaybookJSONResultsPlaysPlay
type AnsiblePlaybookJSONResultsPlaysPlay struct {
	Name     string                                  `json:"name"`
	Id       string                                  `json:"id"`
	Duration *AnsiblePlaybookJSONResultsPlayDuration `json:"duration"`
}

/* AnsiblePlaybookJSONResultsPlayTask
'task': {
	'name': task.get_name(),
	'id': to_text(task._uuid),
	'duration': {
		'start': current_time()
	}
},
'hosts': {}
*/
type AnsiblePlaybookJSONResultsPlayTask struct {
	Task  *AnsiblePlaybookJSONResultsPlayTaskItem                 `json:"task"`
	Hosts map[string]*AnsiblePlaybookJSONResultsPlayTaskHostsItem `json:"hosts"`
}

type AnsiblePlaybookJSONResultsPlayTaskHostsItem struct {
	Action       string                 `json:"action"`
	Changed      bool                   `json:"changed"`
	Msg          string                 `json:"msg"`
	AnsibleFacts map[string]interface{} `json:"ansible_facts"`
}

type AnsiblePlaybookJSONResultsPlayTaskItem struct {
	Name     string                                          `json:"name"`
	Id       string                                          `json:"id"`
	Duration *AnsiblePlaybookJSONResultsPlayTaskItemDuration `json:"duration"`
}

type AnsiblePlaybookJSONResultsPlayTaskItemDuration struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AnsiblePlaybookJSONResultsPlayDuration
type AnsiblePlaybookJSONResultsPlayDuration struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// AnsiblePlaybookJSONResultsStats
type AnsiblePlaybookJSONResultsStats struct {
	Changed     int `json:"changed"`
	Failures    int `json:"failures"`
	Ignored     int `json:"ignored"`
	Ok          int `json:"ok"`
	Rescued     int `json:"rescued"`
	Skipped     int `json:"skipped"`
	Unreachable int `json:"unreachable"`
}

func (s *AnsiblePlaybookJSONResultsStats) String() string {
	str := fmt.Sprintf(" Changed: %s", strconv.Itoa(s.Changed))
	str = fmt.Sprintf("%s Failures: %s", str, strconv.Itoa(s.Failures))
	str = fmt.Sprintf("%s Ignored: %s", str, strconv.Itoa(s.Ignored))
	str = fmt.Sprintf("%s Ok: %s", str, strconv.Itoa(s.Ok))
	str = fmt.Sprintf("%s Rescued: %s", str, strconv.Itoa(s.Rescued))
	str = fmt.Sprintf("%s Skipped: %s", str, strconv.Itoa(s.Skipped))
	str = fmt.Sprintf("%s Unreachable: %s", str, strconv.Itoa(s.Unreachable))

	return str
}

// JSONStdoutCallbackResults method manges the ansible' JSON stdout callback and print the result stats
func JSONStdoutCallbackResults(prefix string, r io.Reader, w io.Writer) error {

	if r == nil {
		return errors.Error("(results::JSONStdoutCallbackResults): Reader is not defined")
	}

	if w == nil {
		w = os.Stdout
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !skipLine(line) {
			fmt.Fprintf(w, "%s", line)
		}
	}

	return nil
}

func skipLine(line string) bool {
	skipPatterns := []string{
		// This pattern skips timer's callback whitelist output
		"^[\\s\\t]*Playbook run took [0-9]+ days, [0-9]+ hours, [0-9]+ minutes, [0-9]+ seconds$",
	}

	for _, pattern := range skipPatterns {
		match, _ := regexp.MatchString(pattern, line)
		if match {
			return true
		}
	}

	return false
}

// JSONParse return an AnsiblePlaybookJSONResults from
func JSONParse(data []byte) (*AnsiblePlaybookJSONResults, error) {

	result := &AnsiblePlaybookJSONResults{}

	err := json.Unmarshal(data, result)
	if err != nil {
		return nil, errors.Wrap(err, "(results::JSONParser): Unmarshall error")
	}

	return result, nil
}
