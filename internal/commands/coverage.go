// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

// resultCoverage separates the inspected upstream page from local matches and
// the returned subset. MatchedCount never estimates matches on unseen pages.
type resultCoverage struct {
	FetchedCount        int  `json:"fetched_count"`
	MatchedCount        int  `json:"matched_count"`
	ReturnedCount       int  `json:"returned_count"`
	ResultTruncated     bool `json:"result_truncated"`
	UpstreamMayHaveMore bool `json:"upstream_may_have_more"`
	UpstreamTotal       *int `json:"upstream_total,omitempty"`
}

func pageCoverage(fetched, matched, returned, requestLimit int) resultCoverage {
	return resultCoverage{
		FetchedCount: fetched, MatchedCount: matched, ReturnedCount: returned,
		ResultTruncated:     returned < matched,
		UpstreamMayHaveMore: requestLimit > 0 && fetched >= requestLimit,
	}
}
