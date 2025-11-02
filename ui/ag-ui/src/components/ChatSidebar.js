import React, { useState, useRef, useEffect } from 'react';
import './ChatSidebar.css';

function ChatSidebar({ onQuerySubmit }) {
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState([
    { 
      role: 'assistant', 
      content: '👋 Hi! I\'m your AI assistant. Ask me about:\n• Error rates and trends\n• Recent logs\n• System health\n• Specific error messages',
      timestamp: new Date()
    }
  ]);
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const userMessage = { role: 'user', content: input, timestamp: new Date() };
    setMessages(prev => [...prev, userMessage]);
    setLoading(true);

    const query = input;
    setInput('');

    try {
      const response = await fetch('/mcp/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query })
      });

      const data = await response.json();
      
      // Generate natural language response from the data
      let naturalResponse = data.response || 'I couldn\'t process that request.';
      
      // If there's data, enhance the response
      if (data.data && Array.isArray(data.data) && data.data.length > 0) {
        const logs = data.data;
        const errorLogs = logs.filter(l => l.level === 'ERROR');
        const warnLogs = logs.filter(l => l.level === 'WARN');
        
        naturalResponse = `📊 I found **${logs.length} logs**:\n\n`;
        
        if (errorLogs.length > 0) {
          naturalResponse += `🔴 **${errorLogs.length} Errors:**\n`;
          errorLogs.slice(0, 3).forEach(log => {
            naturalResponse += `  • ${log.message}\n`;
          });
          if (errorLogs.length > 3) {
            naturalResponse += `  ... and ${errorLogs.length - 3} more\n`;
          }
          naturalResponse += '\n';
        }
        
        if (warnLogs.length > 0) {
          naturalResponse += `⚠️ **${warnLogs.length} Warnings:**\n`;
          warnLogs.slice(0, 2).forEach(log => {
            naturalResponse += `  • ${log.message}\n`;
          });
          if (warnLogs.length > 2) {
            naturalResponse += `  ... and ${warnLogs.length - 2} more\n`;
          }
        }
        
        naturalResponse += '\n💡 **Tip:** Click on the logs panel to see full details.';
      } else if (data.data && typeof data.data === 'object' && !Array.isArray(data.data)) {
        // Handle metrics/stats data
        const metrics = data.data;
        if (metrics.error_rate !== undefined) {
          naturalResponse = `📈 **Error Rate Analysis:**\n\n`;
          naturalResponse += `Current error rate: **${(metrics.error_rate * 100).toFixed(2)}%**\n\n`;
          
          if (metrics.error_rate > 0.15) {
            naturalResponse += '🔴 **Alert:** Error rate is higher than normal!';
          } else if (metrics.error_rate > 0.05) {
            naturalResponse += '⚠️ **Warning:** Error rate is slightly elevated.';
          } else {
            naturalResponse += '✅ **Good:** Error rate is within normal range.';
          }
        }
      }
      
      setMessages(prev => [...prev, { 
        role: 'assistant', 
        content: naturalResponse,
        data: data.data,
        timestamp: new Date()
      }]);
      
      // Pass the data to parent component (for LogPanel)
      onQuerySubmit(query, data.data);
    } catch (error) {
      setMessages(prev => [...prev, { 
        role: 'assistant', 
        content: '❌ Error: Could not reach the server. Please check your connection.',
        timestamp: new Date()
      }]);
    } finally {
      setLoading(false);
    }
  };

  const formatMessage = (content) => {
    return content.split('\n').map((line, i) => (
      <span key={i}>
        {line.includes('**') ? (
          line.split('**').map((part, j) => 
            j % 2 === 1 ? <strong key={j}>{part}</strong> : part
          )
        ) : line}
        <br />
      </span>
    ));
  };

  const suggestedQueries = [
    "What are the recent errors?",
    "Show me error rate trends",
    "Any warnings in the last hour?",
    "Show me nginx logs"
  ];

  return (
    <div className="chat-sidebar">
      <div className="chat-header">
        <h3>💬 AI Assistant</h3>
        <span className="status-indicator">🟢 Online</span>
      </div>
      
      <div className="chat-messages">
        {messages.map((msg, idx) => (
          <div key={idx} className={`message ${msg.role}`}>
            <div className="message-header">
              <span className="message-role">
                {msg.role === 'user' ? '👤 You' : '🤖 AI'}
              </span>
              <span className="message-time">
                {msg.timestamp.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
              </span>
            </div>
            <div className="message-content">
              {formatMessage(msg.content)}
            </div>
          </div>
        ))}
        {loading && (
          <div className="message assistant loading">
            <div className="message-header">
              <span className="message-role">🤖 AI</span>
            </div>
            <div className="message-content">
              <div className="typing-indicator">
                <span></span><span></span><span></span>
              </div>
              Analyzing logs...
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {messages.length === 1 && !loading && (
        <div className="suggested-queries">
          <p className="suggested-title">Try asking:</p>
          {suggestedQueries.map((query, idx) => (
            <button
              key={idx}
              className="suggested-query"
              onClick={() => {
                setInput(query);
              }}
            >
              {query}
            </button>
          ))}
        </div>
      )}
      
      <form className="chat-input" onSubmit={handleSubmit}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about your logs..."
          disabled={loading}
        />
        <button type="submit" disabled={loading || !input.trim()}>
          {loading ? '⏳' : '📤'}
        </button>
      </form>
    </div>
  );
}

export default ChatSidebar;
