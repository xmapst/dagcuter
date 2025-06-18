package dagcuter

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Dagcuter struct {
	Tasks          map[string]Task
	results        *sync.Map
	inDegrees      map[string]int
	dependents     map[string][]string
	executionOrder []string
	mu             *sync.Mutex
	wg             *sync.WaitGroup
}

func NewDagcuter(tasks map[string]Task) (*Dagcuter, error) {
	if HasCycle(tasks) {
		return nil, fmt.Errorf("circular dependency detected")
	}
	dag := &Dagcuter{
		mu:         new(sync.Mutex),
		wg:         new(sync.WaitGroup),
		results:    new(sync.Map),
		inDegrees:  make(map[string]int),
		dependents: make(map[string][]string),
		Tasks:      tasks,
	}

	for name, task := range dag.Tasks {
		dag.inDegrees[name] = len(task.Dependencies())
		for _, dep := range task.Dependencies() {
			dag.dependents[dep] = append(dag.dependents[dep], name)
		}
	}

	return dag, nil
}

func (d *Dagcuter) Execute(ctx context.Context) (map[string]map[string]any, error) {
	defer d.results.Clear()
	errCh := make(chan error, 1)

	for name, deg := range d.inDegrees {
		if deg == 0 {
			d.wg.Add(1)
			go d.runTask(ctx, name, errCh)
		}
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		results := make(map[string]map[string]any)
		d.results.Range(func(key, value any) bool {
			results[key.(string)] = value.(map[string]any)
			return true
		})
		return results, nil
	case err := <-errCh:
		return nil, err
	}
}

func (d *Dagcuter) runTask(ctx context.Context, name string, errCh chan error) {
	defer d.wg.Done()
	task := d.Tasks[name]

	d.mu.Lock()
	inputs := d.prepareInputs(task)
	d.mu.Unlock()

	output, err := d.executeTask(ctx, name, task, inputs)
	if err != nil {
		select {
		case errCh <- err:
		default:
		}
		return
	}

	d.mu.Lock()
	d.executionOrder = append(d.executionOrder, name)
	d.mu.Unlock()

	d.mu.Lock()
	d.results.Store(name, output)
	for _, child := range d.dependents[name] {
		d.inDegrees[child]--
		if d.inDegrees[child] == 0 {
			d.wg.Add(1)
			go d.runTask(ctx, child, errCh)
		}
	}
	d.mu.Unlock()
}

func (d *Dagcuter) executeTask(ctx context.Context, name string, task Task, inputs map[string]any) (map[string]any, error) {
	// get the retry executor for the task
	retryExecutor := d.newRetryExecutor(task.RetryPolicy())

	var result map[string]any

	// use the retry executor to execute the task with retries
	err := retryExecutor.ExecuteWithRetry(ctx, name, func() error {
		// PreExecution
		if err := task.PreExecution(ctx, inputs); err != nil {
			return fmt.Errorf("pre execution failed: %w", err)
		}

		// Execute
		output, err := task.Execute(ctx, inputs)
		if err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}

		// PostExecution
		if err = task.PostExecution(ctx, output); err != nil {
			return fmt.Errorf("post execution failed: %w", err)
		}

		result = output
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("task %s failed after all retry attempts: %w", name, err)
	}

	return result, nil
}

func (d *Dagcuter) prepareInputs(task Task) map[string]any {
	inputs := make(map[string]any)
	for _, dep := range task.Dependencies() {
		if value, ok := d.results.Load(dep); ok {
			inputs[dep] = value
		}
	}
	return inputs
}

func (d *Dagcuter) ExecutionOrder() string {
	var sb = strings.Builder{}
	sb.WriteString("\n")
	for i, step := range d.executionOrder {
		_, _ = fmt.Fprintf(&sb, "%d. %s\n", i+1, step)
	}
	return sb.String()
}

// PrintGraph Output chain dependencies
func (d *Dagcuter) PrintGraph() {
	// 1. Find all root nodes (in-degree 0)
	var roots []string
	for name, deg := range d.inDegrees {
		if deg == 0 {
			roots = append(roots, name)
		}
	}
	// 2. Print from each root node separately
	for _, root := range roots {
		fmt.Println(root)        // Print root first
		d.printChain(root, "  ") // Start indenting two spaces from the next level
		fmt.Println()            // Add a blank line between different roots
	}
}

// printChain Recursively print sub dependencies
// name: Current node name
// prefix: Current indentation prefix (already includes spaces for the arrow)
func (d *Dagcuter) printChain(name, prefix string) {
	children := d.dependents[name]
	for _, child := range children {
		// Print arrow and child node
		fmt.Printf("%s└─> %s\n", prefix, child)
		// Recursively print the chain for the child node
		d.printChain(child, prefix+"    ")
	}
}
