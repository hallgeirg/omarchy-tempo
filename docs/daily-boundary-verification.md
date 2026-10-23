# Daily activity cutoff

The activity chart previously ended today's bucket at the next local midnight, while the calendar-week total and project summary already ended at the current time. A cached completed entry whose duration extended beyond the current time could therefore show more hours in today's chart than in the other summaries.

With a fixed Monday at 10:00 and an entry starting at 09:00 with a 7,200-second duration, the original code reported 7,200 seconds today but 3,600 seconds this week. An entirely future entry and a midnight-spanning future-ending entry reproduced the same inconsistency. These are synthetic cached-data fixtures; no assertion about live Toggl input validation is required.

Each daily bucket now ends at the earlier of its next local midnight and `now`. Earlier days keep their original calendar boundaries. Running entries, completed past entries, Monday week boundaries, and Oslo's 23-hour/25-hour DST days are preserved.

Focused offline regression:

```bash
cd src
GOPROXY=off GOTOOLCHAIN=local go test core.go daily_boundary_test.go -count=1
```

The tests compile the actual `core.go` with a standard-library-only test file. The original implementation fails the three future-ending cases; the corrected implementation passes. No live requests, tokens, installed desktop configuration, saved-timer files, or bundled binaries are changed. A full build or suite is separate and must use the build queue.
