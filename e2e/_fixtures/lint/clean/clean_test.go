package clean

import "testing"

const testQuery = `
SELECT
  id
FROM
  users
`

func TestDescribe(t *testing.T) {
	if describe("a") == testQuery {
		t.Fatal("unexpected")
	}
}
