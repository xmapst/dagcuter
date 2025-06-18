package dagcuter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockTask struct {
	name         string
	dependencies []string
	PreExecFunc  func(ctx context.Context, input map[string]any) error
	ExecFunc     func(ctx context.Context, input map[string]any) (map[string]any, error)
	PostExecFunc func(ctx context.Context, output map[string]any) error
}

func (m *MockTask) Name() string {
	return m.name
}

func (m *MockTask) Dependencies() []string {
	return m.dependencies
}

func (m *MockTask) RetryPolicy() *RetryPolicy {
	return nil // No retry policy for mock tasks
}

func (m *MockTask) PreExecution(ctx context.Context, input map[string]any) error {
	if m.PreExecFunc != nil {
		return m.PreExecFunc(ctx, input)
	}
	return nil
}

func (m *MockTask) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, input)
	}
	return nil, nil
}

func (m *MockTask) PostExecution(ctx context.Context, output map[string]any) error {
	if m.PostExecFunc != nil {
		return m.PostExecFunc(ctx, output)
	}
	return nil
}

func TestDagcuter(t *testing.T) {
	tests := []struct {
		name          string
		tasks         map[string]Task
		expectedError string
		expectedOrder []string
	}{
		{
			name: "Successful execution",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}},
				"task2": &MockTask{name: "task2", dependencies: []string{"task1"}},
				"task3": &MockTask{name: "task3", dependencies: []string{"task2"}},
			},
			expectedError: "",
			expectedOrder: []string{"task1", "task2", "task3"},
		},
		{
			name: "Circular dependency",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{"task3"}},
				"task2": &MockTask{name: "task2", dependencies: []string{"task1"}},
				"task3": &MockTask{name: "task3", dependencies: []string{"task2"}},
			},
			expectedError: "circular dependency detected",
		},
		{
			name: "Task execution failure",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}},
				"task2": &MockTask{name: "task2", dependencies: []string{"task1"}, ExecFunc: func(ctx context.Context, input map[string]any) (map[string]any, error) {
					return nil, fmt.Errorf("task2 execution failed")
				}},
				"task3": &MockTask{name: "task3", dependencies: []string{"task2"}},
			},
			expectedError: "task2 execution failed",
		},
		{
			name:          "Empty task list",
			tasks:         map[string]Task{},
			expectedError: "",
			expectedOrder: []string{},
		},
		{
			name: "Single task with no dependencies",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}},
			},
			expectedError: "",
			expectedOrder: []string{"task1"},
		},
		{
			name: "Multiple independent tasks",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}},
				"task2": &MockTask{name: "task2", dependencies: []string{}},
				"task3": &MockTask{name: "task3", dependencies: []string{}},
			},
			expectedError: "",
			expectedOrder: []string{"task1", "task2", "task3"},
		},
		{
			name: "PreExecution failure",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}, PreExecFunc: func(ctx context.Context, input map[string]any) error {
					return fmt.Errorf("task1 pre-execution failed")
				}},
				"task2": &MockTask{name: "task2", dependencies: []string{"task1"}},
			},
			expectedError: "pre execution task task1 failed",
		},
		{
			name: "PostExecution failure",
			tasks: map[string]Task{
				"task1": &MockTask{name: "task1", dependencies: []string{}, PostExecFunc: func(ctx context.Context, output map[string]any) error {
					return fmt.Errorf("task1 post-execution failed")
				}},
				"task2": &MockTask{name: "task2", dependencies: []string{"task1"}},
			},
			expectedError: "post execution task task1 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDagcuter(tt.tasks)
			if tt.expectedError == "circular dependency detected" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, dag)

			results, err := dag.Execute(context.Background())
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				assert.ElementsMatch(t, tt.expectedOrder, dag.executionOrder)
			}
		})
	}
}
