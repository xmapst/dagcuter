package main

import (
	"context"
	"fmt"

	"github.com/alirezazeynali75/dagcuter"
)

type authTask struct{}

func (a *authTask) Name() string {
	return "auth"
}
func (a *authTask) Dependencies() []string {
	return []string{}
}
func (a *authTask) RetryPolicy() *dagcuter.RetryPolicy {
	return nil // No retry policy for mock tasks
}
func (a *authTask) PreExecution(ctx context.Context, input map[string]any) error {
	fmt.Println("PreExecution auth")
	return nil
}
func (a *authTask) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	fmt.Println("Executing auth")
	return map[string]any{"token": "token"}, nil
}
func (a *authTask) PostExecution(ctx context.Context, output map[string]any) error {
	return nil
}

type profileTask struct{}

func (a *profileTask) Name() string {
	return "profile"
}
func (a *profileTask) Dependencies() []string {
	return []string{
		"auth",
	}
}
func (a *profileTask) RetryPolicy() *dagcuter.RetryPolicy {
	return nil
}
func (a *profileTask) PreExecution(ctx context.Context, input map[string]any) error {
	fmt.Println("PreExecution profile")
	return nil
}
func (a *profileTask) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	fmt.Println("Executing profile")
	for k, v := range input {
		fmt.Println("Input: ", k, v)
	}
	return map[string]any{"profile": "hello"}, nil
}
func (a *profileTask) PostExecution(ctx context.Context, output map[string]any) error {
	fmt.Println("PostExecution profile")
	return nil
}

type offerTask struct{}

func (a *offerTask) Name() string {
	return "offerTask"
}
func (a *offerTask) Dependencies() []string {
	return []string{}
}
func (a *offerTask) RetryPolicy() *dagcuter.RetryPolicy {
	return nil // No retry policy for mock tasks
}
func (a *offerTask) PreExecution(ctx context.Context, input map[string]any) error {
	fmt.Println("PreExecution offerTask")
	return nil
}
func (a *offerTask) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	fmt.Println("Executing offerTask")
	return map[string]any{"offerTask": "hello"}, nil
}
func (a *offerTask) PostExecution(ctx context.Context, output map[string]any) error {
	fmt.Println("PostExecution offerTask")
	return nil
}

func main() {
	tasks := make(map[string]dagcuter.Task)
	tasks["auth"] = &authTask{}
	tasks["profile"] = &profileTask{}
	tasks["offerTask"] = &offerTask{}
	dag, err := dagcuter.NewDagcuter(
		tasks,
	)

	if err != nil {
		panic(err)
	}
	results, err := dag.Execute(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println("Results: ", results)
	fmt.Println("Execution order: ", dag.ExecutionOrder())
}
