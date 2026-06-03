package engram

import "testing"

// TestListProjects_GroupsAndCounts verifies distinct project keys with non-deleted
// counts, most populous first. Seed: atelier=6 non-deleted (the soft-deleted row
// is excluded), other=2.
func TestListProjects_GroupsAndCounts(t *testing.T) {
	c, err := NewClient(buildTestDB(t))
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer func() { _ = c.Close() }()

	stats, err := c.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects error = %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("ListProjects returned %d projects, want 2 (%+v)", len(stats), stats)
	}
	// Ordered by count desc → atelier first.
	if stats[0].Key != "atelier" || stats[0].Count != 6 {
		t.Errorf("stats[0] = %+v, want {atelier 6}", stats[0])
	}
	if stats[1].Key != "other" || stats[1].Count != 2 {
		t.Errorf("stats[1] = %+v, want {other 2}", stats[1])
	}
}

// TestListByProject_CaseInsensitive verifies COLLATE NOCASE: an upper-cased name
// matches the lower-cased stored key.
func TestListByProject_CaseInsensitive(t *testing.T) {
	c, err := NewClient(buildTestDB(t))
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer func() { _ = c.Close() }()

	lower, err := c.ListByProject("atelier")
	if err != nil {
		t.Fatalf("ListByProject(atelier) error = %v", err)
	}
	upper, err := c.ListByProject("ATELIER")
	if err != nil {
		t.Fatalf("ListByProject(ATELIER) error = %v", err)
	}
	if len(lower) == 0 {
		t.Fatal("ListByProject(atelier) returned 0 — seed broken?")
	}
	if len(upper) != len(lower) {
		t.Errorf("case-insensitive mismatch: ATELIER=%d, atelier=%d", len(upper), len(lower))
	}
}
