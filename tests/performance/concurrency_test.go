package performance_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestConcurrency_GeneratorThreadSafety tests that multiple generators can be used concurrently
func TestConcurrency_GeneratorThreadSafety(t *testing.T) {
	t.Parallel()
	const numGoroutines = 50
	const numIterations = 20

	serviceData := &httpinterface.ServiceData{
		PackageName: "concurrent",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "ConcurrentService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetResource",
						InputType:  "GetResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/resources/{id}", Body: "", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "CreateResource",
						InputType:  "CreateResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/resources", Body: "*"},
						},
					},
				},
			},
		},
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Launch multiple goroutines that create and use generators
	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Create a new generator for each iteration to test factory safety
				generator := httpinterface.New()

				generated, err := generator.GenerateCode(serviceData)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, iteration %d: %v", goroutineID, j, err)
					return
				}

				// Verify the generated code is correct
				if generated == "" {
					errors <- fmt.Errorf("goroutine %d, iteration %d: empty generated code", goroutineID, j)
					return
				}

				// Check for expected patterns
				expectedPatterns := []string{
					"package concurrent",
					"ConcurrentServiceHandler interface",
					"HandleGetResource",
					"HandleCreateResource",
				}

				for _, pattern := range expectedPatterns {
					if !strings.Contains(generated, pattern) {
						errors <- fmt.Errorf("goroutine %d, iteration %d: missing pattern %q", goroutineID, j, pattern)
						return
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 { // Limit error output
			t.Error("... (truncated, too many errors)")
			break
		}
	}

	if errorCount > 0 {
		t.Fatalf("Found %d concurrency errors", errorCount)
	}

	t.Logf("Successfully ran %d goroutines × %d iterations = %d concurrent generations",
		numGoroutines, numIterations, numGoroutines*numIterations)
}

// TestConcurrency_SharedGeneratorState tests sharing a single generator across goroutines
func TestConcurrency_SharedGeneratorState(t *testing.T) {
	t.Parallel()
	// Create a single generator to be shared across goroutines
	generator := httpinterface.New()

	const numGoroutines = 25

	// Different service data for each goroutine to test state isolation
	createServiceData := func(id int) *httpinterface.ServiceData {
		return &httpinterface.ServiceData{
			PackageName: fmt.Sprintf("service%d", id),
			Services: []httpinterface.ServiceInfo{
				{
					Name: fmt.Sprintf("Service%d", id),
					Methods: []httpinterface.MethodInfo{
						{
							Name:       fmt.Sprintf("GetResource%d", id),
							InputType:  fmt.Sprintf("GetRequest%d", id),
							OutputType: fmt.Sprintf("Response%d", id),
							HTTPRules: []httpinterface.HTTPRule{
								{Method: "GET", Pattern: fmt.Sprintf("/service%d/resources/{id}", id),
									Body: "", PathParams: []string{"id"}},
							},
						},
					},
				},
			},
		}
	}

	var wg sync.WaitGroup
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Launch goroutines that use the shared generator
	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			serviceData := createServiceData(goroutineID)
			generated, err := generator.GenerateCode(serviceData)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %v", goroutineID, err)
				return
			}

			// Verify the generated code contains goroutine-specific patterns
			expectedPackage := fmt.Sprintf("package service%d", goroutineID)
			expectedService := fmt.Sprintf("Service%dHandler interface", goroutineID)
			expectedMethod := fmt.Sprintf("HandleGetResource%d", goroutineID)

			if !strings.Contains(generated, expectedPackage) ||
				!strings.Contains(generated, expectedService) ||
				!strings.Contains(generated, expectedMethod) {
				errors <- fmt.Errorf("goroutine %d: generated code missing expected patterns", goroutineID)
				return
			}

			results <- fmt.Sprintf("service%d", goroutineID)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all goroutines completed successfully
	resultCount := 0
	for range results {
		resultCount++
	}

	if resultCount != numGoroutines {
		t.Errorf("Expected %d successful results, got %d", numGoroutines, resultCount)
	}

	t.Logf("Successfully shared generator across %d goroutines", numGoroutines)
}

// TestConcurrency_ProtocPluginInterface tests the protoc plugin interface under concurrent load
func TestConcurrency_ProtocPluginInterface(t *testing.T) {
	t.Parallel()
	const numGoroutines = 20

	// Create a realistic file descriptor for testing
	createRequest := func(serviceID int) *pluginpb.CodeGeneratorRequest {
		return &pluginpb.CodeGeneratorRequest{
			Parameter:      proto.String(""),
			FileToGenerate: []string{fmt.Sprintf("service%d.proto", serviceID)},
			ProtoFile: []*descriptorpb.FileDescriptorProto{
				{
					Name:    proto.String(fmt.Sprintf("service%d.proto", serviceID)),
					Package: proto.String(fmt.Sprintf("pkg%d", serviceID)),
					Options: &descriptorpb.FileOptions{
						GoPackage: proto.String(fmt.Sprintf("github.com/test/pkg%d;pkg%d", serviceID, serviceID)),
					},
					Service: []*descriptorpb.ServiceDescriptorProto{
						{
							Name: proto.String(fmt.Sprintf("Service%d", serviceID)),
							Method: []*descriptorpb.MethodDescriptorProto{
								{
									Name:       proto.String("GetTest"),
									InputType:  proto.String("GetTestRequest"),
									OutputType: proto.String("TestResponse"),
									Options:    &descriptorpb.MethodOptions{},
								},
							},
						},
					},
				},
			},
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Launch goroutines that use the protoc plugin interface
	for i := range numGoroutines {
		wg.Add(1)
		go func(serviceID int) {
			defer wg.Done()

			generator := httpinterface.New()
			request := createRequest(serviceID)

			response := generator.Generate(request)
			if response.GetError() != "" {
				errors <- fmt.Errorf("goroutine %d: generation error: %s", serviceID, response.GetError())
				return
			}

			// For services without HTTP annotations, no files should be generated
			// This tests that the plugin correctly handles empty generation scenarios
			if len(response.GetFile()) > 0 {
				errors <- fmt.Errorf("goroutine %d: unexpected files generated for service without HTTP annotations", serviceID)
				return
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Found %d errors in protoc plugin interface concurrency test", errorCount)
	}

	t.Logf("Successfully tested protoc plugin interface with %d concurrent requests", numGoroutines)
}

// TestConcurrency_RaceConditionDetection tests for race conditions using data races
// This test is designed to be run with: go test -race
func TestConcurrency_RaceConditionDetection(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping race condition detection test in short mode")
	}

	const numGoroutines = 30
	const numIterations = 10

	// Shared data structure that might cause race conditions if not handled properly
	sharedGenerator := httpinterface.New()

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Create diverse service data to test different code paths
	serviceTypes := []*httpinterface.ServiceData{
		{
			PackageName: "race1",
			Services: []httpinterface.ServiceInfo{
				{
					Name: "RaceService1",
					Methods: []httpinterface.MethodInfo{
						{Name: "Method1", HTTPRules: []httpinterface.HTTPRule{{Method: "GET", Pattern: "/path1"}}},
					},
				},
			},
		},
		{
			PackageName: "race2",
			Services: []httpinterface.ServiceInfo{
				{
					Name: "RaceService2",
					Methods: []httpinterface.MethodInfo{
						{Name: "Method2", HTTPRules: []httpinterface.HTTPRule{{Method: "POST", Pattern: "/path2", Body: "*"}}},
					},
				},
			},
		},
		{
			PackageName: "race3",
			Services: []httpinterface.ServiceInfo{
				{
					Name: "RaceService3",
					Methods: []httpinterface.MethodInfo{
						{
							Name: "Method3",
							HTTPRules: []httpinterface.HTTPRule{
								{Method: "PUT", Pattern: "/path3/{id}", PathParams: []string{"id"}},
							},
						},
					},
				},
			},
		},
	}

	// Launch goroutines that stress test the generator
	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Use different service data to test various code paths
				serviceData := serviceTypes[j%len(serviceTypes)]

				_, err := sharedGenerator.GenerateCode(serviceData)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d, iteration %d: %v", goroutineID, j, err)
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Report any errors
	for err := range errors {
		t.Error(err)
	}

	t.Logf("Race condition test completed with %d goroutines × %d iterations", numGoroutines, numIterations)
}

// TestConcurrency_MemoryConsistency tests memory consistency under concurrent load
func TestConcurrency_MemoryConsistency(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping memory consistency test in short mode")
	}

	const numGoroutines = 40
	const numGenerations = 5

	// Track memory allocation patterns (simplified test)
	results := make(map[string]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	serviceData := &httpinterface.ServiceData{
		PackageName: "memory",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "MemoryTestService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "TestMethod",
						HTTPRules: []httpinterface.HTTPRule{{Method: "GET", Pattern: "/test"}},
					},
				},
			},
		},
	}

	// Launch goroutines that generate code and track results
	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numGenerations; j++ {
				generator := httpinterface.New()
				generated, err := generator.GenerateCode(serviceData)
				if err != nil {
					t.Errorf("Generation error in goroutine %d: %v", goroutineID, err)
					return
				}

				// Track consistent patterns in generated code
				key := fmt.Sprintf("len_%d", len(generated))
				mu.Lock()
				results[key]++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify memory consistency - all generations should produce identical output
	if len(results) != 1 {
		t.Errorf("Memory consistency issue: expected 1 unique result length, got %d: %v", len(results), results)
	}

	t.Logf("Memory consistency test completed: %v", results)
}

// TestConcurrency_DeadlockDetection tests for potential deadlocks
func TestConcurrency_DeadlockDetection(t *testing.T) {
	t.Parallel()
	const timeout = 5 * time.Second
	const numGoroutines = 10

	done := make(chan bool, numGoroutines)
	serviceData := &httpinterface.ServiceData{
		PackageName: "deadlock",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "DeadlockTestService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "TestMethod",
						HTTPRules: []httpinterface.HTTPRule{{Method: "GET", Pattern: "/test"}},
					},
				},
			},
		},
	}

	// Launch goroutines that could potentially deadlock
	for i := range numGoroutines {
		go func(id int) {
			generator := httpinterface.New()
			_, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Use timeout to detect deadlocks
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	completed := 0
	for completed < numGoroutines {
		select {
		case <-done:
			completed++
		case <-timer.C:
			t.Fatalf("Deadlock detected: only %d/%d goroutines completed within %v", completed, numGoroutines, timeout)
		}
	}

	t.Logf("Deadlock detection test passed: all %d goroutines completed within timeout", numGoroutines)
}
