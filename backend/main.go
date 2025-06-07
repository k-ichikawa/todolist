package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	g "github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title" binding:"required"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type BatchCreateRequest struct {
	Titles []string `json:"titles"`
}

type CreateTodoResult struct {
	Title   string `json:"title"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	ID      int    `json:"id,omitempty"`
}

func createTodoInGoroutine(db *sql.DB, title string, resultChan chan<- CreateTodoResult, wg *sync.WaitGroup) {
	defer wg.Done()

	stmt, err := db.Prepare("INSERT INTO todos (title) VALUES (?)")
	if err != nil {
		resultChan <- CreateTodoResult{Title: title, Status: "error", Message: fmt.Sprintf("ステートメント準備エラー: %w", err)}
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(title)
	if err != nil {
		resultChan <- CreateTodoResult{Title: title, Status: "error", Message: fmt.Sprintf("Todo作成エラー: %v", err)}
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		resultChan <- CreateTodoResult{Title: title, Status: "error", Message: fmt.Sprintf("ID取得エラー: %v", err)}
		return
	}

	resultChan <- CreateTodoResult{Title: title, Status: "success", ID: int(id)}
}

func setupRouter(db *sql.DB) *gin.Engine {
	router := gin.Default()

	// CORSミドルウェアの設定
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	// Todoリストの取得
	router.GET("/todos", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, title, completed, created_at FROM todos ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "データベースエラー"})
			return
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var todo Todo
			if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt); err != nil {
				log.Printf("Error scanning todo: %v", err)
				c.JSON(http.StatusInternalServerError, g.H{"error": "データの読み込みエラー"})
				return
			}
			todos = append(todos, todo)
		}
		c.JSON(http.StatusOK, todos)
	})

	// 単一Todoの作成
	router.POST("/todos", func(c *gin.Context) {
		var todo Todo
		if err := c.BindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, g.H{"error": "無効なリクエストデータ"})
			return
		}

		stmt, err := db.Prepare("INSERT INTO todos (title) VALUES (?)")
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "データベースエラー"})
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(todo.Title)
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Todo作成エラー"})
			return
		}

		id, err := res.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "ID取得エラー"})
			return
		}
		todo.ID = int(id)
		todo.Completed = false      // デフォルトでfalse
		todo.CreatedAt = time.Now() // 現時刻を設定

		c.JSON(http.StatusCreated, todo)
	})

	// Todoの更新
	router.PUT("/todos/:id", func(c *gin.Context) {
		id := c.Param("id")
		var todo Todo
		if err := c.BindJSON(&todo); err != nil {
			c.JSON(http.StatusBadRequest, g.H{"error": "無効なリクエストデータ"})
			return
		}

		stmt, err := db.Prepare("UPDATE todos SET title = ?, completed = ? WHERE id = ?")
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "データベースエラー"})
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(todo.Title, todo.Completed, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Todo更新エラー"})
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			c.JSON(http.StatusNotFound, g.H{"error": "Todoが見つからないか、更新されませんでした"})
			return
		}
		c.JSON(http.StatusOK, g.H{"message": "Todoが更新されました"})
	})

	// Todoの削除
	router.DELETE("/todos/:id", func(c *gin.Context) {
		id := c.Param("id")

		stmt, err := db.Prepare("DELETE FROM todos WHERE id = ?")
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "データベースエラー"})
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, g.H{"error": "Todo削除エラー"})
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			c.JSON(http.StatusNotFound, g.H{"error": "Todoが見つからないか、削除されませんでした"})
			return
		}
		c.JSON(http.StatusOK, g.H{"message": "Todoが削除されました"})
	})

	// 並行処理で複数のTodoを作成
	router.POST("/todos/batch", func(c *gin.Context) {
		var req BatchCreateRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, g.H{"error": "無効なリクエストデータ"})
			return
		}

		// バッファ付きチャネルをタイトル数と同じ容量で作成
		// 全てのゴルーチンが結果を送信するまでブロックしない
		resultChannel := make(chan CreateTodoResult, len(req.Titles))
		var wg sync.WaitGroup // WaitGroupを初期化

		// 各タイトルに対してゴルーチンを起動
		for _, title := range req.Titles {
			wg.Add(1) // 待機するゴルーチンの数を増やす
			go createTodoInGoroutine(db, title, resultChannel, &wg)
		}

		// 全てのゴルーチンが終了するのを待つ
		wg.Wait()
		close(resultChannel) // 全ての送信が終わったのでチャネルを閉じる

		// チャネルからすべての結果を収集
		results := []CreateTodoResult{}
		for result := range resultChannel { // チャネルが閉じるまで結果を受信する
			results = append(results, result)
		}

		c.JSON(http.StatusOK, g.H{"results": results})
	})

	return router
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

	// データベース接続確認
	if err = db.Ping(); err != nil {
		log.Fatalf("データベース接続確認エラー: %v", err)
	}
	log.Println("データベースに正常に接続しました！")

	router := setupRouter(db)

	// Ginサーバーを起動
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("サーバー起動エラー: %v", err)
	}
}
