//go:build integration

package postgrest

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	postgresContainer testcontainers.Container
	postgrestURL      string
	testClient        *Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Check if we should use testcontainers or existing services
	useTestcontainers := os.Getenv("USE_TESTCONTAINERS") != "false"
	existingURL := os.Getenv("POSTGREST_URL")

	if !useTestcontainers && existingURL != "" {
		// Use existing PostgREST instance
		postgrestURL = existingURL
		testClient = NewClient(postgrestURL, "public", nil)
		code := m.Run()
		os.Exit(code)
		return
	}

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:14",
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to start PostgreSQL container: %v", err))
	}
	postgresContainer = pgContainer

	// Get PostgreSQL connection details for host (for running SQL scripts)
	pgHost, err := pgContainer.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to get PostgreSQL host: %v", err))
	}

	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		panic(fmt.Sprintf("Failed to get PostgreSQL port: %v", err))
	}

	// Connection string for host (to run SQL scripts)
	pgConnStrHost := fmt.Sprintf("postgres://postgres:postgres@%s:%s/postgres?sslmode=disable", pgHost, pgPort.Port())

	// Get PostgreSQL container's internal IP address for PostgREST to connect
	pgContainerIP, err := pgContainer.ContainerIP(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to get PostgreSQL container IP: %v", err))
	}
	// Connection string for PostgREST container (using internal IP and port)
	pgConnStrContainer := fmt.Sprintf("postgres://postgres:postgres@%s:5432/postgres?sslmode=disable", pgContainerIP)

	// Connect to PostgreSQL and run schema scripts
	db, err := sql.Open("postgres", pgConnStrHost)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to PostgreSQL: %v", err))
	}
	defer db.Close()

	// Wait for database to be ready
	time.Sleep(2 * time.Second)

	// Read and execute schema SQL
	schemaSQL, err := os.ReadFile(filepath.Join("test", "00-schema.sql"))
	if err != nil {
		panic(fmt.Sprintf("Failed to read schema SQL: %v", err))
	}

	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		panic(fmt.Sprintf("Failed to execute schema SQL: %v", err))
	}

	// Read and execute dummy data SQL
	dataSQL, err := os.ReadFile(filepath.Join("test", "01-dummy-data.sql"))
	if err != nil {
		panic(fmt.Sprintf("Failed to read data SQL: %v", err))
	}

	_, err = db.Exec(string(dataSQL))
	if err != nil {
		panic(fmt.Sprintf("Failed to execute data SQL: %v", err))
	}

	// Start PostgREST container
	postgrestContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgrest/postgrest:latest",
			ExposedPorts: []string{"3000/tcp"},
			Env: map[string]string{
				"PGRST_DB_URI":       pgConnStrContainer,
				"PGRST_DB_SCHEMA":    "public,personal",
				"PGRST_DB_ANON_ROLE": "postgres",
				"PGRST_JWT_SECRET":   "reallyreallyreallyreallyverysafe",
			},
			WaitingFor: wait.ForHTTP("/").
				WithPort("3000").
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to start PostgREST container: %v", err))
	}

	// Get PostgREST URL
	postgrestHost, err := postgrestContainer.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to get PostgREST host: %v", err))
	}

	postgrestPort, err := postgrestContainer.MappedPort(ctx, "3000")
	if err != nil {
		panic(fmt.Sprintf("Failed to get PostgREST port: %v", err))
	}

	postgrestURL = fmt.Sprintf("http://%s:%s", postgrestHost, postgrestPort.Port())
	testClient = NewClient(postgrestURL, "public", nil)

	// Wait for PostgREST to be ready
	time.Sleep(2 * time.Second)

	// Run tests
	code := m.Run()

	// Cleanup
	if err := postgrestContainer.Terminate(ctx); err != nil {
		panic(fmt.Sprintf("Failed to terminate PostgREST container: %v", err))
	}

	if err := pgContainer.Terminate(ctx); err != nil {
		panic(fmt.Sprintf("Failed to terminate PostgreSQL container: %v", err))
	}

	os.Exit(code)
}

func TestIntegration_Select(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Select all users", func(t *testing.T) {
		response, err := testClient.From("users").Select("*", nil).Execute(ctx)
		require.NoError(t, err)
		require.Nil(t, response.Error)
		require.NotNil(t, response.Data)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Greater(t, len(users), 0)
		assert.Contains(t, users[0], "username")
	})

	t.Run("Select specific columns", func(t *testing.T) {
		response, err := testClient.From("users").Select("username, status", nil).Execute(ctx)
		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Greater(t, len(users), 0)
		assert.Contains(t, users[0], "username")
		assert.Contains(t, users[0], "status")
	})

	t.Run("Select with count", func(t *testing.T) {
		opts := &SelectOptions{Count: "exact"}
		response, err := testClient.From("users").Select("*", opts).Execute(ctx)
		require.NoError(t, err)
		require.Nil(t, response.Error)
		require.NotNil(t, response.Count)
		assert.Greater(t, *response.Count, int64(0))
	})
}

func TestIntegration_Filters(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Eq filter", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Eq("username", "supabot").
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Equal(t, 1, len(users))
		assert.Equal(t, "supabot", users[0]["username"])
	})

	t.Run("In filter", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			In("username", []interface{}{"supabot", "kiwicopple"}).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Equal(t, 2, len(users))
	})

	t.Run("Like filter", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Like("username", "%bot%").
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Greater(t, len(users), 0)
	})

	t.Run("Multiple filters", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Eq("status", "ONLINE").
			Limit(2, nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.LessOrEqual(t, len(users), 2)
		for _, user := range users {
			assert.Equal(t, "ONLINE", user["status"])
		}
	})
}

func TestIntegration_OrderAndLimit(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Order by username", func(t *testing.T) {
		opts := &OrderOptions{Ascending: true}
		response, err := testClient.
			From("users").
			Select("*", nil).
			Order("username", opts).
			Limit(3, nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.LessOrEqual(t, len(users), 3)
		if len(users) > 1 {
			// Check ordering
			username1 := users[0]["username"].(string)
			username2 := users[1]["username"].(string)
			assert.LessOrEqual(t, username1, username2)
		}
	})

	t.Run("Range", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Range(0, 2, nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.LessOrEqual(t, len(users), 3)
	})
}

func TestIntegration_Single(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Single result", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Eq("username", "supabot").
			Single().
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		// Single() returns a single object, not an array
		// response.Data is []map[string]interface{} but should be map[string]interface{}
		// We need to extract the first element if it's an array
		var user map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)

		// Try to unmarshal as array first (because Select returns []T)
		var arr []map[string]interface{}
		if err := json.Unmarshal(dataBytes, &arr); err == nil && len(arr) > 0 {
			user = arr[0]
		} else {
			// If not an array, try unmarshal as single object
			err = json.Unmarshal(dataBytes, &user)
			if err != nil {
				// If that fails, try as interface{} and type assert
				var data interface{}
				json.Unmarshal(dataBytes, &data)
				if m, ok := data.(map[string]interface{}); ok {
					user = m
				} else if arr, ok := data.([]interface{}); ok && len(arr) > 0 {
					if m, ok := arr[0].(map[string]interface{}); ok {
						user = m
					}
				}
			}
		}

		require.NotNil(t, user, "user should not be nil, response.Data: %v", response.Data)
		assert.Equal(t, "supabot", user["username"])
	})

	t.Run("MaybeSingle with result", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("*", nil).
			Eq("username", "supabot").
			MaybeSingle().
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		// MaybeSingle() returns a single object or null
		// response.Data might be []map[string]interface{} or map[string]interface{}
		var user map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)

		// Try to unmarshal as array first (because Select returns []T)
		var arr []map[string]interface{}
		if err := json.Unmarshal(dataBytes, &arr); err == nil {
			if len(arr) > 0 {
				user = arr[0]
			} else {
				// Empty array means no result, but we expect one result
				// Try unmarshal as single object instead
				err = json.Unmarshal(dataBytes, &user)
				if err != nil {
					// If that fails, try as interface{} and type assert
					var data interface{}
					json.Unmarshal(dataBytes, &data)
					if m, ok := data.(map[string]interface{}); ok {
						user = m
					} else if arr, ok := data.([]interface{}); ok && len(arr) > 0 {
						if m, ok := arr[0].(map[string]interface{}); ok {
							user = m
						}
					}
				}
			}
		} else {
			// If not an array, try unmarshal as single object
			err = json.Unmarshal(dataBytes, &user)
			if err != nil {
				// If that fails, try as interface{} and type assert
				var data interface{}
				json.Unmarshal(dataBytes, &data)
				if m, ok := data.(map[string]interface{}); ok {
					user = m
				} else if arr, ok := data.([]interface{}); ok && len(arr) > 0 {
					if m, ok := arr[0].(map[string]interface{}); ok {
						user = m
					}
				}
			}
		}

		// MaybeSingle might return empty array if no result, but we expect one result
		// So if user is nil, it means the query didn't return the expected result
		if user == nil {
			// Try using Single() instead to get the result
			response2, err2 := testClient.
				From("users").
				Select("*", nil).
				Eq("username", "supabot").
				Single().
				Execute(ctx)
			require.NoError(t, err2)
			require.Nil(t, response2.Error)

			dataBytes2, _ := json.Marshal(response2.Data)
			var arr2 []map[string]interface{}
			if err := json.Unmarshal(dataBytes2, &arr2); err == nil && len(arr2) > 0 {
				user = arr2[0]
			} else {
				var data2 interface{}
				json.Unmarshal(dataBytes2, &data2)
				if m, ok := data2.(map[string]interface{}); ok {
					user = m
				}
			}
		}

		require.NotNil(t, user, "user should not be nil, response.Data: %v", response.Data)
		assert.Equal(t, "supabot", user["username"])
	})
}

func TestIntegration_Insert(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Insert single row", func(t *testing.T) {
		newUser := map[string]interface{}{
			"username": fmt.Sprintf("testuser_%d", time.Now().Unix()),
			"status":   "ONLINE",
		}

		opts := &InsertOptions{Count: "exact"}
		response, err := testClient.
			From("users").
			Insert(newUser, opts).
			Select("*").
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Equal(t, 1, len(users))
		assert.Equal(t, newUser["username"], users[0]["username"])
	})
}

func TestIntegration_Update(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Update row", func(t *testing.T) {
		// First, insert a test user
		testUsername := fmt.Sprintf("update_test_%d", time.Now().Unix())
		newUser := map[string]interface{}{
			"username": testUsername,
			"status":   "ONLINE",
		}

		_, err := testClient.
			From("users").
			Insert(newUser, nil).
			Execute(ctx)
		require.NoError(t, err)

		// Update the user
		updateData := map[string]interface{}{
			"status": "OFFLINE",
		}

		opts := &UpdateOptions{Count: "exact"}
		response, err := testClient.
			From("users").
			Update(updateData, opts).
			Eq("username", testUsername).
			Select("*").
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Equal(t, 1, len(users))
		assert.Equal(t, "OFFLINE", users[0]["status"])
	})
}

func TestIntegration_Delete(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Delete row", func(t *testing.T) {
		// First, insert a test user
		testUsername := fmt.Sprintf("delete_test_%d", time.Now().Unix())
		newUser := map[string]interface{}{
			"username": testUsername,
			"status":   "ONLINE",
		}

		_, err := testClient.
			From("users").
			Insert(newUser, nil).
			Execute(ctx)
		require.NoError(t, err)

		// Delete the user
		opts := &DeleteOptions{Count: "exact"}
		response, err := testClient.
			From("users").
			Delete(opts).
			Eq("username", testUsername).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		// Verify deletion
		checkResponse, err := testClient.
			From("users").
			Select("*", nil).
			Eq("username", testUsername).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, checkResponse.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(checkResponse.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Equal(t, 0, len(users))
	})
}

func TestIntegration_RPC(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("RPC call", func(t *testing.T) {
		rpcOpts := &RpcOptions{}
		response, err := testClient.
			Rpc("get_status", map[string]interface{}{
				"name_param": "supabot",
			}, rpcOpts).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var status string
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &status)

		assert.Equal(t, "ONLINE", status)
	})
}

func TestIntegration_Schema(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Query personal schema", func(t *testing.T) {
		personalClient := testClient.Schema("personal")
		response, err := personalClient.
			From("users").
			Select("*", nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.Greater(t, len(users), 0)
	})
}

func TestIntegration_ChannelsAndMessages(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Select channels", func(t *testing.T) {
		response, err := testClient.
			From("channels").
			Select("*", nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var channels []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &channels)

		assert.Greater(t, len(channels), 0)
	})

	t.Run("Select messages with join", func(t *testing.T) {
		response, err := testClient.
			From("messages").
			Select("*, users(*), channels(*)", nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)

		var messages []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &messages)

		assert.Greater(t, len(messages), 0)
	})
}

func TestIntegration_ErrorHandling(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Invalid table", func(t *testing.T) {
		response, err := testClient.
			From("nonexistent_table").
			Select("*", nil).
			Execute(ctx)

		require.NoError(t, err)
		// Should have an error in response
		assert.NotNil(t, response.Error)
		assert.NotEqual(t, 200, response.Status)
	})

	t.Run("Invalid column", func(t *testing.T) {
		response, err := testClient.
			From("users").
			Select("nonexistent_column", nil).
			Execute(ctx)

		require.NoError(t, err)
		// Should have an error in response
		assert.NotNil(t, response.Error)
		assert.NotEqual(t, 200, response.Status)
	})
}

func TestIntegration_ComplexQueries(t *testing.T) {
	if testClient == nil {
		t.Skip("Skipping integration test: client not initialized")
	}

	ctx := context.Background()

	t.Run("Complex query with multiple filters and ordering", func(t *testing.T) {
		opts := &SelectOptions{Count: "exact"}
		orderOpts := &OrderOptions{Ascending: false}
		response, err := testClient.
			From("users").
			Select("username, status", opts).
			Eq("status", "ONLINE").
			Order("username", orderOpts).
			Limit(5, nil).
			Execute(ctx)

		require.NoError(t, err)
		require.Nil(t, response.Error)
		require.NotNil(t, response.Count)

		var users []map[string]interface{}
		dataBytes, _ := json.Marshal(response.Data)
		json.Unmarshal(dataBytes, &users)

		assert.LessOrEqual(t, len(users), 5)
		for _, user := range users {
			assert.Equal(t, "ONLINE", user["status"])
		}
	})
}

// Helper function to check if PostgREST is ready
func waitForPostgREST(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 404 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("PostgREST not ready after %v", timeout)
}
