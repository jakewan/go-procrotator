package logger_test

import (
	"testing"

	"github.com/jakewan/go-procrotator/logger"
	"github.com/stretchr/testify/assert"
)

func TestEnum(t *testing.T) {
	assert.Equal(t, "NOTSET", logger.NOTSET.String())
	assert.Equal(t, 0, logger.NOTSET.EnumIndex())
	assert.Equal(t, "DEBUG", logger.DEBUG.String())
	assert.Equal(t, 1, logger.DEBUG.EnumIndex())
	assert.Equal(t, "INFO", logger.INFO.String())
	assert.Equal(t, 2, logger.INFO.EnumIndex())
	assert.Equal(t, "NOTICE", logger.NOTICE.String())
	assert.Equal(t, 3, logger.NOTICE.EnumIndex())
	assert.Equal(t, "WARNING", logger.WARNING.String())
	assert.Equal(t, 4, logger.WARNING.EnumIndex())
	assert.Equal(t, "ERROR", logger.ERROR.String())
	assert.Equal(t, 5, logger.ERROR.EnumIndex())
}
