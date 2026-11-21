// Package runner executes a collection or folder of requests.
package runner

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Postwoman/internal/database"
	"Postwoman/internal/model"

	"github.com/google/uuid"
)

// Executor runs one request with the supplied variable scopes and returns the
// response and the (possibly script-mutated) request.
type Executor func(ctx context.Context, req *model.Request, vars map[string]map[string]string) (*model.Response, *model.Request, error)

// Run executes the selected requests. Progress is reported through onProgress
// which may be nil.
func Run(ctx context.Context, repo *database.Repository, exec Executor, opts model.RunOptions, onProgress func(model.RequestResult)) (*model.RunResult, error) {
	if opts.Iterations <= 0 {
		opts.Iterations = 1
	}
	requests, err := selectRequests(ctx, repo, opts)
	if err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("no requests selected")
	}

	dataRows, err := loadData(opts.DataFile)
	if err != nil {
		return nil, err
	}

	environment := map[string]string{}
	collectionVars := map[string]string{}
	if opts.EnvironmentID != "" {
		if env, err := repo.GetEnvironment(ctx, opts.EnvironmentID); err == nil && env != nil {
			for _, v := range env.Values {
				if v.Enabled {
					environment[v.Key] = v.Value
				}
			}
		}
	}
	if len(requests) > 0 {
		if coll, err := repo.GetCollection(ctx, requests[0].CollectionID); err == nil && coll != nil {
			for _, v := range coll.Variables {
				if v.Enabled {
					collectionVars[v.Key] = v.Value
				}
			}
		}
	}
	globals := map[string]string{}
	if gvars, err := repo.ListGlobalVariables(ctx); err == nil {
		for _, v := range gvars {
			if v.Enabled {
				globals[v.Key] = v.Value
			}
		}
	}

	result := &model.RunResult{
		ID:            uuid.New().String(),
		CollectionID:  opts.CollectionID,
		StartedAt:     time.Now().UnixMilli(),
		Status:        "running",
		TotalRequests: len(requests) * opts.Iterations,
		Iterations:    []model.IterationResult{},
	}

	stopped := false
	for iteration := 0; iteration < opts.Iterations && !stopped; iteration++ {
		if err := ctx.Err(); err != nil {
			result.Status = "cancelled"
			result.Error = "run cancelled"
			break
		}
		data := map[string]string{}
		if len(dataRows) > 0 {
			data = dataRows[iteration%len(dataRows)]
		}
		iter := model.IterationResult{
			Iteration: iteration,
			StartedAt: time.Now().UnixMilli(),
			Requests:  []model.RequestResult{},
		}
		for idx := range requests {
			if err := ctx.Err(); err != nil {
				stopped = true
				result.Status = "cancelled"
				break
			}
			req := requests[idx]
			vars := map[string]map[string]string{
				"local":       {},
				"data":        data,
				"environment": environment,
				"collection":  collectionVars,
				"globals":     globals,
			}
			start := time.Now()
			resp, mutated, err := exec(ctx, &req, vars)
			rr := model.RequestResult{
				RequestID:   req.ID,
				RequestName: req.Name,
				Duration:    time.Since(start).Milliseconds(),
			}
			if err != nil {
				rr.Status = "error"
				rr.Error = err.Error()
				result.FailedTests++
			} else {
				rr.Response = resp
				if mutated != nil {
					rr.RequestName = mutated.Name
				}
				rr.Tests = resp.Tests
				passed := true
				for _, t := range resp.Tests {
					if t.Passed {
						result.PassedTests++
					} else {
						result.FailedTests++
						passed = false
					}
					result.TotalTests++
				}
				if resp.Error != "" {
					rr.Status = "error"
					rr.Error = resp.Error
					passed = false
				} else if passed {
					rr.Status = "passed"
				} else {
					rr.Status = "failed"
				}
				// Scripts may mutate variables; carry the changes forward.
				if resp != nil && mutated != nil {
					_ = mutated
				}
			}
			iter.Requests = append(iter.Requests, rr)
			if onProgress != nil {
				onProgress(rr)
			}
			if opts.StopOnError && rr.Status != "passed" {
				stopped = true
				break
			}
			if opts.Delay > 0 && idx < len(requests)-1 {
				select {
				case <-ctx.Done():
					stopped = true
				case <-time.After(time.Duration(opts.Delay) * time.Millisecond):
				}
			}
		}
		iter.CompletedAt = time.Now().UnixMilli()
		result.Iterations = append(result.Iterations, iter)
		if stopped {
			break
		}
	}
	result.CompletedAt = time.Now().UnixMilli()
	if result.Status == "running" {
		if result.FailedTests > 0 {
			result.Status = "failed"
		} else {
			result.Status = "passed"
		}
	}
	return result, nil
}

func selectRequests(ctx context.Context, repo *database.Repository, opts model.RunOptions) ([]model.Request, error) {
	if len(opts.RequestIDs) > 0 {
		out := make([]model.Request, 0, len(opts.RequestIDs))
		for _, id := range opts.RequestIDs {
			req, err := repo.GetRequest(ctx, id)
			if err != nil {
				return nil, err
			}
			if req != nil {
				out = append(out, *req)
			}
		}
		return out, nil
	}
	if opts.CollectionID == "" {
		return nil, fmt.Errorf("collectionId is required")
	}
	all, err := repo.ListRequests(ctx, opts.CollectionID)
	if err != nil {
		return nil, err
	}
	if opts.FolderID == "" {
		return all, nil
	}
	// Include requests in the selected folder and all descendant folders.
	folders, err := repo.ListFolders(ctx, opts.CollectionID)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{opts.FolderID: true}
	changed := true
	for changed {
		changed = false
		for _, f := range folders {
			if f.ParentID != nil && allowed[*f.ParentID] && !allowed[f.ID] {
				allowed[f.ID] = true
				changed = true
			}
		}
	}
	out := []model.Request{}
	for _, r := range all {
		if r.FolderID != nil && allowed[*r.FolderID] {
			out = append(out, r)
		}
	}
	return out, nil
}

func loadData(path string) ([]map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read data file: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".csv" {
		reader := csv.NewReader(strings.NewReader(string(raw)))
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("invalid CSV data file: %w", err)
		}
		if len(records) == 0 {
			return nil, nil
		}
		header := records[0]
		out := make([]map[string]string, 0, len(records)-1)
		for _, row := range records[1:] {
			entry := map[string]string{}
			for i, key := range header {
				if i < len(row) {
					entry[key] = row[i]
				}
			}
			out = append(out, entry)
		}
		return out, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		// Also accept a single object.
		var single map[string]any
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("data file must be a JSON array of objects or a CSV file")
		}
		rows = []map[string]any{single}
	}
	out := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		entry := map[string]string{}
		for k, v := range row {
			switch t := v.(type) {
			case string:
				entry[k] = t
			case nil:
				entry[k] = ""
			default:
				b, _ := json.Marshal(t)
				entry[k] = string(b)
			}
		}
		out = append(out, entry)
	}
	return out, nil
}
