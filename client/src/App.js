import React, { useState, useEffect } from 'react';
import axios from 'axios';

function App() {
  const [todos, setTodos] = useState([]);
  const [task, setTask] = useState('');

  useEffect(() => {
    fetchTodos();
  }, []);

  const fetchTodos = async () => {
    try {
      const response = await axios.get('http://localhost:8090/api/todos');
      setTodos(response.data);
    } catch (error) {
      console.error('Failed to fetch todos:', error);
    }
  };

  const markTodoAsDone = async (id) => {
    try {
      await axios.put(`http://localhost:8090/api/todos/${id}`, { status: true });
      // Update the todos state with the updated todo item
      setTodos(prevTodos =>
        prevTodos.map(todo =>
          todo.id === id ? { ...todo, status: true } : todo
        )
      );
    } catch (error) {
      console.error('Failed to mark todo as done:', error);
    }
  };

  const deleteTodo = async (id) => {
    try {
      await axios.delete(`http://localhost:8090/api/todos/${id}`);
      fetchTodos();
    } catch (error) {
      console.error('Failed to delete todo:', error);
    }
  };

  const addTodo = async (e) => {
    e.preventDefault();
    if (!task.trim()) return;
    try {
      await axios.post('http://localhost:8090/api/todos', { task, status: false });
      setTask('');
      fetchTodos();
    } catch (error) {
      console.error('Failed to add todo:', error);
    }
  };

  return (
    <div className='middle'>
      <div className='title'>
        <h1>Todo List</h1>
      </div>
      <form onSubmit={addTodo}>
        <input type="text" value={task} onChange={(e) => setTask(e.target.value)} style={{ width: '' }} />
        <button type="submit">Add Todo</button>
      </form>
      <ul>
        {todos.map((todo) => (
          <li key={todo.id}>
            {todo.task} - {todo.status ? 'true' : 'false'}
            {!todo.status && <button onClick={() => markTodoAsDone(todo.id)}>Mark as Done</button>}
            <button onClick={() => deleteTodo(todo.id)}>Delete</button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;
