package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
)

type Todo struct {
	ID     int    `json:"id"`
	Task   string `json:"task"`
	Status bool   `json:"status"`
}

var db *pgx.Conn

func main() {

	// Connect to PostgreSQL
	connString := "postgres://postgres:postgres@localhost:5432/todoapp"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())
	db = conn
	fmt.Println("Connected to database")

	router := gin.Default()

	router.Use(cors.Default())

	// Routes

	// GET route to list all todos
	router.GET("/api/todos", func(c *gin.Context) {
		rows, err := db.Query(context.Background(), "SELECT id, task, status FROM todos")
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
		_, err := db.Exec(context.Background(), "INSERT INTO todos (task, status) VALUES ($1, $2)", todo.Task, todo.Status)
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
		err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM todos WHERE id = $1", id).Scan(&count)
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
		_, err = db.Exec(context.Background(), "UPDATE todos SET task = $1, status = $2 WHERE id = $3", updatedTodo.Task, updatedTodo.Status, id)
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
		err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM todos WHERE id = $1", id).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
			return
		}

		// Delete the todo
		_, err = db.Exec(context.Background(), "DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Todo deleted successfully"})
	})

	log.Fatal(router.Run(":8090"))
}
