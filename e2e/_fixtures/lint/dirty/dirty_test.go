package dirty

import "testing"

const testQuery = `select id from users`

func TestQuery(t *testing.T) {
	if testQuery == activeUsers {
		t.Fatal("unexpected")
	}
}
