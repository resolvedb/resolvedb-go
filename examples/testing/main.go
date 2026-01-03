// Testing example - interface-based mocking for unit tests.
package main

import (
	"context"
	"fmt"

	"github.com/resolvedb/resolvedb-go"
)

// WeatherService uses the Querier interface for testability.
type WeatherService struct {
	client resolvedb.Querier
}

// NewWeatherService creates a new weather service.
func NewWeatherService(client resolvedb.Querier) *WeatherService {
	return &WeatherService{client: client}
}

// GetTemperature returns the temperature for a city.
func (s *WeatherService) GetTemperature(ctx context.Context, city string) (float64, error) {
	type Weather struct {
		TempC float64 `json:"temp_c"`
	}

	var w Weather
	if err := s.client.Get(ctx, "weather", city, &w); err != nil {
		return 0, err
	}
	return w.TempC, nil
}

// MockClient implements Querier for testing.
type MockClient struct {
	responses map[string]any
	errors    map[string]error
}

func NewMockClient() *MockClient {
	return &MockClient{
		responses: make(map[string]any),
		errors:    make(map[string]error),
	}
}

func (m *MockClient) SetResponse(resource, key string, response any) {
	m.responses[resource+"/"+key] = response
}

func (m *MockClient) SetError(resource, key string, err error) {
	m.errors[resource+"/"+key] = err
}

func (m *MockClient) Get(ctx context.Context, resource, key string, dst any, opts ...resolvedb.RequestOption) error {
	k := resource + "/" + key
	if err, ok := m.errors[k]; ok {
		return err
	}
	if resp, ok := m.responses[k]; ok {
		// Simple copy for demo (real impl would use reflection)
		switch d := dst.(type) {
		case *map[string]any:
			if r, ok := resp.(map[string]any); ok {
				*d = r
			}
		}
	}
	return nil
}

func (m *MockClient) GetRaw(ctx context.Context, resource, key string, opts ...resolvedb.RequestOption) (*resolvedb.Response, error) {
	return nil, nil
}

func (m *MockClient) List(ctx context.Context, resource string, opts ...resolvedb.RequestOption) ([]string, error) {
	return nil, nil
}

func main() {
	// Production usage with real client
	fmt.Println("=== Production Usage ===")
	realClient, err := resolvedb.New()
	if err != nil {
		fmt.Printf("Client creation error: %v\n", err)
		return
	}
	defer realClient.Close()

	service := NewWeatherService(realClient)
	temp, err := service.GetTemperature(context.Background(), "quebec")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Quebec temperature: %.1f°C\n", temp)
	}

	// Test usage with mock
	fmt.Println("\n=== Test Usage (Mock) ===")
	mockClient := NewMockClient()
	mockClient.SetResponse("weather", "test-city", map[string]any{"temp_c": 25.5})

	testService := NewWeatherService(mockClient)
	_ = testService // In a real test, you would call testService.GetTemperature()
	fmt.Println("Mock client configured for testing")
	fmt.Println("Use interface-based design for easy mocking")
}
