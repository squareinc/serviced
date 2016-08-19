// Copyright 2016 The Serviced Authors.:
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package statsapi

import (
	"fmt"
)

func init() {
	AddStatInfo("masters", StatInfo{
		Details: []StatDetails{
			{
				StatID:    "mem",
				Label:     "Memory",
				Unit:      "B",
				Threshold: "90%",
			},
			{
				StatID:    "cpu",
				Label:     "CPU",
				Unit:      "pct",
				Threshold: "90%",
			},
			{
				StatID:    "docker_storage",
				Label:     "Docker Storage",
				Unit:      "B",
				Threshold: "90%",
			},
			{
				StatID:    "dfs_storage",
				Label:     "DFS Storage",
				Unit:      "B",
				Threshold: "90%",
			},
		},
		Fetch: mastersStatFetcher,
	})
}

func mastersStatFetcherMock(sr *StatRequest, info *StatInfo) (results []StatResult, err error) {
	entity := "masters"
	details := info.Details

	for _, stat := range sr.Stats {
		// if detailErr, create results for each
		// EntityID anyway, just make it an "error" result
		detail, detailErr := GetStatDetail(details, stat)

		for _, id := range sr.EntityIDs {

			// TODO - go somewhere and fetch values, capacity
			values := []int{40, 27, 27, 34, 40, 90, 89, 50, 40, 30}
			capacity := 100

			if detailErr != nil {
				results = append(results, StatResult{
					EntityID: id,
					Stat:     stat,
					Error:    fmt.Sprintf("Invalid stat %s for entity %s", stat, entity),
				})
				continue
			}

			threshold, err := ApplyThreshold(detail.Threshold, capacity)
			if err != nil {
				results = append(results, StatResult{
					EntityID: id,
					Stat:     stat,
					Error:    fmt.Sprintf("Could not apply threshold %s", detail.Threshold),
				})
				continue
			}

			results = append(results, StatResult{
				EntityID:  id,
				Stat:      stat,
				Values:    values,
				Capacity:  capacity,
				Threshold: threshold,
			})
		}
	}
	return results, nil
}

// Build Query Service query from StatRequest to retrieve the requested stats for the
// cc master host
func mastersStatFetcher(sr *StatRequest, info *StatInfo) (results []StatResult, err error) {

	// the query service client is accessible in sr.QueryServiceClient) for now,
	// we need to decide if we want to pass it to the fetcher methods or if we should have
	// a StatFetcher struct that contains a ref to the client and also has methods for each
	// type of stats

	return []StatResult{}, nil
}