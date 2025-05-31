import React, { useState, useEffect } from 'react';
import './App.css'; 

interface Todo {
    id: number;
    title: string;
    completed: boolean;
    created_at: string;
}

function App() {
    const [todos, setTodos] = useState<Todo[]>([])
    const [newTodoTitle, setNewTodoTitle] = useState<string>('')
    const apiUrl = 'http://localhost:8080/api/todos';

    useEffect(() => {
        fetchTodos();
    }, []);

    const fetchTodos = async () => {
        try {
            const response = await fetch(apiUrl);
            if (!response.ok) {
                throw new Error(`HTTPエラー ステータス: ${response.status}`)
            }
            const data: Todo[] = await response.json();
            setTodos(data);
        } catch (error) {
            console.error('Todoの取得に失敗しました:', error);
        }
    };

    const handleAddTodo = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newTodoTitle.trim()) return;

        try {
            const response = await fetch(apiUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ title: newTodoTitle }),
            });

            if (!response.ok) {
                throw new Error(`HTTPエラー! ステータス: ${response.status}`);
            }

            const addedTodo: Todo = await response.json();
            setTodos([addedTodo, ...todos]);
            setNewTodoTitle('')
        } catch (error) {
            console.error('Todoの追加に失敗しました:', error)
        }
    };

    return (
        <div className="App">
            <header className="App-header">
                <h1>Todoアプリ</h1>
                <form onSubmit={handleAddTodo}>
                    <input
                        type="text"
                        value={newTodoTitle}
                        onChange={(e) => setNewTodoTitle(e.target.value)}
                        placeholder="新しいTodoを追加"
                    />
                </form>
                <h2>Todoリスト</h2>
                <ul>
                    {todos.map((todo) => (
                        <li key={todo.id}>
                            {todo.title} ({todo.completed ? '完了' : '未完了'})
                            <br/>
                            <small>作成日: {new Date(todo.created_at).toLocaleString()}</small>
                        </li>                        
                    ))}
                </ul>
            </header>
        </div>
    )
}

export default App;