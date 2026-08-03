package discovery

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestErrWaitTimeout_errorsIs 超时错误可识别
//
// 【功能点】ErrWaitTimeout 与 DeadlineExceeded 包装
// 【测试流程】
// 1. wrap ErrWaitTimeout + DeadlineExceeded
// 2. errors.Is 分别命中
func TestErrWaitTimeout_errorsIs(t *testing.T) {
	err := fmt.Errorf("%w: %w", ErrWaitTimeout, context.DeadlineExceeded)
	require.True(t, errors.Is(err, ErrWaitTimeout))
	require.True(t, errors.Is(err, context.DeadlineExceeded))
}
