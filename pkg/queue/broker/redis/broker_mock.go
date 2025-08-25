package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"reflect"
	"sync"
	"testing"
)

type mockCmdable struct {
	redis.UniversalClient
	expectations []expectation
	calls        []call
	mu           sync.Mutex
	nextExpect   int
}

type expectation struct {
	method string
	args   []interface{}
	result interface{}
	err    error
}

type call struct {
	method string
	args   []interface{}
}

func (m *mockCmdable) expect(method string, args []interface{}, result interface{}, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = append(m.expectations, expectation{method, args, result, err})
}

func (m *mockCmdable) recordCall(method string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call{method, args})
}

func (m *mockCmdable) check(t *testing.T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.expectations) != len(m.calls) {
		t.Errorf("expected %d calls, got %d", len(m.expectations), len(m.calls))
		return
	}

	if m.nextExpect != len(m.expectations) {
		t.Errorf("some expectations were not used: %d/%d", m.nextExpect, len(m.expectations))
	}

	for i, exp := range m.expectations {
		got := m.calls[i]
		if exp.method != got.method {
			t.Errorf("call %d: expected method %s, got %s", i, exp.method, got.method)
		}
		if !reflect.DeepEqual(exp.args, got.args) {
			t.Errorf("call %d: args mismatch: expected %v, got %v", i, exp.args, got.args)
		}
	}
}

func (m *mockCmdable) XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd {
	m.recordCall("XAdd", a)
	cmd := redis.NewStringCmd(ctx)

	m.mu.Lock()
	if m.nextExpect < len(m.expectations) {
		exp := m.expectations[m.nextExpect]
		m.nextExpect++
		m.mu.Unlock()

		if exp.err != nil {
			cmd.SetErr(exp.err)
		} else {
			cmd.SetVal(exp.result.(string))
		}
	} else {
		m.mu.Unlock()
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

func (m *mockCmdable) XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	m.recordCall("XReadGroup", a)
	cmd := redis.NewXStreamSliceCmd(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.nextExpect < len(m.expectations) {
		exp := m.expectations[m.nextExpect]
		if exp.method == "XReadGroup" {
			m.nextExpect++
			if exp.err != nil {
				cmd.SetErr(exp.err)
			} else {
				cmd.SetVal(exp.result.([]redis.XStream))
			}
		} else {
			cmd.SetErr(redis.Nil)
		}
	} else {
		cmd.SetErr(redis.Nil)
	}

	return cmd
}

func (m *mockCmdable) XInfoGroups(ctx context.Context, stream string) *redis.XInfoGroupsCmd {
	m.recordCall("XInfoGroups", stream)
	cmd := redis.NewXInfoGroupsCmd(ctx, stream)
	if len(m.expectations) > 0 {
		exp := m.expectations[0]
		m.expectations = m.expectations[1:]
		if exp.err != nil {
			cmd.SetErr(exp.err)
		} else {
			cmd.SetVal(exp.result.([]redis.XInfoGroup))
		}
	}
	return cmd
}

func (m *mockCmdable) XGroupCreateMkStream(ctx context.Context, stream, group, id string) *redis.StatusCmd {
	m.recordCall("XGroupCreateMkStream", stream, group, id)
	cmd := redis.NewStatusCmd(ctx)
	if len(m.expectations) > 0 {
		exp := m.expectations[0]
		m.expectations = m.expectations[1:]
		if exp.err != nil && !isGroupExists(exp.err) {
			cmd.SetErr(exp.err)
		} else {
			cmd.SetVal("OK")
		}
	}
	return cmd
}

func (m *mockCmdable) XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd {
	m.recordCall("XAck", stream, group, ids)
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(1)
	return cmd
}

func (m *mockCmdable) XPendingExt(ctx context.Context, a *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	m.recordCall("XPendingExt", a)
	cmd := redis.NewXPendingExtCmd(ctx)
	if len(m.expectations) > 0 {
		exp := m.expectations[0]
		m.expectations = m.expectations[1:]
		if exp.err != nil {
			cmd.SetErr(exp.err)
		} else {
			cmd.SetVal(exp.result.([]redis.XPendingExt))
		}
	}
	return cmd
}

func (m *mockCmdable) XClaim(ctx context.Context, a *redis.XClaimArgs) *redis.XMessageSliceCmd {
	m.recordCall("XClaim", a)
	cmd := redis.NewXMessageSliceCmd(ctx)
	if len(m.expectations) > 0 {
		exp := m.expectations[0]
		m.expectations = m.expectations[1:]
		if exp.err != nil {
			cmd.SetErr(exp.err)
		} else {
			cmd.SetVal(exp.result.([]redis.XMessage))
		}
	}
	return cmd
}
