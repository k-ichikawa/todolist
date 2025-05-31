package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" // MySQLドライバーをインポート
)

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title" binding:"required"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(db:3306)/todo_app?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci&interpolateParams=true")
	if err != nil {
		log.Fatalf("データベースの接続に失敗しました: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("データベースへのPingに失敗しました: %v", err)
	}
	log.Println("MySQLに正常に接続しました!")

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/health", healthCheck)
		apiGroup.GET("/todos", getTodos)
		apiGroup.POST("/todos", createTodo)
	}

	log.Println("Go APIサーバーがポート8080で起動しました")
	log.Fatal(router.Run(":8080"))
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getTodos(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, completed, created_at FROM todos ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Todoのクエリに失敗しました: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Sever Error"})
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt); err != nil {
			log.Printf("Todoのスキャンに失敗しました: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		log.Printf("Debug: Fetched todo title from DB: %s", todo.Title) // ★★★ この行を追加 ★★★
		todos = append(todos, todo)
	}
	c.JSON(http.StatusOK, todos)
}

func createTodo(c *gin.Context) {
	var newTodo Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stmt, err := db.Prepare("INSERT INTO todos (title, completed, created_at) Values (?, ?, ?)")
	if err != nil {
		log.Printf("ステートメントの準備に失敗しました: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Sever Error"})
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(newTodo.Title, false, time.Now())
	if err != nil {
		log.Printf("挿入の実行に失敗しました: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Sever Error"})
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("挿入されたIDの取得に失敗しました: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Sever Error"})
		return
	}

	createdTodo := Todo{
		ID:        int(id),
		Title:     newTodo.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	c.JSON(http.StatusCreated, createdTodo)
}
