//go:build darwin

package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Event struct {
	Agent   string
	T       time.Time
	Kind    string
	Outcome string
	Chip    string
	Desc    string
}

type swarmFile struct {
	Author    string `json:"author_org_name"`
	CreatedAt int64  `json:"created_at"`
	Desc      string `json:"description"`
	KeyName   string `json:"key_name"`
	Status    string `json:"status"`
}

func loadSwarm(root string) ([]Event, error) {
	var events []Event
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		if filepath.Base(path) == "index.jsonl" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		var f swarmFile
		if jsonErr := json.Unmarshal(raw, &f); jsonErr != nil {
			return nil
		}
		if f.Author == "" || f.CreatedAt == 0 {
			return nil
		}
		chip, typeSeg := chipAndTypeFromKey(f.KeyName)
		events = append(events, Event{
			Agent:   f.Author,
			T:       time.Unix(f.CreatedAt, 0),
			Kind:    kindFromTypeSegment(typeSeg),
			Outcome: strings.ToLower(f.Status),
			Chip:    chip,
			Desc:    f.Desc,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool { return events[i].T.Before(events[j].T) })
	return events, nil
}

func chipAndTypeFromKey(key string) (chip, typeSeg string) {
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		return "", ""
	}
	return parts[0] + "/" + parts[1], parts[2]
}

func kindFromTypeSegment(seg string) string {
	switch seg {
	case "results":
		return "result"
	case "insights":
		return "insight"
	case "hypotheses":
		return "claim"
	case "best":
		return "best"
	case "baseline":
		return "baseline"
	default:
		return seg
	}
}

func uniqueAgents(events []Event) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range events {
		if _, ok := seen[e.Agent]; ok {
			continue
		}
		seen[e.Agent] = struct{}{}
		out = append(out, e.Agent)
	}
	sort.Strings(out)
	return out
}

func timeBounds(events []Event) (start, end time.Time) {
	if len(events) == 0 {
		now := time.Now()
		return now.Add(-time.Hour), now
	}
	start = events[0].T
	end = events[0].T
	for _, e := range events[1:] {
		if e.T.Before(start) {
			start = e.T
		}
		if e.T.After(end) {
			end = e.T
		}
	}
	return start, end
}
