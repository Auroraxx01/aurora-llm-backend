package persistent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitDB(t *testing.T) {
	defer destroyDB()
	assert.NotPanics(t, func() {
		InitDB()
	})
}
