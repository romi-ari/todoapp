package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
)

type Todo struct {
	ID     int    `json:"id"`
	Task   string `json:"task"`
	Status bool   `json:"status"`
}

func main() {
	godotenv.Load()
	// Create a new AWS session
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			os.Getenv("AWS_SESSION_TOKEN"),
		)),
	)

	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create a new AWS RDS client
	svc := rds.NewFromConfig(cfg)

	// Define the name of your RDS instance
	instanceName := os.Getenv("DB_IDENTIFIER")

	// Create a DescribeDBInstances input
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceName),
	}

	// Retrieve information about the RDS instance
	result, err := svc.DescribeDBInstances(context.TODO(), input)
	if err != nil {
		log.Fatalf("Failed to describe RDS instance: %v", err)
	}

	// Ensure that the result contains at least one DB instance
	if len(result.DBInstances) == 0 {
		log.Fatalf("No DB instance found with the name: %s", instanceName)
	}

	// Extract the RDS endpoint from the first DB instance in the result
	endpoint := *result.DBInstances[0].Endpoint.Address

	connString := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		endpoint,
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))
	// Connect to PostgreSQL
	conn, err := pgx.Connect(context.Background(), connString)

	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("Connected to database")

	router := gin.Default()

	router.Use(cors.Default())

	// Routes

	// GET route to list all todos
	router.GET("/api/todos", func(c *gin.Context) {
		rows, err := conn.Query(context.Background(), "SELECT id, task, status FROM todos")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var todo Todo
			if err := rows.Scan(&todo.ID, &todo.Task, &todo.Status); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			todos = append(todos, todo)
		}

		c.JSON(http.StatusOK, todos)
	})

	// POST route to add a new todo
	router.POST("/api/todos", func(c *gin.Context) {
		var todo Todo
		if err := c.BindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Insert into PostgreSQL
		_, err := conn.Exec(context.Background(), "INSERT INTO todos (task, status) VALUES ($1, $2)", todo.Task, todo.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, todo)
	})

	// PUT route to mark a todo as done
	router.PUT("/api/todos/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Check if todo exists
		var count int
		err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM todos WHERE id = $1", id).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}

		var updatedTodo Todo
		if err := c.BindJSON(&updatedTodo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update the todo
		_, err = conn.Exec(context.Background(), "UPDATE todos SET task = $1, status = $2 WHERE id = $3", updatedTodo.Task, updatedTodo.Status, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, updatedTodo)
	})

	// DELETE route to delete a todo
	router.DELETE("/api/todos/:id", func(c *gin.Context) {
		id := c.Param("id")

		// Check if todo exists
		var count int
		err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM todos WHERE id = $1", id).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}

		// Delete the todo
		_, err = conn.Exec(context.Background(), "DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
	})

	log.Fatal(router.Run(":8090"))
}
